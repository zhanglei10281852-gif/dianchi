package sqlite

import (
	"context"
	"database/sql"
	"time"
)

func BeginImmediate(ctx context.Context, s *Store) (*sql.Tx, error) {
	return s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
}
func RetryTransaction(ctx context.Context, s *Store, tries int, fn func(context.Context, *sql.Tx) error) error {
	if tries < 1 {
		tries = 1
	}
	var err error
	for i := 0; i < tries; i++ {
		tx, e := BeginImmediate(ctx, s)
		if e != nil {
			err = e
			continue
		}
		err = fn(ctx, tx)
		if err == nil {
			err = tx.Commit()
		} else {
			_ = tx.Rollback()
		}
		if err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(i+1) * time.Millisecond):
		}
	}
	return err
}
func RowsToStrings(rows *sql.Rows) ([]string, error) {
	defer rows.Close()
	out := make([]string, 0)
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
