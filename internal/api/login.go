package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/ratifydata/ratify/internal/auth"
)

const jwtCookieName = "jwt"

type loginService interface {
	Login(ctx context.Context, params auth.LoginParams) (string, error)
}

type loginResponse struct {
	Message string `json:"message"`
}

func loginHandler(service loginService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var params auth.LoginParams
		if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		token, err := service.Login(r.Context(), params)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     jwtCookieName,
			Value:    token,
			Path:     "/",
			MaxAge:   24 * 60 * 60,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   r.TLS != nil,
		})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(loginResponse{Message: "login successful"}); err != nil {
			return
		}
	}
}
