// Package apperrors defines errors shared by services and HTTP handlers.
package apperrors

type NotFoundError struct {
	Err error
}

func (e *NotFoundError) Error() string { return e.Err.Error() }
func (e *NotFoundError) Unwrap() error { return e.Err }

type BadRequestError struct {
	Err error
}

func (e *BadRequestError) Error() string { return e.Err.Error() }
func (e *BadRequestError) Unwrap() error { return e.Err }

type ConflictError struct {
	Err error
}

func (e *ConflictError) Error() string { return e.Err.Error() }
func (e *ConflictError) Unwrap() error { return e.Err }
