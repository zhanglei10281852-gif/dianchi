package httpapi

import (
	"github.com/zhanglei10281852-gif/dianchi/internal/apperr"
	"net/http"
	"strings"
)

func RequireMethod(r *http.Request, want string) error {
	if r.Method != want {
		return apperr.Invalidf("method %s is not allowed", r.Method)
	}
	return nil
}
func Bearer(r *http.Request) (string, error) {
	raw := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(raw, "Bearer ") {
		return "", apperr.New(apperr.Unauthorized, "bearer token required")
	}
	token := strings.TrimSpace(strings.TrimPrefix(raw, "Bearer "))
	if token == "" {
		return "", apperr.New(apperr.Unauthorized, "empty token")
	}
	return token, nil
}
func Header(r *http.Request, name string) string { return strings.TrimSpace(r.Header.Get(name)) }
