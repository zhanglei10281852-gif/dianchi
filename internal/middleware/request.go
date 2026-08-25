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
		// Carry the request id on the request context itself, preserving its
		// cancellation channel. The client disconnect cancels r.Context(); the
		// service call chain must observe that so an in-flight intake is
		// aborted instead of committing a duplicate batch on network retry.
		ctx := context.WithValue(r.Context(), RequestID, id)
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OperationContext returns a copy of ctx that is not canceled when ctx is
// canceled. It is intended only for work that must outlive a single request,
// such as resolving the bearer token into a principal during authentication; it
// must NOT be propagated into the business call chain, which must stay bound to
// the request's cancellation.
func OperationContext(ctx context.Context) context.Context {
	return context.WithoutCancel(ctx)
}

func ID(ctx context.Context) string {
	if v, ok := ctx.Value(RequestID).(string); ok {
		return v
	}
	return "missing"
}
