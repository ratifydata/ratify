package api

import (
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	apiMiddleware "github.com/ratifydata/ratify/internal/api/middleware"
	"github.com/ratifydata/ratify/internal/auth"
	"github.com/ratifydata/ratify/internal/db"
	sqlc "github.com/ratifydata/ratify/internal/db/generated"
	"github.com/ratifydata/ratify/internal/schema"
)

// NewRouter creates and configures the HTTP router.
func NewRouter(pool *db.Pool) *chi.Mux {
	r := chi.NewRouter()

	// Middleware applied to every request.
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)

	queries := sqlc.New(pool)
	authLogin := auth.NewUsernamePasswordAuth(queries)
	apiKey := auth.NewAPIKey(queries)
	inspector := schema.NewInspector(queries)

	// Health check endpoint.
	r.Get("/health", healthHandler(pool))
	r.Post("/auth/login", loginHandler(authLogin))
	r.Group(func(r chi.Router) {
		r.Use(apiMiddleware.ApiKeyAuthHandler(apiKey))
		r.Post("/api/v1/connections", schemaConnectionHandler(inspector))
	})

	return r
}
