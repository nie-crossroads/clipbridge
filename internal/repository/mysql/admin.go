package mysql

import (
	"context"
	"database/sql"
	"time"
)

type Admin struct {
	ID           uint64
	Name         string
	PasswordHash string
}

func (s *Store) FindAdmin(ctx context.Context, username string) (Admin, error) {
	var admin Admin
	err := s.DB.QueryRowContext(ctx, `SELECT id, COALESCE(NULLIF(display_name,''),username), password_hash
 FROM cb_admin_users WHERE username=? AND status='active'`, username).Scan(&admin.ID, &admin.Name, &admin.PasswordHash)
	return admin, err
}

func (s *Store) BootstrapAdmin(ctx context.Context, username, hash, requestID string) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO cb_admin_users(username,password_hash) VALUES (?,?)`, username, hash); err != nil {
		return err
	}
	// LAST_INSERT_ID stays within the transaction connection and preserves unsigned IDs.
	if _, err := tx.ExecContext(ctx, `INSERT INTO cb_audit_logs(action,target_type,target_id,request_id)
 VALUES ('bootstrap_admin','admin',CAST(LAST_INSERT_ID() AS CHAR),?)`, requestID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) ResetAdminPassword(ctx context.Context, admin Admin, hash, requestID string) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE cb_admin_users SET password_hash=?
 WHERE id=? AND status='active' AND password_hash=?`, hash, admin.ID, admin.PasswordHash)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return sql.ErrNoRows
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM cb_admin_sessions WHERE admin_id=?", admin.ID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO cb_audit_logs(action,target_type,target_id,request_id)
 VALUES ('reset_admin_password','admin',CAST(? AS CHAR),?)`, admin.ID, requestID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) CreateSession(ctx context.Context, adminID uint64, tokenHash, csrfHash []byte, expiry time.Time) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "DELETE FROM cb_admin_sessions WHERE expires_at <= UTC_TIMESTAMP(3) ORDER BY expires_at LIMIT 100"); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO cb_admin_sessions(admin_id,token_hash,csrf_hash,expires_at)
 VALUES (?,?,?,?)`, adminID, tokenHash, csrfHash, expiry); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "UPDATE cb_admin_users SET last_login_at=UTC_TIMESTAMP(3) WHERE id=?", adminID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) Session(ctx context.Context, tokenHash []byte) (Admin, []byte, error) {
	var admin Admin
	var csrfHash []byte
	err := s.DB.QueryRowContext(ctx, `SELECT u.id, COALESCE(NULLIF(u.display_name,''),u.username), s.csrf_hash
 FROM cb_admin_sessions s JOIN cb_admin_users u ON u.id=s.admin_id
 WHERE s.token_hash=? AND s.expires_at>UTC_TIMESTAMP(3) AND u.status='active'`, tokenHash).
		Scan(&admin.ID, &admin.Name, &csrfHash)
	return admin, csrfHash, err
}

func (s *Store) DeleteSession(ctx context.Context, tokenHash []byte) error {
	_, err := s.DB.ExecContext(ctx, "DELETE FROM cb_admin_sessions WHERE token_hash=?", tokenHash)
	return err
}

func audit(ctx context.Context, tx *sql.Tx, actorID uint64, action, target, id, requestID string, details any) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO cb_audit_logs(actor_admin_id,action,target_type,target_id,request_id,details_json)
 VALUES (?,?,?,?,?,?)`, actorID, action, target, id, requestID, details)
	return err
}
