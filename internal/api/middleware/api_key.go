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
package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	sqlc "github.com/ratifydata/ratify/internal/db/generated"
)

const KeyPrefix = 8
const OrgId = "OrgID"
const UserId = "UserID"

type apiKeyAuthenticator interface {
	ApiKeyAuthentication(ctx context.Context, prefix, keyHash string) (*sqlc.ApiKey, error)
	UpdateVerificationTimestamp(ctx context.Context, id pgtype.UUID) error
}

// ApiKeyAuthHandler validates the request has the pre-requisite authentication headers
func ApiKeyAuthHandler(apiKeyAuth apiKeyAuthenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}
			//Strip the Bearer Prefix. If it lacks, return 401 for invalid format
			if !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}
			authHeaderValue := strings.TrimPrefix(authHeader, "Bearer ")
			//Trim the prefix (8 Characters). Should be unique
			if len(authHeaderValue) <= KeyPrefix+1 || authHeaderValue[KeyPrefix] != '.' {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}
			prefix := authHeaderValue[:KeyPrefix]
			keyHash := authHeaderValue[KeyPrefix+1:]

			apiKey, err := apiKeyAuth.ApiKeyAuthentication(r.Context(), prefix, keyHash)
			if err != nil || apiKey == nil {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			//Set last_used_at without blocking the authenticated request.
			go func(ctx context.Context, id pgtype.UUID) {
				updateCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
				defer cancel()

				//if not update log the error and proceed.
				if err := apiKeyAuth.UpdateVerificationTimestamp(updateCtx, id); err != nil {
					slog.Error("failed to update verification timestamp", "error", err)
				}
			}(r.Context(), apiKey.ID)

			//Set the OrgID && UserID in the Context for downstream functionalities
			//Add keys iteratively using a constant to bypass package boundaries
			r = r.WithContext(context.WithValue(r.Context(), OrgId, apiKey.OrgID))
			r = r.WithContext(context.WithValue(r.Context(), UserId, apiKey.UserID))
			next.ServeHTTP(w, r)
		})
	}
}

func verifyAuthHeader(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("missing Authorization header")
	}
	//Strip the Bearer Prefix. If it lacks, return 401 for invalid format
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", errors.New("invalid Authorization header")
	}
	return strings.TrimPrefix(authHeader, "Bearer "), nil

}
