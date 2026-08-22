package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

type key int

const RequestID key = 1

func RequestContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b := make([]byte, 8)
		_, _ = rand.Read(b)
		id := hex.EncodeToString(b)
		ctx := context.WithValue(r.Context(), RequestID, id)
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
func ID(ctx context.Context) string {
	if v, ok := ctx.Value(RequestID).(string); ok {
		return v
	}
	return "missing"
}
