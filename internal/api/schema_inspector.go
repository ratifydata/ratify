package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

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
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
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

		response, err := json.Marshal(connections)
		if err != nil {
			slog.Error("failed to encode database connections", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write(response); err != nil {
			slog.Error("failed to write database connections response", "error", err)
		}
	}

}

func testConnection(inspector connectionTester) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
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

func writeJSONResponse(w http.ResponseWriter, statusCode int, response Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}
