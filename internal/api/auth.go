package api

import (
	"context"
	"net/http"
	"strings"

	sqlc "github.com/ratifydata/ratify/internal/db/generated"
)

const KeyPrefix = 8

type apiKeyAuthenticator interface {
	ApiKeyAuthentication(ctx context.Context, prefix, keyHash string) (*sqlc.ApiKey, error)
}

// authHandler validates the request has the pre-requisite authentication headers
func authHandler(apiKeyAuth apiKeyAuthenticator) func(http.Handler) http.Handler {
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

			//Set the OrgID && UserID Headers for downstream functions
			w.Header().Set("x-org-id", apiKey.OrgID.String())
			w.Header().Set("x-user-id", apiKey.UserID.String())
			next.ServeHTTP(w, r)
		})
	}
}
