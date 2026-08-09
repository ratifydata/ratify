package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/ratifydata/ratify/internal/schema"
)

type inspectionValidator interface {
	SchemaInspection(ctx context.Context, params schema.ConnectionParams) error
}

type connectionLister interface {
	ListDatabaseConnections(ctx context.Context) ([]schema.StoredConnection, error)
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
