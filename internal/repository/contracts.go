package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/zhanglei10281852-gif/dianchi/internal/domain/audit"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/battery"
	"github.com/zhanglei10281852-gif/dianchi/internal/pagination"
)

type Tx interface {
	Commit() error
	Rollback() error
}
type Store interface {
	WithTx(context.Context, func(context.Context, *sql.Tx) error) error
	CreateOperator(context.Context, auth.Principal, string) error
	FindOperator(context.Context, string, string) (auth.Principal, string, error)
	CreateSession(context.Context, string, auth.Principal, time.Time) error
	FindSession(context.Context, string, time.Time) (auth.Principal, error)
	RevokeSession(context.Context, string, time.Time) error
	InsertLot(context.Context, *sql.Tx, battery.Lot) error
	GetLot(context.Context, *sql.Tx, string, string) (battery.Lot, error)
	UpdateLotState(context.Context, *sql.Tx, string, string, battery.State, int) error
	ListLots(context.Context, string, pagination.Query) (pagination.Result[battery.Lot], error)
	InsertInspection(context.Context, *sql.Tx, battery.Inspection) error
	LatestInspection(context.Context, *sql.Tx, string, string) (battery.Inspection, error)
	InsertReservation(context.Context, *sql.Tx, battery.Reservation) error
	ActiveReservation(context.Context, *sql.Tx, string, string) (battery.Reservation, error)
	InsertRecovery(context.Context, *sql.Tx, battery.Recovery) error
	InsertCertificate(context.Context, *sql.Tx, battery.Certificate) error
	FindCertificate(context.Context, *sql.Tx, string, string) (battery.Certificate, error)
	InsertAudit(context.Context, *sql.Tx, audit.Event) error
	InsertOutbox(context.Context, *sql.Tx, string, string, string, string, string) error
	ClaimOutbox(context.Context, time.Time) (string, string, string, error)
	MarkOutbox(context.Context, string, error) error
	SaveIdempotency(context.Context, *sql.Tx, string, string, string, string) error
	GetIdempotency(context.Context, *sql.Tx, string, string, string) (string, error)
}
