package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEcoHTTPRejectsMalformedBearer(t *testing.T) {
	h, close := httpFixture(t)
	defer close()
	w := request(t, h.Handler(), http.MethodPost, "/v1/auth/login", map[string]string{"Tenant": "t", "Name": "op", "Password": "pw"}, "")
	var out struct{ Token string }
	if e := json.Unmarshal(w.Body.Bytes(), &out); e != nil || out.Token == "" {
		t.Fatal(w.Body.String())
	}
	r := httptest.NewRequest(http.MethodGet, "/v1/lots", nil)
	r.Header.Set("Authorization", "Bearer"+out.Token)
	rw := httptest.NewRecorder()
	h.Handler().ServeHTTP(rw, r)
	if rw.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", rw.Code)
	}
}
