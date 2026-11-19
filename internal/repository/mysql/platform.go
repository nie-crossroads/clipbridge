package mysql

import (
	"context"
	"encoding/json"
	"time"

	"github.com/nie-crossroads/clipbridge/internal/parser"
)

func (s *Store) Platforms(ctx context.Context, items []parser.Platform) error {
	rows, err := s.DB.QueryContext(ctx, "SELECT platform, enabled, last_test_status, last_tested_at, config_json, config_version FROM cb_platform_configs")
	if err != nil {
		return err
	}
	defer rows.Close()
	positions := make(map[string]int, len(items))
	for index, item := range items {
		positions[item.Code] = index
	}
	for rows.Next() {
		var code string
		var enabled bool
		var status *string
		var testedAt *time.Time
		var raw []byte
		var version uint64
		if err := rows.Scan(&code, &enabled, &status, &testedAt, &raw, &version); err != nil {
			return err
		}
		if index, ok := positions[code]; ok {
			items[index].Enabled = enabled && items[index].Implemented
			items[index].LastTestStatus, items[index].LastTestedAt = status, testedAt
			items[index].ConfigVersion = version
			var config PlatformPatch
			if len(raw) > 0 && json.Unmarshal(raw, &config) == nil && config.TimeoutSeconds != nil {
				items[index].TimeoutSeconds = *config.TimeoutSeconds
			}
		}
	}
	return rows.Err()
}

type PlatformPatch struct {
	Enabled        *bool `json:"enabled,omitempty"`
	TimeoutSeconds *int  `json:"timeout_seconds,omitempty"`
}

func (s *Store) SetPlatform(ctx context.Context, code string, change PlatformPatch, actorID uint64, requestID string) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO cb_platform_configs(platform,enabled,config_json,updated_by)
 VALUES (?,COALESCE(?,0),IF(? IS NULL,NULL,JSON_OBJECT('timeout_seconds',?)),?)
 ON DUPLICATE KEY UPDATE enabled=COALESCE(?,enabled),
 config_json=IF(? IS NULL,config_json,JSON_SET(COALESCE(config_json,JSON_OBJECT()),'$.timeout_seconds',?)),
 updated_by=?,config_version=config_version+1`, code, change.Enabled, change.TimeoutSeconds, change.TimeoutSeconds, actorID,
		change.Enabled, change.TimeoutSeconds, change.TimeoutSeconds, actorID); err != nil {
		return err
	}
	details, err := json.Marshal(change)
	if err != nil {
		return err
	}
	if err := audit(ctx, tx, actorID, "update_platform", "platform", code, requestID, string(details)); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) SavePlatformTest(ctx context.Context, code, status string, errorCode *string) error {
	_, err := s.DB.ExecContext(ctx, `INSERT INTO cb_platform_configs(platform,last_test_status,last_test_error_code,last_tested_at)
 VALUES (?,?,?,UTC_TIMESTAMP(3)) ON DUPLICATE KEY UPDATE last_test_status=?,last_test_error_code=?,last_tested_at=UTC_TIMESTAMP(3)`,
		code, status, errorCode, status, errorCode)
	return err
}
