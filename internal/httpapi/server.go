package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/zhanglei10281852-gif/dianchi/internal/apperr"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/battery"
	"github.com/zhanglei10281852-gif/dianchi/internal/middleware"
	"github.com/zhanglei10281852-gif/dianchi/internal/pagination"
	"github.com/zhanglei10281852-gif/dianchi/internal/service"
)

type Server struct {
	Service *service.Service
	Auth    service.Auth
	Mux     *http.ServeMux
}

func New(s *service.Service, a service.Auth) *Server {
	h := &Server{Service: s, Auth: a, Mux: http.NewServeMux()}
	h.routes()
	return h
}
func (h *Server) Handler() http.Handler { return middleware.RequestContext(h.Mux) }
func (h *Server) routes() {
	h.Mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { write(w, 200, map[string]any{"ok": true}) })
	h.Mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		if err := h.Service.ValidateContext(r.Context()); err != nil {
			writeErr(w, err)
			return
		}
		write(w, 200, map[string]any{"ready": true})
	})
	h.Mux.HandleFunc("POST /v1/auth/login", h.login)
	h.Mux.HandleFunc("POST /v1/auth/logout", h.logout)
	h.Mux.HandleFunc("POST /v1/lots", h.withPrincipal(h.intake))
	h.Mux.HandleFunc("GET /v1/lots", h.withPrincipal(h.list))
	h.Mux.HandleFunc("POST /v1/lots/", h.withPrincipal(h.action))
}

type principalHandler func(http.ResponseWriter, *http.Request, auth.Principal)

func (h *Server) withPrincipal(fn principalHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, err := h.Auth.Principal(r.Context(), strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		if err != nil {
			writeErr(w, err)
			return
		}
		fn(w, r, p)
	}
}
func (h *Server) login(w http.ResponseWriter, r *http.Request) {
	var in struct{ Tenant, Name, Password string }
	if !decode(r, &in) {
		writeErr(w, apperr.Invalidf("invalid JSON"))
		return
	}
	token, p, err := h.Auth.Login(r.Context(), in.Tenant, in.Name, in.Password)
	if err != nil {
		writeErr(w, err)
		return
	}
	write(w, 200, map[string]any{"token": token, "operator": p})
}
func (h *Server) logout(w http.ResponseWriter, r *http.Request) {
	if err := h.Auth.Logout(r.Context(), strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")); err != nil {
		writeErr(w, err)
		return
	}
	write(w, 204, nil)
}
func (h *Server) intake(w http.ResponseWriter, r *http.Request, p auth.Principal) {
	var in struct {
		Code, Chemistry string
		Hazard          int
		ExpiresAt       string
	}
	if !decode(r, &in) {
		writeErr(w, apperr.Invalidf("invalid JSON"))
		return
	}
	var exp *time.Time
	if in.ExpiresAt != "" {
		t, e := time.Parse(time.RFC3339, in.ExpiresAt)
		if e != nil {
			writeErr(w, apperr.Invalidf("invalid expiration"))
			return
		}
		exp = &t
	}
	lot, err := h.Service.Intake(r.Context(), p, in.Code, in.Chemistry, in.Hazard, exp, middleware.ID(r.Context()))
	if err != nil {
		writeErr(w, err)
		return
	}
	write(w, 201, lot)
}
func (h *Server) list(w http.ResponseWriter, r *http.Request, p auth.Principal) {
	q := pagination.Query{Limit: toInt(r.URL.Query().Get("limit")), Offset: toInt(r.URL.Query().Get("offset")), State: r.URL.Query().Get("state"), Chemistry: r.URL.Query().Get("chemistry")}
	out, err := h.Service.List(r.Context(), p, q)
	if err != nil {
		writeErr(w, err)
		return
	}
	write(w, 200, out)
}
func (h *Server) action(w http.ResponseWriter, r *http.Request, p auth.Principal) {
	id := strings.TrimPrefix(r.URL.Path, "/v1/lots/")
	parts := strings.Split(id, "/")
	if len(parts) != 2 {
		writeErr(w, apperr.New(apperr.NotFound, "action not found"))
		return
	}
	var in struct {
		Result                                 battery.InspectionResult
		Notes, Station                         string
		LithiumGrams, NickelGrams, CobaltGrams int
		IdempotencyKey                         string
	}
	if !decode(r, &in) {
		writeErr(w, apperr.Invalidf("invalid JSON"))
		return
	}
	var v any
	var err error
	switch parts[1] {
	case "inspection":
		v, err = h.Service.Inspect(r.Context(), p, parts[0], in.Result, in.Notes, middleware.ID(r.Context()))
	case "reserve":
		v, err = h.Service.Reserve(r.Context(), p, parts[0], in.Station, middleware.ID(r.Context()))
	case "dismantling":
		err = h.Service.BeginDismantling(r.Context(), p, parts[0], middleware.ID(r.Context()))
	case "recover":
		v, err = h.Service.Recover(r.Context(), p, parts[0], in.LithiumGrams, in.NickelGrams, in.CobaltGrams, in.IdempotencyKey, middleware.ID(r.Context()))
	default:
		err = apperr.New(apperr.NotFound, "action not found")
	}
	if err != nil {
		writeErr(w, err)
		return
	}
	write(w, 200, v)
}
func decode(r *http.Request, v any) bool { return json.NewDecoder(r.Body).Decode(v) == nil }
func write(w http.ResponseWriter, status int, v any) {
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}
func writeErr(w http.ResponseWriter, err error) {
	status := 500
	code := apperr.Internal
	msg := "internal error"
	var e *apperr.Error
	if errors.As(err, &e) {
		code = e.Code
		msg = e.Message
		switch e.Code {
		case apperr.Invalid:
			status = 400
		case apperr.Unauthorized:
			status = 401
		case apperr.Forbidden:
			status = 403
		case apperr.NotFound:
			status = 404
		case apperr.Conflict:
			status = 409
		}
	}
	write(w, status, map[string]any{"code": code, "message": msg})
}
func toInt(v string) int { var n int; _, _ = fmt.Sscanf(v, "%d", &n); return n }

var _ context.Context
