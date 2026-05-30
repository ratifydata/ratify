package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/ratifydata/ratify/internal/db"
)

// NewRouter creates and configures the HTTP router.
func NewRouter(pool *db.Pool) *chi.Mux {
	r := chi.NewRouter()

	// Middleware applied to every request.
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Health check endpoint.
	r.Get("/health", healthHandler(pool))

	return r
}
