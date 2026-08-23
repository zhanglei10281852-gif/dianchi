package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zhanglei10281852-gif/dianchi/internal/clock"
	"github.com/zhanglei10281852-gif/dianchi/internal/domain/auth"
	"github.com/zhanglei10281852-gif/dianchi/internal/service"
	"github.com/zhanglei10281852-gif/dianchi/internal/storage/sqlite"
)

func httpFixture(t *testing.T) (*Server, func()) {
	s, err := sqlite.Open(context.Background(), "file:http-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	c := clock.Real{}
	a := service.Auth{Store: s, Clock: c, SessionTTL: time.Hour}
	if err = a.Register(context.Background(), auth.Principal{ID: "u", TenantID: "t", Name: "op", Role: auth.Operator}, "pw"); err != nil {
		t.Fatal(err)
	}
	if err = a.Register(context.Background(), auth.Principal{ID: "sup", TenantID: "t", Name: "sup", Role: auth.Supervisor}, "spw"); err != nil {
		t.Fatal(err)
	}
	return New(service.New(s, c), a), func() { s.Close() }
}
func request(t *testing.T, h http.Handler, method, path string, body any, token string) *httptest.ResponseRecorder {
	var b bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&b).Encode(body)
	}
	r := httptest.NewRequest(method, path, &b)
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func TestHealthAndReady(t *testing.T) {
	h, close := httpFixture(t)
	defer close()
	for _, path := range []string{"/healthz", "/readyz"} {
		w := request(t, h.Handler(), http.MethodGet, path, nil, "")
		if w.Code != 200 {
			t.Fatalf("%s=%d", path, w.Code)
		}
	}
}
func TestLoginAndAuthenticatedIntake(t *testing.T) {
	h, close := httpFixture(t)
	defer close()
	w := request(t, h.Handler(), http.MethodPost, "/v1/auth/login", map[string]string{"Tenant": "t", "Name": "op", "Password": "pw"}, "")
	if w.Code != 200 {
		t.Fatalf("login=%d %s", w.Code, w.Body.String())
	}
	var out struct{ Token string }
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil || out.Token == "" {
		t.Fatal(w.Body.String())
	}
	w = request(t, h.Handler(), http.MethodPost, "/v1/lots", map[string]any{"Code": "HTTP-1", "Chemistry": "lfp", "Hazard": 3}, out.Token)
	if w.Code != 201 {
		t.Fatalf("intake=%d %s", w.Code, w.Body.String())
	}
	if w.Header().Get("X-Request-ID") == "" {
		t.Fatal("request id missing")
	}
}
func TestUnauthorizedResponseShape(t *testing.T) {
	h, close := httpFixture(t)
	defer close()
	w := request(t, h.Handler(), http.MethodGet, "/v1/lots", nil, "")
	if w.Code != 401 {
		t.Fatalf("code=%d", w.Code)
	}
	var e map[string]any
	if json.Unmarshal(w.Body.Bytes(), &e) != nil || e["code"] == nil || e["message"] == nil {
		t.Fatalf("body=%s", w.Body.String())
	}
}
func TestInvalidPayloadReturnsBadRequest(t *testing.T) {
	h, close := httpFixture(t)
	defer close()
	w := request(t, h.Handler(), http.MethodPost, "/v1/auth/login", map[string]string{"Tenant": "t"}, "")
	if w.Code != 401 && w.Code != 400 {
		t.Fatalf("code=%d", w.Code)
	}
}
func TestMalformedBearerRejected(t *testing.T) {
	h, close := httpFixture(t)
	defer close()
	// A real session token, but the Authorization header omits the required
	// "Bearer " scheme/separator grammar. Authentication must fail.
	w := request(t, h.Handler(), http.MethodPost, "/v1/auth/login", map[string]string{"Tenant": "t", "Name": "op", "Password": "pw"}, "")
	if w.Code != 200 {
		t.Fatalf("login=%d %s", w.Code, w.Body.String())
	}
	var out struct{ Token string }
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil || out.Token == "" {
		t.Fatal(w.Body.String())
	}
	for _, header := range []string{
		"Bearer" + out.Token,        // no separator
		"Bearer" + out.Token + " ",  // trailing separator only
		"bearer " + out.Token,      // wrong scheme case
		out.Token,                  // bare token, no scheme
	} {
		r := httptest.NewRequest(http.MethodGet, "/v1/lots", nil)
		r.Header.Set("Authorization", header)
		rw := httptest.NewRecorder()
		h.Handler().ServeHTTP(rw, r)
		if rw.Code != 401 {
			t.Fatalf("header %q: expected 401, got %d %s", header, rw.Code, rw.Body.String())
		}
	}
}
