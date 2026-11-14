// Copyright (C) 2026 Ratify Data
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.
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
