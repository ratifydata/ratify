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
	"time"

	"github.com/ratifydata/ratify/internal/db"
)

const version = "0.1.0"

type healthResponse struct {
	Status   string `json:"status"`
	Database string `json:"database"`
	Version  string `json:"version"`
}

// healthHandler returns an HTTP handler that checks application health.
func healthHandler(pool *db.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Enforce a tight timeout constraint to prevent database bottlenecks
		// from cascading into hanging HTTP requests.
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		dbStatus := "ok"
		httpStatus := http.StatusOK

		if err := pool.Ping(ctx); err != nil {
			dbStatus = "unreachable"
			httpStatus = http.StatusServiceUnavailable
		}

		resp := healthResponse{
			Status:   statusFromHTTP(httpStatus),
			Database: dbStatus,
			Version:  version,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(httpStatus)

		if err := json.NewEncoder(w).Encode(resp); err != nil {
			return
		}
	}
}

func statusFromHTTP(code int) string {
	if code == http.StatusOK {
		return "ok"
	}
	return "degraded"
}
