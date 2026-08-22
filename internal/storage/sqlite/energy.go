package sqlite

import (
	"context"
	"time"
)

func EnsureEnergyTables(ctx context.Context, s *Store) error {
	_, err := s.DB.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS energy_readings(id TEXT PRIMARY KEY,tenant_id TEXT NOT NULL,lot_id TEXT NOT NULL,voltage REAL NOT NULL,current REAL NOT NULL,temperature REAL NOT NULL,recorded_at TEXT NOT NULL);CREATE INDEX IF NOT EXISTS idx_energy_lot_time ON energy_readings(tenant_id,lot_id,recorded_at)`)
	return err
}
func (s *Store) EnergyCount(ctx context.Context, tenant, lot string) (int, error) {
	var n int
	err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM energy_readings WHERE tenant_id=? AND lot_id=?`, tenant, lot).Scan(&n)
	return n, err
}
func (s *Store) EnergySince(ctx context.Context, tenant, lot string, since time.Time) (int, error) {
	var n int
	err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM energy_readings WHERE tenant_id=? AND lot_id=? AND recorded_at>=?`, tenant, lot, since.Format(time.RFC3339Nano)).Scan(&n)
	return n, err
}
