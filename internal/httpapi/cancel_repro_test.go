package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zhanglei10281852-gif/dianchi/internal/storage/sqlite"
)

// TestClientCancelStopsIntake reproduces the request-cancel-loss bug: when the
// mobile client disconnects immediately after issuing an intake request, the
// request context is canceled. The service call chain must observe that
// cancellation so the battery batch is not created; otherwise a network retry
// produces a duplicate business operation.
func TestClientCancelStopsIntake(t *testing.T) {
	h, closeFn := httpFixture(t)
	defer closeFn()

	w := request(t, h.Handler(), http.MethodPost, "/v1/auth/login", map[string]string{"Tenant": "t", "Name": "op", "Password": "pw"}, "")
	if w.Code != 200 {
		t.Fatalf("login=%d %s", w.Code, w.Body.String())
	}
	var auth struct{ Token string }
	if err := json.Unmarshal(w.Body.Bytes(), &auth); err != nil || auth.Token == "" {
		t.Fatal(w.Body.String())
	}

	// Client disconnects the instant the request is issued: the request
	// context is already canceled before the handler runs.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var b bytes.Buffer
	_ = json.NewEncoder(&b).Encode(map[string]any{"Code": "CANCEL-1", "Chemistry": "lfp", "Hazard": 3})
	r := httptest.NewRequest(http.MethodPost, "/v1/lots", &b)
	r = r.WithContext(ctx)
	r.Header.Set("Authorization", "Bearer "+auth.Token)

	rec := httptest.NewRecorder()
	h.Handler().ServeHTTP(rec, r)

	store, ok := h.Service.Store.(*sqlite.Store)
	if !ok {
		t.Fatalf("store=%T", h.Service.Store)
	}
	var n int
	if err := store.DB.QueryRow(`SELECT COUNT(*) FROM lots WHERE code=?`, "CANCEL-1").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("cancelled intake created lot; count=%d", n)
	}
}
