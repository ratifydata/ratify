package middleware

import (
	"context"
	"net/http"

	"github.com/ratifydata/ratify/internal/auth"
	"github.com/ratifydata/ratify/internal/config"
)

func JwtAuthMiddleware(config *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeaderValue, err := verifyAuthHeader(r)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}
			claims, err := auth.VerifyToken(authHeaderValue, config.JWTSecret)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}
			r = r.WithContext(context.WithValue(r.Context(), OrgId, claims.OrgId))
			r = r.WithContext(context.WithValue(r.Context(), UserId, claims.UserId))
			next.ServeHTTP(w, r)

		})

	}
}
