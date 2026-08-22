package httpapi

import (
	"errors"
	"github.com/zhanglei10281852-gif/dianchi/internal/apperr"
	"github.com/zhanglei10281852-gif/dianchi/internal/middleware"
	"net/http"
)

func ErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	status := http.StatusInternalServerError
	code := string(apperr.Internal)
	msg := "internal error"
	var e *apperr.Error
	if errors.As(err, &e) {
		code = string(e.Code)
		msg = e.Message
		status = StatusFor(code)
	}
	WriteProblem(w, status, Problem{Code: code, Message: msg, RequestID: middleware.ID(r.Context())})
}
func MethodNotAllowed(w http.ResponseWriter, r *http.Request) {
	ErrorResponse(w, r, apperr.Invalidf("method %s is not allowed", r.Method))
}
