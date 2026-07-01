package errorsuc

import "errors"

// Typed application errors let the HTTP layer map use-case failures to the
// correct status code (400/404/409/422) instead of collapsing everything to
// 500. Use errors.As/errors.Is at the boundary (see security.RespondUseCaseError).

// ValidationError signals a client-side problem with the request payload
// (missing/invalid fields). Maps to HTTP 422.
type ValidationError struct{ Msg string }

func (e *ValidationError) Error() string { return e.Msg }

// NewValidationError builds a ValidationError with the given message.
func NewValidationError(msg string) error { return &ValidationError{Msg: msg} }

// ConflictError signals that the request conflicts with existing state, most
// commonly a duplicate unique key. Maps to HTTP 409.
type ConflictError struct{ Msg string }

func (e *ConflictError) Error() string { return e.Msg }

// NewConflictError builds a ConflictError with the given message.
func NewConflictError(msg string) error { return &ConflictError{Msg: msg} }

// NotFoundError signals that the requested resource does not exist. Maps to 404.
type NotFoundError struct{ Msg string }

func (e *NotFoundError) Error() string { return e.Msg }

// NewNotFoundError builds a NotFoundError with the given message.
func NewNotFoundError(msg string) error { return &NotFoundError{Msg: msg} }

// AsValidation reports whether err is (or wraps) a ValidationError.
func AsValidation(err error) (*ValidationError, bool) {
	var v *ValidationError
	if errors.As(err, &v) {
		return v, true
	}
	return nil, false
}

// AsConflict reports whether err is (or wraps) a ConflictError.
func AsConflict(err error) (*ConflictError, bool) {
	var c *ConflictError
	if errors.As(err, &c) {
		return c, true
	}
	return nil, false
}

// AsNotFound reports whether err is (or wraps) a NotFoundError.
func AsNotFound(err error) (*NotFoundError, bool) {
	var n *NotFoundError
	if errors.As(err, &n) {
		return n, true
	}
	return nil, false
}
