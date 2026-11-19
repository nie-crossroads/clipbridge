package mysql

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"
)

type KeySettings struct {
	Name        string     `json:"name"`
	Channel     string     `json:"channel"`
	Scopes      []string   `json:"scopes"`
	Platforms   []string   `json:"platform_allowlist"`
	Rate        uint32     `json:"rate_limit_per_minute"`
	Daily       uint64     `json:"daily_parse_limit"`
	Concurrency uint32     `json:"max_concurrency"`
	ExpiresAt   *time.Time `json:"expires_at"`
}

type KeySummary struct {
	KeySettings
	ID         string     `json:"id"`
	Hint       string     `json:"key_hint"`
	Status     string     `json:"status"`
	LastUsedAt *time.Time `json:"last_used_at"`
}

func (s *Store) Keys(ctx context.Context) ([]KeySummary, error) {
	result := []KeySummary{}
	rows, err := s.DB.QueryContext(ctx, `SELECT CAST(id AS CHAR),name,channel,key_hint,
 CASE WHEN revoked_at IS NOT NULL THEN 'revoked' WHEN status='active' AND expires_at <= UTC_TIMESTAMP(3) THEN 'expired' ELSE status END,
 expires_at,last_used_at,scopes,platform_allowlist,rate_limit_per_minute,daily_parse_limit,max_concurrency
 FROM cb_api_keys ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var key KeySummary
		var scopes, platforms []byte
		if err := rows.Scan(&key.ID, &key.Name, &key.Channel, &key.Hint, &key.Status, &key.ExpiresAt, &key.LastUsedAt,
			&scopes, &platforms, &key.Rate, &key.Daily, &key.Concurrency); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(scopes, &key.Scopes); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(platforms, &key.Platforms); err != nil {
			return nil, err
		}
		result = append(result, key)
	}
	return result, rows.Err()
}

func (s *Store) CreateKey(ctx context.Context, settings KeySettings, hash []byte, hint string, actor uint64, requestID string) error {
	scopes, err := json.Marshal(settings.Scopes)
	if err != nil {
		return err
	}
	platforms, err := json.Marshal(settings.Platforms)
	if err != nil {
		return err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO cb_api_keys(name,channel,key_hash,key_hint,scopes,platform_allowlist,
 rate_limit_per_minute,daily_parse_limit,max_concurrency,expires_at,created_by) VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
		settings.Name, settings.Channel, hash, hint, string(scopes), string(platforms), settings.Rate, settings.Daily, settings.Concurrency, settings.ExpiresAt, actor); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO cb_audit_logs(actor_admin_id,action,target_type,target_id,request_id)
 VALUES(?,'create_key','key',CAST(LAST_INSERT_ID() AS CHAR),?)`, actor, requestID); err != nil {
		return err
	}
	return tx.Commit()
}

var ErrKeyState = errors.New("key cannot change state")

type KeyAction struct {
	Action      string `json:"action"`
	ConfirmName string `json:"confirm_name"`
}

func (s *Store) ChangeKey(ctx context.Context, id uint64, change KeyAction, actor uint64, requestID string, generate func(string) (string, []byte, string)) (string, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	var name, channel, status string
	var expired, revoked bool
	err = tx.QueryRowContext(ctx, `SELECT name,channel,status,COALESCE(expires_at<=UTC_TIMESTAMP(3),0), revoked_at IS NOT NULL
 FROM cb_api_keys WHERE id=? FOR UPDATE`, id).Scan(&name, &channel, &status, &expired, &revoked)
	if err != nil {
		return "", err
	}
	if revoked || status == "revoked" {
		return "", ErrKeyState
	}
	var secret string
	switch change.Action {
	case "rotate":
		if expired {
			return "", ErrKeyState
		}
		value, hash, hint := generate(channel)
		if _, err := tx.ExecContext(ctx, "UPDATE cb_api_keys SET key_hash=?,key_hint=?,version=version+1 WHERE id=?", hash, hint, id); err != nil {
			return "", err
		}
		secret = value
	case "disable":
		if status != "active" || expired {
			return "", ErrKeyState
		}
		_, err = tx.ExecContext(ctx, "UPDATE cb_api_keys SET status='disabled',version=version+1 WHERE id=?", id)
	case "enable":
		if status != "disabled" || expired {
			return "", ErrKeyState
		}
		_, err = tx.ExecContext(ctx, "UPDATE cb_api_keys SET status='active',version=version+1 WHERE id=?", id)
	case "revoke":
		if change.ConfirmName != name {
			return "", ErrKeyState
		}
		_, err = tx.ExecContext(ctx, "UPDATE cb_api_keys SET status='revoked',revoked_at=UTC_TIMESTAMP(3),version=version+1 WHERE id=?", id)
	default:
		return "", ErrKeyState
	}
	if err != nil {
		return "", err
	}
	if err := audit(ctx, tx, actor, change.Action+"_key", "key", strconv.FormatUint(id, 10), requestID, nil); err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return secret, nil
}

type KeyUsage struct {
	Date  string `json:"date"`
	Used  uint64 `json:"used_units"`
	Limit uint64 `json:"daily_parse_limit"`
}

func (s *Store) KeyUsage(ctx context.Context, id uint64, date string) (KeyUsage, error) {
	result := KeyUsage{Date: date}
	err := s.DB.QueryRowContext(ctx, `SELECT COALESCE(q.used_units,0),k.daily_parse_limit
 FROM cb_api_keys k LEFT JOIN cb_quota_daily q ON q.api_key_id=k.id AND q.quota_date=? WHERE k.id=?`, date, id).
		Scan(&result.Used, &result.Limit)
	return result, err
}
