package repository

import (
	"context"
	"database/sql"
	"fmt"
)

type ManifestRow struct {
	ID, Tenant, Origin, Destination, Status string
	Weight                                  int
}

func InsertManifest(ctx context.Context, tx *sql.Tx, m ManifestRow) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO manifests(id,tenant_id,origin,destination,status,total_weight) VALUES(?,?,?,?,?,?)`, m.ID, m.Tenant, m.Origin, m.Destination, m.Status, m.Weight)
	return err
}
func InsertManifestItem(ctx context.Context, tx *sql.Tx, id, manifest, lot, tenant string, weight int) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO manifest_items(id,manifest_id,lot_id,tenant_id,weight) VALUES(?,?,?,?,?)`, id, manifest, lot, tenant, weight)
	return err
}
func ReadManifest(ctx context.Context, row *sql.Row) (ManifestRow, error) {
	var m ManifestRow
	err := row.Scan(&m.ID, &m.Tenant, &m.Origin, &m.Destination, &m.Status, &m.Weight)
	return m, err
}
func RequireManifestStatus(status string) error {
	switch status {
	case "prepared", "sealed", "dispatched", "received", "cancelled":
		return nil
	default:
		return fmt.Errorf("unknown manifest status")
	}
}
