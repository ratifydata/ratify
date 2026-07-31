package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/ratifydata/ratify/internal/schema"
)

type inspectionValidator interface {
	SchemaInspection(ctx context.Context, params schema.ConnectionParams) error
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
		w.WriteHeader(http.StatusOK)
		return
	}

}
