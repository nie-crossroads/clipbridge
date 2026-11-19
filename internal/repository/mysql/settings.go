package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

type RuntimeSettings struct {
	WebRate          int `json:"web_rate"`
	WebWindowSeconds int `json:"web_window_seconds"`
	ParseConcurrency int `json:"parse_concurrency"`
	RequestDays      int `json:"request_days"`
	AuditDays        int `json:"audit_days"`
	CacheItems       int `json:"cache_items"`
	CacheMiB         int `json:"cache_mib"`
	CacheTTLSeconds  int `json:"cache_ttl_seconds"`
}

func (s *Store) RuntimeSettings(ctx context.Context, settings *RuntimeSettings) (bool, error) {
	var value []byte
	err := s.DB.QueryRowContext(ctx, "SELECT setting_value FROM cb_system_settings WHERE setting_key='runtime'").Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, json.Unmarshal(value, settings)
}

func (s *Store) SaveRuntimeSettings(ctx context.Context, settings RuntimeSettings, actor uint64, requestID string) error {
	data, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO cb_system_settings(setting_key,setting_value,updated_by)
 VALUES('runtime',?,?) ON DUPLICATE KEY UPDATE setting_value=?,updated_by=?,version=version+1`, string(data), actor, string(data), actor); err != nil {
		return err
	}
	if err := audit(ctx, tx, actor, "update_settings", "settings", "runtime", requestID, string(data)); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) ClearRuntimeData(ctx context.Context, action string, actor uint64, requestID string) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	switch action {
	case "clear_requests":
		if _, err := tx.ExecContext(ctx, "DELETE FROM cb_request_logs"); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "DELETE FROM cb_parse_records"); err != nil {
			return err
		}
	case "clear_cache":
	default:
		return errors.New("unknown clear action")
	}
	if err := audit(ctx, tx, actor, action, "settings", action, requestID, nil); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) ExpireLogs(ctx context.Context, requestsBefore, auditBefore time.Time) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "DELETE FROM cb_request_logs WHERE created_at<? ORDER BY created_at,id LIMIT 500", requestsBefore); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM cb_parse_records WHERE created_at<? ORDER BY created_at,id LIMIT 500", requestsBefore); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM cb_audit_logs WHERE created_at<? ORDER BY created_at,id LIMIT 500", auditBefore); err != nil {
		return err
	}
	return tx.Commit()
}
