package api

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ratifydata/ratify/internal/apperrors"
)

func TestWriteHTTPError(t *testing.T) {
	for _, tt := range []struct {
		name    string
		err     error
		status  int
		message string
	}{
		{"team not found", &apperrors.NotFoundError{Err: errors.New("team not found")}, http.StatusNotFound, "team not found"},
		{"member not found", &apperrors.NotFoundError{Err: errors.New("team member not found")}, http.StatusNotFound, "team member not found"},
		{"duplicate team", &apperrors.ConflictError{Err: errors.New("team already exists")}, http.StatusConflict, "team already exists"},
		{"duplicate member", &apperrors.ConflictError{Err: errors.New("team member already exists")}, http.StatusConflict, "team member already exists"},
		{"invalid email", &apperrors.BadRequestError{Err: errors.New("valid email is required")}, http.StatusBadRequest, "valid email is required"},
		{"missing organization", &apperrors.BadRequestError{Err: errors.New("OrgID missing from context")}, http.StatusBadRequest, "OrgID missing from context"},
		{"typed not found", &apperrors.NotFoundError{Err: errors.New("missing")}, http.StatusNotFound, "missing"},
		{"typed bad request", &apperrors.BadRequestError{Err: errors.New("invalid")}, http.StatusBadRequest, "invalid"},
		{"typed conflict", &apperrors.ConflictError{Err: errors.New("duplicate")}, http.StatusConflict, "duplicate"},
		{"database error", errors.New("private database details"), http.StatusInternalServerError, "internal server error"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for _, err := range []error{tt.err, fmt.Errorf("%w", tt.err)} {
				rec := httptest.NewRecorder()
				writeHTTPError(rec, err)
				assertTeamResponse(t, rec, tt.status, Response{Status: "error", Message: tt.message})
			}
		})
	}
}
