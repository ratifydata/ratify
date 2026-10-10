package apperrors

import (
	"errors"
	"testing"
)

func TestErrorsPreserveCause(t *testing.T) {
	cause := errors.New("underlying failure")
	for _, err := range []error{
		&NotFoundError{Err: cause},
		&BadRequestError{Err: cause},
		&ConflictError{Err: cause},
	} {
		if err.Error() != cause.Error() {
			t.Errorf("message = %q", err.Error())
		}
		if !errors.Is(err, cause) {
			t.Errorf("%T does not preserve cause", err)
		}
	}
}
