package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	sqlc "github.com/ratifydata/ratify/internal/db/generated"
)

const KeyPrefix = 8

type apiKeyAuthenticator interface {
	ApiKeyAuthentication(ctx context.Context, prefix, keyHash string) (*sqlc.ApiKey, error)
}

// apiKeyAuthHandler validates the request has the pre-requisite authentication headers
func apiKeyAuthHandler(apiKeyAuth apiKeyAuthenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			//Extracts Authorization Header and Get the value of the Bearer Token
			authHeaderValue, err := verifyAuthHeader(r)

			if err != nil {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}
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
