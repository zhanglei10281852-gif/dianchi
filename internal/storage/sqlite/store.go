package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/zhanglei10281852-gif/dianchi/internal/apperr"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/audit"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/battery"
	"github.com/zhanglei10281852-gif/dianchi/internal/pagination"
	"github.com/zhanglei10281852-gif/dianchi/internal/repository"
	_ "modernc.org/sqlite"
)

type Store struct{ DB *sql.DB }

func Open(ctx context.Context, dsn string) (*Store, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if _, err = db.ExecContext(ctx, `PRAGMA foreign_keys = ON`); err != nil {
		db.Close()
		return nil, err
	}
	data, err := os.ReadFile("migrations/001_initial.sql")
	if err != nil {
		data, err = os.ReadFile("../../migrations/001_initial.sql")
	}
	if err != nil {
		data, err = os.ReadFile("../../../migrations/001_initial.sql")
	}
	if err != nil {
		db.Close()
		return nil, err
	}
	if _, err = db.ExecContext(ctx, string(data)); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{DB: db}, nil
}
func (s *Store) Close() error { return s.DB.Close() }
func (s *Store) WithTx(ctx context.Context, fn func(context.Context, *sql.Tx) error) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	return repository.WithRollback(ctx, tx, func() error {
		return fn(ctx, tx)
	})
}
func scanLot(row interface{ Scan(...any) error }) (battery.Lot, error) {
	var l battery.Lot
	var state string
	var received, expires string
	err := row.Scan(&l.ID, &l.TenantID, &l.Code, &l.Chemistry, &state, &l.Version, &received, &expires, &l.HazardScore, &l.CreatedBy)
	if err != nil {
		return l, err
	}
	l.State = battery.State(state)
	l.ReceivedAt, _ = time.Parse(time.RFC3339Nano, received)
	if expires != "" {
		t, _ := time.Parse(time.RFC3339Nano, expires)
		l.ExpiresAt = &t
	}
	return l, nil
}
func (s *Store) CreateOperator(ctx context.Context, p auth.Principal, password string) error {
	_, err := s.DB.ExecContext(ctx, `INSERT INTO operators(id,tenant_id,name,role,password_hash,created_at) VALUES(?,?,?,?,?,?)`, p.ID, p.TenantID, p.Name, p.Role, hash(password), time.Now().UTC().Format(time.RFC3339Nano))
	return err
}
func (s *Store) FindOperator(ctx context.Context, tenant, name string) (auth.Principal, string, error) {
	var p auth.Principal
	var role string
	var disabled int
	var pass string
	err := s.DB.QueryRowContext(ctx, `SELECT id,tenant_id,name,role,password_hash,disabled FROM operators WHERE tenant_id=? AND name=?`, tenant, name).Scan(&p.ID, &p.TenantID, &p.Name, &role, &pass, &disabled)
	if errors.Is(err, sql.ErrNoRows) {
		return p, "", apperr.New(apperr.Unauthorized, "invalid credentials")
	}
	if err != nil {
		return p, "", err
	}
	if disabled != 0 {
		return p, "", apperr.New(apperr.Forbidden, "operator disabled")
	}
	p.Role = auth.Role(role)
	return p, pass, nil
}
func (s *Store) CreateSession(ctx context.Context, id string, p auth.Principal, expires time.Time) error {
	_, err := s.DB.ExecContext(ctx, `INSERT INTO sessions(id,operator_id,tenant_id,expires_at,created_at) VALUES(?,?,?,?,?)`, id, p.ID, p.TenantID, expires.Format(time.RFC3339Nano), time.Now().UTC().Format(time.RFC3339Nano))
	return err
}
func (s *Store) FindSession(ctx context.Context, id string, now time.Time) (auth.Principal, error) {
	var p auth.Principal
	var role, exp string
	var revoked sql.NullString
	err := s.DB.QueryRowContext(ctx, `SELECT o.id,o.tenant_id,o.name,o.role,s.expires_at,s.revoked_at FROM sessions s JOIN operators o ON o.id=s.operator_id WHERE s.id=?`, id).Scan(&p.ID, &p.TenantID, &p.Name, &role, &exp, &revoked)
	if errors.Is(err, sql.ErrNoRows) {
		return p, apperr.New(apperr.Unauthorized, "session missing")
	}
	if err != nil {
		return p, err
	}
	t, _ := time.Parse(time.RFC3339Nano, exp)
	if revoked.Valid || !now.Before(t) {
		return p, apperr.New(apperr.Unauthorized, "session expired")
	}
	p.Role = auth.Role(role)
	p.SessionID = id
	return p, nil
}
func (s *Store) RevokeSession(ctx context.Context, id string, now time.Time) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE sessions SET revoked_at=? WHERE id=? AND revoked_at IS NULL`, now.Format(time.RFC3339Nano), id)
	return err
}
func (s *Store) InsertLot(ctx context.Context, tx *sql.Tx, l battery.Lot) error {
	ex := ""
	if l.ExpiresAt != nil {
		ex = l.ExpiresAt.Format(time.RFC3339Nano)
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO lots(id,tenant_id,code,chemistry,state,version,received_at,expires_at,hazard_score,created_by) VALUES(?,?,?,?,?,?,?,?,?,?)`, l.ID, l.TenantID, l.Code, l.Chemistry, l.State, l.Version, l.ReceivedAt.Format(time.RFC3339Nano), ex, l.HazardScore, l.CreatedBy)
	return err
}
func (s *Store) GetLot(ctx context.Context, tx *sql.Tx, id, tenant string) (battery.Lot, error) {
	var l battery.Lot
	var err error
	query := `SELECT id,tenant_id,code,chemistry,state,version,received_at,expires_at,hazard_score,created_by FROM lots WHERE id=? AND tenant_id=?`
	if tx != nil {
		l, err = scanLot(tx.QueryRowContext(ctx, query, id, tenant))
	} else {
		l, err = scanLot(s.DB.QueryRowContext(ctx, query, id, tenant))
	}
	if errors.Is(err, sql.ErrNoRows) {
		return l, apperr.New(apperr.NotFound, "lot not found")
	}
	return l, err
}
func (s *Store) UpdateLotState(ctx context.Context, tx *sql.Tx, id, tenant string, state battery.State, version int) error {
	r, err := tx.ExecContext(ctx, `UPDATE lots SET state=?,version=version+1 WHERE id=? AND tenant_id=? AND version=?`, state, id, tenant, version)
	if err != nil {
		return err
	}
	n, _ := r.RowsAffected()
	if n != 1 {
		return apperr.New(apperr.Conflict, "lot version changed")
	}
	return nil
}
func (s *Store) ListLots(ctx context.Context, tenant string, q pagination.Query) (pagination.Result[battery.Lot], error) {
	q = pagination.Normalize(q)
	where := []string{"tenant_id=?"}
	args := []any{tenant}
	if q.State != "" {
		where = append(where, "state=?")
		args = append(args, q.State)
	}
	if q.Chemistry != "" {
		where = append(where, "chemistry=?")
		args = append(args, q.Chemistry)
	}
	w := strings.Join(where, " AND ")
	var total int
	if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM lots WHERE "+w, args...).Scan(&total); err != nil {
		return pagination.Result[battery.Lot]{}, err
	}
	rows, err := s.DB.QueryContext(ctx, "SELECT id,tenant_id,code,chemistry,state,version,received_at,expires_at,hazard_score,created_by FROM lots WHERE "+w+" ORDER BY received_at DESC LIMIT ? OFFSET ?", append(args, q.Limit, q.Offset)...)
	if err != nil {
		return pagination.Result[battery.Lot]{}, err
	}
	defer rows.Close()
	out := pagination.Result[battery.Lot]{Items: make([]battery.Lot, 0), Total: total, Limit: q.Limit, Offset: q.Offset}
	for rows.Next() {
		l, e := scanLot(rows)
		if e != nil {
			return out, e
		}
		out.Items = append(out.Items, l)
	}
	return out, rows.Err()
}
func (s *Store) InsertInspection(ctx context.Context, tx *sql.Tx, i battery.Inspection) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO inspections(id,lot_id,tenant_id,result,notes,inspector_id,created_at) VALUES(?,?,?,?,?,?,?)`, i.ID, i.LotID, i.TenantID, i.Result, i.Notes, i.InspectorID, i.CreatedAt.Format(time.RFC3339Nano))
	return err
}
func (s *Store) LatestInspection(ctx context.Context, tx *sql.Tx, lot, tenant string) (battery.Inspection, error) {
	var i battery.Inspection
	var result, created string
	row := tx.QueryRowContext(ctx, `SELECT id,lot_id,tenant_id,result,notes,inspector_id,created_at FROM inspections WHERE lot_id=? AND tenant_id=? ORDER BY created_at DESC LIMIT 1`, lot, tenant)
	err := row.Scan(&i.ID, &i.LotID, &i.TenantID, &result, &i.Notes, &i.InspectorID, &created)
	i.Result = battery.InspectionResult(result)
	i.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	return i, err
}
func (s *Store) InsertReservation(ctx context.Context, tx *sql.Tx, r battery.Reservation) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO reservations(id,lot_id,tenant_id,station,operator_id,state,created_at) VALUES(?,?,?,?,?,?,?)`, r.ID, r.LotID, r.TenantID, r.Station, r.OperatorID, r.State, r.CreatedAt.Format(time.RFC3339Nano))
	return err
}
func (s *Store) ActiveReservation(ctx context.Context, tx *sql.Tx, lot, tenant string) (battery.Reservation, error) {
	var r battery.Reservation
	var created string
	err := tx.QueryRowContext(ctx, `SELECT id,lot_id,tenant_id,station,operator_id,state,created_at FROM reservations WHERE lot_id=? AND tenant_id=? AND released_at IS NULL ORDER BY created_at DESC LIMIT 1`, lot, tenant).Scan(&r.ID, &r.LotID, &r.TenantID, &r.Station, &r.OperatorID, &r.State, &created)
	r.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	return r, err
}
func (s *Store) InsertRecovery(ctx context.Context, tx *sql.Tx, r battery.Recovery) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO recoveries(id,lot_id,tenant_id,lithium_grams,nickel_grams,cobalt_grams,state,version,recovered_at) VALUES(?,?,?,?,?,?,?,?,?)`, r.ID, r.LotID, r.TenantID, r.LithiumGrams, r.NickelGrams, r.CobaltGrams, r.State, r.Version, r.RecoveredAt.Format(time.RFC3339Nano))
	return err
}
func (s *Store) InsertCertificate(ctx context.Context, tx *sql.Tx, c battery.Certificate) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO certificates(id,lot_id,tenant_id,serial,state,created_at) VALUES(?,?,?,?,?,?)`, c.ID, c.LotID, c.TenantID, c.Serial, c.State, c.CreatedAt.Format(time.RFC3339Nano))
	return err
}
func (s *Store) FindCertificate(ctx context.Context, tx *sql.Tx, lot, tenant string) (battery.Certificate, error) {
	var c battery.Certificate
	var created string
	err := tx.QueryRowContext(ctx, `SELECT id,lot_id,tenant_id,serial,state,created_at FROM certificates WHERE lot_id=? AND tenant_id=?`, lot, tenant).Scan(&c.ID, &c.LotID, &c.TenantID, &c.Serial, &c.State, &created)
	c.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	return c, err
}
func (s *Store) InsertAudit(ctx context.Context, tx *sql.Tx, e audit.Event) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO audit_events(id,tenant_id,actor_id,object_type,object_id,action,result,request_id,payload,created_at) VALUES(?,?,?,?,?,?,?,?,?,?)`, e.ID, e.TenantID, e.ActorID, e.ObjectType, e.ObjectID, e.Action, e.Result, e.RequestID, e.Payload, e.CreatedAt.Format(time.RFC3339Nano))
	return err
}
func (s *Store) InsertOutbox(ctx context.Context, tx *sql.Tx, id, tenant, aggregate, event, payload string) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO outbox(id,tenant_id,aggregate_id,event_type,payload,state,next_attempt_at,created_at) VALUES(?,?,?,?,?,'pending',?,?)`, id, tenant, aggregate, event, payload, time.Now().UTC().Format(time.RFC3339Nano), time.Now().UTC().Format(time.RFC3339Nano))
	return err
}
func (s *Store) ClaimOutbox(ctx context.Context, now time.Time) (string, string, string, error) {
	var id, agg, payload string
	err := s.DB.QueryRowContext(ctx, `SELECT id,aggregate_id,payload FROM outbox WHERE state='pending' AND next_attempt_at<=? ORDER BY created_at LIMIT 1`, now.Format(time.RFC3339Nano)).Scan(&id, &agg, &payload)
	if err != nil {
		return "", "", "", err
	}
	_, err = s.DB.ExecContext(ctx, `UPDATE outbox SET state='processing',attempts=attempts+1 WHERE id=? AND state='pending'`, id)
	return id, agg, payload, err
}
func (s *Store) MarkOutbox(ctx context.Context, id string, err error) error {
	if err == nil {
		_, e := s.DB.ExecContext(ctx, `UPDATE outbox SET state='published' WHERE id=?`, id)
		return e
	}
	_, e := s.DB.ExecContext(ctx, `UPDATE outbox SET state=CASE WHEN attempts>=3 THEN 'failed' ELSE 'pending' END,last_error=?,next_attempt_at=? WHERE id=?`, err.Error(), time.Now().UTC().Add(time.Minute).Format(time.RFC3339Nano), id)
	return e
}
func (s *Store) SaveIdempotency(ctx context.Context, tx *sql.Tx, tenant, key, operation, response string) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO idempotency_keys(tenant_id,key,operation,response,created_at) VALUES(?,?,?,?,?)`, tenant, key, operation, response, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}
func (s *Store) GetIdempotency(ctx context.Context, tx *sql.Tx, tenant, key, operation string) (string, error) {
	var response string
	err := tx.QueryRowContext(ctx, `SELECT response FROM idempotency_keys WHERE tenant_id=? AND key=? AND operation=?`, tenant, key, operation).Scan(&response)
	return response, err
}
func hash(v string) string { return fmt.Sprintf("%x", sha256.Sum256([]byte(v))) }
