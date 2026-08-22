package httpapi

import (
	"encoding/json"
	"net/http"
)

type Problem struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}
func NoContent(w http.ResponseWriter) { w.WriteHeader(http.StatusNoContent) }
func DecodeJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}
func WriteProblem(w http.ResponseWriter, status int, p Problem) { JSON(w, status, p) }
func StatusFor(code string) int {
	switch code {
	case "invalid":
		return 400
	case "unauthorized":
		return 401
	case "forbidden":
		return 403
	case "not_found":
		return 404
	case "conflict":
		return 409
	case "unavailable":
		return 503
	default:
		return 500
	}
}
