package middleware

import (
	"log/slog"
	"net/http"

	"github.com/ratifydata/ratify/internal/auth"
)

func jwtAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeaderValue, err := verifyAuthHeader(r)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		claims, err := auth.VerifyToken(authHeaderValue)
		if err != nil {
			slog.Error("error verifying token", "error", err)
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		//Set the OrgID && UserID Headers for downstream functions
		w.Header().Set("x-org-id", claims.OrgId.String())
		w.Header().Set("x-user-id", claims.UserId.String())
		next.ServeHTTP(w, r)

	})

}
