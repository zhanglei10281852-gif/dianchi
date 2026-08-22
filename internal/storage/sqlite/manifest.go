package sqlite

import (
	"context"
	"fmt"
)

func EnsureManifestTables(ctx context.Context, s *Store) error {
	_, err := s.DB.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS manifests(id TEXT PRIMARY KEY,tenant_id TEXT NOT NULL,origin TEXT NOT NULL,destination TEXT NOT NULL,status TEXT NOT NULL,total_weight INTEGER NOT NULL);CREATE TABLE IF NOT EXISTS manifest_items(id TEXT PRIMARY KEY,manifest_id TEXT NOT NULL REFERENCES manifests(id),lot_id TEXT NOT NULL,tenant_id TEXT NOT NULL,weight INTEGER NOT NULL)`)
	return err
}
func (s *Store) Manifest(ctx context.Context, id, tenant string) (string, error) {
	var status string
	err := s.DB.QueryRowContext(ctx, `SELECT status FROM manifests WHERE id=? AND tenant_id=?`, id, tenant).Scan(&status)
	return status, err
}
func (s *Store) ManifestCount(ctx context.Context, tenant string) (int, error) {
	var n int
	err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM manifests WHERE tenant_id=?`, tenant).Scan(&n)
	return n, err
}
func (s *Store) VerifyTable(ctx context.Context, name string) error {
	var out string
	err := s.DB.QueryRowContext(ctx, `SELECT name FROM sqlite_master WHERE type='table' AND name=?`, name).Scan(&out)
	if err != nil {
		return err
	}
	if out != name {
		return fmt.Errorf("table mismatch")
	}
	return nil
}
