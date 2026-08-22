package sqlite

import (
	"context"
	"database/sql"
)

func (s *Store) Vacuum(ctx context.Context) error {
	_, err := s.DB.ExecContext(ctx, `VACUUM`)
	return err
}
func (s *Store) Backup(ctx context.Context, dst *sql.DB) error {
	rows, err := s.DB.QueryContext(ctx, `SELECT name,sql FROM sqlite_master WHERE sql IS NOT NULL`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var name, ddl string
		if err := rows.Scan(&name, &ddl); err != nil {
			return err
		}
		if _, err := dst.ExecContext(ctx, ddl); err != nil {
			return err
		}
	}
	return rows.Err()
}
func (s *Store) CheckIntegrity(ctx context.Context) (string, error) {
	var out string
	err := s.DB.QueryRowContext(ctx, `PRAGMA integrity_check`).Scan(&out)
	return out, err
}
