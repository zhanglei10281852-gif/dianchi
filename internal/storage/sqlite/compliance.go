package sqlite

import (
	"context"
	"fmt"
)

func EnsureComplianceTables(ctx context.Context, s *Store) error {
	_, err := s.DB.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS compliance_certificates(id TEXT PRIMARY KEY,tenant_id TEXT NOT NULL,lot_id TEXT NOT NULL,serial TEXT NOT NULL UNIQUE,status TEXT NOT NULL);CREATE TABLE IF NOT EXISTS compliance_reviews(id TEXT PRIMARY KEY,certificate_id TEXT NOT NULL REFERENCES compliance_certificates(id),reviewer_id TEXT NOT NULL,decision TEXT NOT NULL,notes TEXT NOT NULL)`)
	return err
}
func (s *Store) ComplianceStatus(ctx context.Context, id, tenant string) (string, error) {
	var st string
	err := s.DB.QueryRowContext(ctx, `SELECT status FROM compliance_certificates WHERE id=? AND tenant_id=?`, id, tenant).Scan(&st)
	return st, err
}
func (s *Store) CheckConstraint(ctx context.Context) error {
	var name string
	err := s.DB.QueryRowContext(ctx, `SELECT name FROM sqlite_master WHERE type='table' AND name='compliance_certificates'`).Scan(&name)
	if err != nil {
		return err
	}
	if name == "" {
		return fmt.Errorf("missing compliance table")
	}
	return nil
}
