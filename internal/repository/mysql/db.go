package mysql

import (
	"context"
	"database/sql"
	"time"

	driver "github.com/go-sql-driver/mysql"
)

type Store struct{ DB *sql.DB }

func Open(cfg *driver.Config) (*Store, error) {
	connector, err := driver.NewConnector(cfg)
	if err != nil {
		return nil, err
	}
	db := sql.OpenDB(connector)
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(5 * time.Minute)
	return &Store{DB: db}, nil
}

func (s *Store) Ready(ctx context.Context) error {
	// Check the schema as well as the connection.
	var present int
	var hasSourceColumn bool
	err := s.DB.QueryRowContext(ctx, `SELECT (SELECT COUNT(*) FROM information_schema.tables
 WHERE table_schema = DATABASE() AND table_name IN ('cb_admin_users','cb_admin_sessions','cb_platform_configs',
 'cb_request_logs','cb_parse_records','cb_audit_logs','cb_api_keys','cb_quota_daily','cb_system_settings')),
 EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='cb_request_logs'
 AND column_name='source_ip_redacted')`).Scan(&present, &hasSourceColumn)
	if err == nil && (present != 9 || !hasSourceColumn) {
		return sql.ErrNoRows
	}
	return err
}
