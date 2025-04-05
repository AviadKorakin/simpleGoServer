package errors

import "fmt"

// HTTPError represents an error with an associated HTTP status code.
type HTTPError struct {
	Code int
	Msg  string
}

// Error implements the error interface.
func (e *HTTPError) Error() string {
	return fmt.Sprintf("%d: %s", e.Code, e.Msg)
}

// NewHTTPError creates a new HTTPError with the given code and message.
func NewHTTPError(code int, msg string) error {
	return &HTTPError{
		Code: code,
		Msg:  msg,
	}
}