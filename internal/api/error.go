package api

import (
	"errors"
	"net/http"

	"github.com/ratifydata/ratify/internal/apperrors"
)

func writeHTTPError(w http.ResponseWriter, err error) {

	var notFoundError *apperrors.NotFoundError
	var conflictError *apperrors.ConflictError
	var badRequestError *apperrors.BadRequestError

	w.Header().Set("Content-Type", "application/json")
	var status int
	switch {
	case errors.As(err, &notFoundError):
		status = http.StatusNotFound
	case errors.As(err, &badRequestError):
		status = http.StatusBadRequest
	case errors.As(err, &conflictError):
		status = http.StatusConflict
	default:
		status = http.StatusInternalServerError
	}
	message := err.Error()
	if status == http.StatusInternalServerError {
		message = "internal server error"
	}
	writeJSONResponse(w, status, Response{Status: "error", Message: message})
}
