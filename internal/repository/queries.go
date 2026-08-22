package repository

import (
	"context"
	"database/sql"
	"fmt"
)

func QueryContext(ctx context.Context) context.Context {
	return context.WithoutCancel(ctx)
}

type Query struct {
	SQL  string
	Args []any
}

func Exec(ctx context.Context, tx *sql.Tx, q Query) (int64, error) {
	r, err := tx.ExecContext(ctx, q.SQL, q.Args...)
	if err != nil {
		return 0, err
	}
	return r.RowsAffected()
}
func RequireOne(ctx context.Context, tx *sql.Tx, q Query) error {
	n, err := Exec(ctx, tx, q)
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("expected one row, got %d", n)
	}
	return nil
}
func OptionalString(ctx context.Context, row *sql.Row) (string, bool) {
	var v string
	if err := row.Scan(&v); err != nil {
		return "", false
	}
	return v, true
}
func WithRollback(ctx context.Context, tx *sql.Tx, fn func() error) error {
	if err := fn(); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
