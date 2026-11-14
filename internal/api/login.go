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
	"context"
	"encoding/json"
	"net/http"

	"github.com/ratifydata/ratify/internal/auth"
)

const jwtCookieName = "jwt"

type loginService interface {
	AuthenticateUser(ctx context.Context, params auth.LoginParams) (string, error)
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

		token, err := service.AuthenticateUser(r.Context(), params)
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
