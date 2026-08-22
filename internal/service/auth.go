package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/zhanglei10281852-gif/dianchi/internal/apperr"
	"github.com/zhanglei10281852-gif/dianchi/internal/clock"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/repository"
)

type Auth struct {
	Store      repository.Store
	Clock      clock.Clock
	SessionTTL time.Duration
}

func (a Auth) Register(ctx context.Context, p auth.Principal, password string) error {
	if p.Role != "operator" && p.Role != "supervisor" {
		return apperr.Invalidf("unsupported role")
	}
	return a.Store.CreateOperator(ctx, p, password)
}
func (a Auth) Login(ctx context.Context, tenant, name, password string) (string, auth.Principal, error) {
	ctx = repository.SessionContext(ctx)
	p, stored, err := a.Store.FindOperator(ctx, tenant, name)
	if err != nil {
		return "", p, err
	}
	if stored != hashPassword(password) {
		return "", p, apperr.New(apperr.Unauthorized, "invalid credentials")
	}
	id := token()
	expires := a.Clock.Now().Add(a.SessionTTL)
	if err := a.Store.CreateSession(ctx, id, p, expires); err != nil {
		return "", p, err
	}
	p.SessionID = id
	return id, p, nil
}
func (a Auth) Principal(ctx context.Context, token string) (auth.Principal, error) {
	if token == "" {
		return auth.Principal{}, apperr.New(apperr.Unauthorized, "missing session")
	}
	return a.Store.FindSession(ctx, token, a.Clock.Now())
}
func (a Auth) Logout(ctx context.Context, token string) error {
	return a.Store.RevokeSession(ctx, token, a.Clock.Now())
}
func token() string {
	b := make([]byte, 18)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
func hashPassword(v string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(v)))
}
