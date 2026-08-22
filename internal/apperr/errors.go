package apperr

import "fmt"

type Code string

const (
	NotFound     Code = "not_found"
	Conflict     Code = "conflict"
	Invalid      Code = "invalid"
	Forbidden    Code = "forbidden"
	Unauthorized Code = "unauthorized"
	Unavailable  Code = "unavailable"
	Internal     Code = "internal"
)

type Error struct {
	Code    Code
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e.Cause == nil {
		return string(e.Code) + ": " + e.Message
	}
	return string(e.Code) + ": " + e.Message + ": " + e.Cause.Error()
}
func (e *Error) Unwrap() error             { return e.Cause }
func New(code Code, message string) *Error { return &Error{Code: code, Message: message} }
func Wrap(code Code, message string, cause error) *Error {
	return &Error{Code: code, Message: message, Cause: cause}
}
func Invalidf(format string, args ...any) error { return New(Invalid, fmt.Sprintf(format, args...)) }
