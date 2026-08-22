package repository

import (
	"context"
	"database/sql"
	"fmt"
)

type ComplianceRow struct{ ID, Tenant, Lot, Serial, Status string }

func InsertCompliance(ctx context.Context, tx *sql.Tx, v ComplianceRow) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO compliance_certificates(id,tenant_id,lot_id,serial,status) VALUES(?,?,?,?,?)`, v.ID, v.Tenant, v.Lot, v.Serial, v.Status)
	return err
}
func UpdateCompliance(ctx context.Context, tx *sql.Tx, id, tenant, status string) error {
	r, err := tx.ExecContext(ctx, `UPDATE compliance_certificates SET status=? WHERE id=? AND tenant_id=?`, status, id, tenant)
	if err != nil {
		return err
	}
	n, _ := r.RowsAffected()
	if n != 1 {
		return fmt.Errorf("compliance record missing")
	}
	return nil
}
func ReadCompliance(ctx context.Context, row *sql.Row) (ComplianceRow, error) {
	var v ComplianceRow
	err := row.Scan(&v.ID, &v.Tenant, &v.Lot, &v.Serial, &v.Status)
	return v, err
}
