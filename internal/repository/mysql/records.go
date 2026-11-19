package mysql

import (
	"context"
	"encoding/json"
	"time"
)

type Record struct {
	ID         string    `json:"request_id"`
	Channel    string    `json:"channel"`
	AdminID    *uint64   `json:"-"`
	KeyID      *uint64   `json:"-"`
	Platform   string    `json:"platform"`
	ItemID     string    `json:"item_id"`
	InputHash  []byte    `json:"-"`
	MediaType  string    `json:"media_type"`
	ErrorCode  *string   `json:"error_code"`
	Status     int       `json:"http_status"`
	DurationMS int64     `json:"duration_ms"`
	CreatedAt  time.Time `json:"created_at"`
	Accepted   int       `json:"-"`
	KeyHint    *string   `json:"key_hint"`
	SourceIP   *string   `json:"source_ip"`
	CacheHit   *bool     `json:"cache_hit"`
	Operation  string    `json:"operation"`
}

func (s *Store) SaveRecord(ctx context.Context, record Record) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	success := 0
	status := "failed"
	if record.ErrorCode == nil {
		success, status = 1, "success"
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO cb_request_logs(request_id,channel,admin_id,api_key_id,operation,accepted_units,
 success_items,failed_items,http_status,error_code,duration_ms,source_ip_redacted) VALUES (?,?,?,?,'parse',?,?,?,?,?,?,?)`,
		record.ID, record.Channel, record.AdminID, record.KeyID, record.Accepted, success, 1-success, record.Status, record.ErrorCode, record.DurationMS, record.SourceIP); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO cb_parse_records(request_id,platform,item_id,input_hash,media_type,status,error_code,duration_ms,cache_hit)
 VALUES (?,NULLIF(?,''),NULLIF(?,''),?,NULLIF(?,''),?,?,?,COALESCE(?,0))`, record.ID, record.Platform, record.ItemID, record.InputHash,
		record.MediaType, status, record.ErrorCode, record.DurationMS, record.CacheHit); err != nil {
		return err
	}
	return tx.Commit()
}

type RecordFilter struct {
	Channel, Platform, ErrorCode, Status, RequestID string
	KeyID                                           *uint64
	Public                                          bool
	From, To                                        *time.Time
}

func (s *Store) Records(ctx context.Context, filter RecordFilter, offset int) ([]Record, error) {
	result := []Record{}
	rows, err := s.DB.QueryContext(ctx, `SELECT r.request_id, r.channel, COALESCE(p.platform,''), COALESCE(p.item_id,''),
 COALESCE(p.media_type,''), r.error_code, COALESCE(r.http_status,0), r.duration_ms, r.created_at,
 k.key_hint,r.source_ip_redacted,IF(r.accepted_units>0,p.cache_hit,NULL),r.operation
 FROM cb_request_logs r LEFT JOIN cb_parse_records p ON r.request_id=p.request_id AND p.item_index=0
 LEFT JOIN cb_api_keys k ON k.id=r.api_key_id
 WHERE (?='' OR r.channel=?) AND (?='' OR p.platform=?) AND (?='' OR r.error_code=?)
 AND (? IS NULL OR r.created_at>=?) AND (? IS NULL OR r.created_at<?)
 AND (? IS NULL OR r.api_key_id=?) AND (NOT ? OR (r.channel='web' AND r.api_key_id IS NULL))
 AND (?='' OR LOCATE(?,r.request_id)>0)
 AND (?='' OR (?='success' AND r.http_status BETWEEN 200 AND 299 AND r.error_code IS NULL)
 OR (?='failed' AND (r.http_status>=400 OR r.error_code IS NOT NULL)))
 ORDER BY r.created_at DESC,r.id DESC LIMIT 26 OFFSET ?`, filter.Channel, filter.Channel, filter.Platform, filter.Platform, filter.ErrorCode, filter.ErrorCode,
		filter.From, filter.From, filter.To, filter.To, filter.KeyID, filter.KeyID, filter.Public, filter.RequestID, filter.RequestID,
		filter.Status, filter.Status, filter.Status, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var item Record
		if err := rows.Scan(&item.ID, &item.Channel, &item.Platform, &item.ItemID, &item.MediaType, &item.ErrorCode, &item.Status, &item.DurationMS, &item.CreatedAt,
			&item.KeyHint, &item.SourceIP, &item.CacheHit, &item.Operation); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

type Audit struct {
	ID          string          `json:"id"`
	Action      string          `json:"action"`
	Target      string          `json:"target_type"`
	TargetID    string          `json:"target_id"`
	Actor       string          `json:"actor"`
	Details     json.RawMessage `json:"details"`
	CreatedAt   time.Time       `json:"created_at"`
	RequestID   *string         `json:"request_id"`
	TargetLabel string          `json:"target_label"`
}

type AuditFilter struct {
	Action, Actor, Target, Object, RequestID string
	From, To                                 *time.Time
}

func (s *Store) Audits(ctx context.Context, filter AuditFilter, offset int) ([]Audit, error) {
	result := []Audit{}
	rows, err := s.DB.QueryContext(ctx, `SELECT CAST(a.id AS CHAR),a.action,a.target_type,a.target_id,
 COALESCE(NULLIF(u.display_name,''),u.username,'系统'),a.details_json,a.created_at,a.request_id,
 COALESCE(k.key_hint,o.username,a.target_id)
 FROM cb_audit_logs a LEFT JOIN cb_admin_users u ON u.id=a.actor_admin_id
 LEFT JOIN cb_api_keys k ON a.target_type='key' AND CAST(k.id AS CHAR)=a.target_id
 LEFT JOIN cb_admin_users o ON a.target_type='admin' AND CAST(o.id AS CHAR)=a.target_id
 WHERE (?='' OR a.action=?) AND (?='' OR COALESCE(NULLIF(u.display_name,''),u.username,'系统')=?) AND (?='' OR a.target_type=?)
 AND (?='' OR LOCATE(?,COALESCE(a.request_id,''))>0)
 AND (?='' OR LOCATE(?,a.target_id)>0 OR LOCATE(?,COALESCE(k.name,o.username,''))>0 OR LOCATE(?,COALESCE(k.key_hint,''))>0)
 AND (? IS NULL OR a.created_at>=?) AND (? IS NULL OR a.created_at<?)
 ORDER BY a.created_at DESC,a.id DESC LIMIT 26 OFFSET ?`, filter.Action, filter.Action, filter.Actor, filter.Actor, filter.Target, filter.Target,
		filter.RequestID, filter.RequestID, filter.Object, filter.Object, filter.Object, filter.Object, filter.From, filter.From, filter.To, filter.To, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var item Audit
		var details []byte
		if err := rows.Scan(&item.ID, &item.Action, &item.Target, &item.TargetID, &item.Actor, &details, &item.CreatedAt, &item.RequestID, &item.TargetLabel); err != nil {
			return nil, err
		}
		// Only fields written and explicitly allowed by this application are exposed.
		if item.Action == "update_platform" {
			var update PlatformPatch
			if json.Unmarshal(details, &update) == nil {
				item.Details, _ = json.Marshal(update)
			}
		}
		if item.Action == "update_settings" {
			var update RuntimeSettings
			if json.Unmarshal(details, &update) == nil {
				item.Details, _ = json.Marshal(update)
			}
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

type Stats struct {
	Accepted    int64   `json:"accepted"`
	Success     int64   `json:"success"`
	ProxyBytes  int64   `json:"proxy_bytes"`
	DBOpen      int     `json:"db_open"`
	DBMax       int     `json:"db_max"`
	DBInUse     int     `json:"db_in_use"`
	DBIdle      int     `json:"db_idle"`
	DBWaitCount int64   `json:"db_wait_count"`
	HeapBytes   uint64  `json:"heap_bytes"`
	RSSBytes    *uint64 `json:"rss_bytes"`
}

func (s *Store) Stats(ctx context.Context, since time.Time) (Stats, error) {
	var result Stats
	err := s.DB.QueryRowContext(ctx, `SELECT COALESCE(SUM(accepted_units),0), COALESCE(SUM(success_items),0),
 COALESCE(SUM(proxy_bytes_sent),0) FROM cb_request_logs WHERE created_at >= ?`, since).
		Scan(&result.Accepted, &result.Success, &result.ProxyBytes)
	pool := s.DB.Stats()
	result.DBOpen, result.DBMax = pool.OpenConnections, pool.MaxOpenConnections
	result.DBInUse, result.DBIdle, result.DBWaitCount = pool.InUse, pool.Idle, pool.WaitCount
	return result, err
}

func (s *Store) AuditActors(ctx context.Context) ([]string, error) {
	items := []string{}
	rows, err := s.DB.QueryContext(ctx, `SELECT DISTINCT COALESCE(NULLIF(u.display_name,''),u.username,'系统') AS actor
 FROM cb_audit_logs a LEFT JOIN cb_admin_users u ON u.id=a.actor_admin_id ORDER BY actor`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var actor string
		if err := rows.Scan(&actor); err != nil {
			return nil, err
		}
		items = append(items, actor)
	}
	return items, rows.Err()
}
