package sqlite

import (
	"context"
	"database/sql"
)

func (s *Store) Ping(ctx context.Context) error { return s.DB.PingContext(ctx) }
func (s *Store) TableCount(ctx context.Context) (int, error) {
	var n int
	err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='table'`).Scan(&n)
	return n, err
}
func (s *Store) ForeignKeys(ctx context.Context) (bool, error) {
	var n int
	err := s.DB.QueryRowContext(ctx, `PRAGMA foreign_keys`).Scan(&n)
	return n == 1, err
}
func (s *Store) Begin(ctx context.Context) (*sql.Tx, error) { return s.DB.BeginTx(ctx, nil) }
