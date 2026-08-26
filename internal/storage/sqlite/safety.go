package sqlite

import "context"

func EnsureSafetyTables(ctx context.Context, s *Store) error {
	_, err := s.DB.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS safety_findings(id TEXT PRIMARY KEY,tenant_id TEXT NOT NULL,lot_id TEXT NOT NULL,code TEXT NOT NULL,level TEXT NOT NULL,measured REAL NOT NULL,limit_value REAL NOT NULL,observed_at TEXT NOT NULL,resolved_at TEXT);CREATE INDEX IF NOT EXISTS idx_safety_lot ON safety_findings(tenant_id,lot_id,level)`)
	return err
}
func (s *Store) OpenFindingCount(ctx context.Context, tenant, lot string) (int, error) {
	var n int
	err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM safety_findings WHERE tenant_id=? AND lot_id=? AND resolved_at IS NULL`, tenant, lot).Scan(&n)
	return n, err
}
