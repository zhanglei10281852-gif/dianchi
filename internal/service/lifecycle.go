package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/zhanglei10281852-gif/dianchi/internal/apperr"
	"github.com/zhanglei10281852-gif/dianchi/internal/clock"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/audit"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/battery"
	"github.com/zhanglei10281852-gif/dianchi/internal/pagination"
	"github.com/zhanglei10281852-gif/dianchi/internal/repository"
)

type Service struct {
	Store    repository.Store
	Clock    clock.Clock
	Audit    repository.Store
	Requests *RequestTracker
}
type RequestTracker struct{ Seen map[string]struct{} }

func New(store repository.Store, c clock.Clock) *Service {
	return &Service{Store: store, Clock: c, Audit: store, Requests: &RequestTracker{Seen: map[string]struct{}{}}}
}

func (s *Service) Intake(ctx context.Context, p auth.Principal, code, chem string, hazard int, expires *time.Time, requestID string) (battery.Lot, error) {
	if !p.Can("intake") {
		return battery.Lot{}, apperr.New(apperr.Forbidden, "role cannot intake")
	}
	chem, err := battery.NormalizeChemistry(chem)
	if err != nil {
		return battery.Lot{}, apperr.Wrap(apperr.Invalid, "invalid chemistry", err)
	}
	if code == "" {
		return battery.Lot{}, apperr.Invalidf("code required")
	}
	lot := battery.Lot{ID: token(), TenantID: p.TenantID, Code: code, Chemistry: chem, State: battery.Received, Version: 1, ReceivedAt: s.Clock.Now(), ExpiresAt: expires, HazardScore: hazard, CreatedBy: p.ID}
	err = s.Store.WithTx(ctx, func(txctx context.Context, tx *sql.Tx) error {
		if err := s.Store.InsertLot(txctx, tx, lot); err != nil {
			return err
		}
		return s.writeAudit(txctx, tx, p, lot.ID, "intake", "success", requestID)
	})
	return lot, err
}
func (s *Service) Inspect(ctx context.Context, p auth.Principal, lotID string, result battery.InspectionResult, notes, requestID string) (battery.Lot, error) {
	if !p.Can("inspect") {
		return battery.Lot{}, apperr.New(apperr.Forbidden, "role cannot inspect")
	}
	var out battery.Lot
	err := s.Store.WithTx(ctx, func(txctx context.Context, tx *sql.Tx) error {
		lot, e := s.Store.GetLot(txctx, tx, lotID, p.TenantID)
		if e != nil {
			return e
		}
		if !lot.CanInspect() {
			return apperr.New(apperr.Conflict, "lot is not awaiting inspection")
		}
		target := battery.Inspected
		if result == battery.InspectionHold || result == battery.InspectionReject {
			target = battery.Quarantined
		}
		if e := s.Store.UpdateLotState(txctx, tx, lotID, p.TenantID, target, lot.Version); e != nil {
			return e
		}
		lot.State = target
		ins := battery.Inspection{ID: token(), LotID: lotID, TenantID: p.TenantID, Result: result, Notes: notes, InspectorID: p.ID, CreatedAt: s.Clock.Now()}
		if e := s.Store.InsertInspection(txctx, tx, ins); e != nil {
			return e
		}
		if e := s.writeAudit(txctx, tx, p, lotID, "inspection", string(result), requestID); e != nil {
			return e
		}
		out = lot
		return nil
	})
	return out, err
}
func (s *Service) Reserve(ctx context.Context, p auth.Principal, lotID, station, requestID string) (battery.Reservation, error) {
	if !p.Can("reserve") {
		return battery.Reservation{}, apperr.New(apperr.Forbidden, "role cannot reserve")
	}
	var out battery.Reservation
	err := s.Store.WithTx(ctx, func(txctx context.Context, tx *sql.Tx) error {
		lot, e := s.Store.GetLot(txctx, tx, lotID, p.TenantID)
		if e != nil {
			return e
		}
		if !lot.CanReserve() {
			return apperr.New(apperr.Conflict, "lot is not eligible for reservation")
		}
		if _, e = s.Store.ActiveReservation(txctx, tx, lotID, p.TenantID); e == nil {
			return apperr.New(apperr.Conflict, "lot already reserved")
		}
		r := battery.Reservation{ID: token(), LotID: lotID, TenantID: p.TenantID, Station: station, OperatorID: p.ID, State: "active", CreatedAt: s.Clock.Now()}
		if e = s.Store.InsertReservation(txctx, tx, r); e != nil {
			return e
		}
		if e = s.Store.UpdateLotState(txctx, tx, lotID, p.TenantID, battery.Reserved, lot.Version); e != nil {
			return e
		}
		if e = s.writeAudit(txctx, tx, p, lotID, "reserve", station, requestID); e != nil {
			return e
		}
		out = r
		return nil
	})
	return out, err
}
func (s *Service) BeginDismantling(ctx context.Context, p auth.Principal, lotID, requestID string) error {
	if !p.Can("reserve") {
		return apperr.New(apperr.Forbidden, "role cannot begin dismantling")
	}
	return s.Store.WithTx(ctx, func(txctx context.Context, tx *sql.Tx) error {
		lot, e := s.Store.GetLot(txctx, tx, lotID, p.TenantID)
		if e != nil {
			return e
		}
		if lot.State != battery.Reserved {
			return apperr.New(apperr.Conflict, "lot has no active reservation")
		}
		if _, e = s.Store.ActiveReservation(txctx, tx, lotID, p.TenantID); e != nil {
			return apperr.New(apperr.Conflict, "reservation missing")
		}
		if e = s.Store.UpdateLotState(txctx, tx, lotID, p.TenantID, battery.Dismantling, lot.Version); e != nil {
			return e
		}
		return s.writeAudit(txctx, tx, p, lotID, "dismantling", "started", requestID)
	})
}
func (s *Service) Recover(ctx context.Context, p auth.Principal, lotID string, li, ni, co int, idempotency, requestID string) (battery.Recovery, error) {
	if !p.Can("recover") {
		return battery.Recovery{}, apperr.New(apperr.Forbidden, "role cannot recover")
	}
	var out battery.Recovery
	err := s.Store.WithTx(ctx, func(txctx context.Context, tx *sql.Tx) error {
		if idempotency != "" {
			if raw, e := s.Store.GetIdempotency(txctx, tx, p.TenantID, idempotency, "recover"); e == nil {
				_ = json.Unmarshal([]byte(raw), &out)
				return nil
			}
		}
		lot, e := s.Store.GetLot(txctx, tx, lotID, p.TenantID)
		if e != nil {
			return e
		}
		if !lot.CanRecover() {
			return apperr.New(apperr.Conflict, "lot is not in dismantling")
		}
		if li < 0 || ni < 0 || co < 0 || li+ni+co == 0 {
			return apperr.Invalidf("material weights must be positive")
		}
		out = battery.Recovery{ID: token(), LotID: lotID, TenantID: p.TenantID, LithiumGrams: li, NickelGrams: ni, CobaltGrams: co, State: "confirmed", Version: 1, RecoveredAt: s.Clock.Now()}
		if e = s.Store.InsertRecovery(txctx, tx, out); e != nil {
			return e
		}
		if e = s.Store.UpdateLotState(txctx, tx, lotID, p.TenantID, battery.Recovered, lot.Version); e != nil {
			return e
		}
		payload, _ := json.Marshal(out)
		if idempotency != "" {
			if e = s.Store.SaveIdempotency(txctx, tx, p.TenantID, idempotency, "recover", string(payload)); e != nil {
				return e
			}
		}
		if e = s.Store.InsertOutbox(txctx, tx, token(), p.TenantID, lotID, "recovery.confirmed", string(payload)); e != nil {
			return e
		}
		return s.writeAudit(txctx, tx, p, lotID, "recover", "confirmed", requestID)
	})
	return out, err
}
func (s *Service) Certify(ctx context.Context, p auth.Principal, lotID, requestID string) (battery.Certificate, error) {
	if !p.Can("certify") {
		return battery.Certificate{}, apperr.New(apperr.Forbidden, "only supervisors certify")
	}
	var out battery.Certificate
	err := s.Store.WithTx(ctx, func(txctx context.Context, tx *sql.Tx) error {
		lot, e := s.Store.GetLot(txctx, tx, lotID, p.TenantID)
		if e != nil {
			return e
		}
		if !lot.CanCertify() {
			return apperr.New(apperr.Conflict, "lot is not recovered")
		}
		if _, e = s.Store.FindCertificate(txctx, tx, lotID, p.TenantID); e == nil {
			return apperr.New(apperr.Conflict, "certificate already exists")
		}
		out = battery.Certificate{ID: token(), LotID: lotID, TenantID: p.TenantID, Serial: fmt.Sprintf("DC-%s", lot.Code), State: "issued", CreatedAt: s.Clock.Now()}
		if e = s.Store.InsertCertificate(txctx, tx, out); e != nil {
			return e
		}
		if e = s.Store.UpdateLotState(txctx, tx, lotID, p.TenantID, battery.Certified, lot.Version); e != nil {
			return e
		}
		return s.writeAudit(txctx, tx, p, lotID, "certificate", "issued", requestID)
	})
	return out, err
}
func (s *Service) List(ctx context.Context, p auth.Principal, q pagination.Query) (pagination.Result[battery.Lot], error) {
	if !p.Can("intake") {
		return pagination.Result[battery.Lot]{}, apperr.New(apperr.Forbidden, "role cannot list lots")
	}
	return s.Store.ListLots(ctx, p.TenantID, q)
}
func (s *Service) writeAudit(ctx context.Context, tx *sql.Tx, p auth.Principal, obj, action, result, requestID string) error {
	e := audit.Event{ID: token(), TenantID: p.TenantID, ActorID: p.ID, ObjectType: "lot", ObjectID: obj, Action: action, Result: result, RequestID: requestID, Payload: "{}", CreatedAt: s.Clock.Now()}
	return s.Audit.InsertAudit(ctx, tx, e)
}
func (s *Service) ValidateContext(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}
