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
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ratifydata/ratify/internal/schema"
)

type inspectionValidator interface {
	SchemaInspection(ctx context.Context, params schema.ConnectionParams) error
}

type connectionLister interface {
	ListDatabaseConnections(ctx context.Context) ([]schema.StoredConnection, error)
}

type connectionTester interface {
	TestConnection(ctx context.Context, id pgtype.UUID) error
}

type Response struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Body    interface{} `json:"body,omitempty"`
}

func schemaConnectionHandler(inspector inspectionValidator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var params schema.ConnectionParams
		if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		err := inspector.SchemaInspection(r.Context(), params)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSONResponse(w, http.StatusOK, Response{Status: "ok"})
	}

}

func listDatabaseConnections(inspector connectionLister) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		connections, err := inspector.ListDatabaseConnections(r.Context())
		if err != nil {
			slog.Error("failed to list database connections", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if connections == nil {
			connections = []schema.StoredConnection{}
		}

		writeJSONResponse(w, http.StatusOK, Response{Status: "ok", Body: connections})
	}

}

func testConnection(inspector connectionTester) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "" {
			id = r.PathValue("id")
		}
		if id == "" {
			writeJSONResponse(w, http.StatusBadRequest, Response{Status: "err", Message: "no connection id"})
			return
		}
		var connId pgtype.UUID
		err := connId.Scan(id)
		if err != nil {
			slog.Error("invalid database connection id", "error", err)
			writeJSONResponse(w, http.StatusBadRequest, Response{Status: "err", Message: "invalid connection id"})
			return
		}

		err = inspector.TestConnection(r.Context(), connId)
		if err != nil {
			writeJSONResponse(w, http.StatusInternalServerError, Response{Status: "err", Message: err.Error()})
			return
		}

		writeJSONResponse(w, http.StatusOK, Response{Status: "ok"})
	}

}
