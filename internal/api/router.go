package api

import (
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	apiMiddleware "github.com/ratifydata/ratify/internal/api/middleware"
	"github.com/ratifydata/ratify/internal/auth"
	"github.com/ratifydata/ratify/internal/config"
	"github.com/ratifydata/ratify/internal/db"
	sqlc "github.com/ratifydata/ratify/internal/db/generated"
	"github.com/ratifydata/ratify/internal/schema"
)

// NewRouter creates and configures the HTTP router.
func NewRouter(pool *db.Pool, cfg *config.Config) *chi.Mux {
	r := chi.NewRouter()

	// Middleware applied to every request.
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)

	queries := sqlc.New(pool)
	authLogin := auth.NewUsernamePasswordAuth(queries)
	apiKey := auth.NewAPIKey(queries)
	inspector := schema.NewInspector(queries, cfg.EncryptionKey)

	// Health check endpoint.
	r.Get("/health", healthHandler(pool))
	r.Post("/auth/login", loginHandler(authLogin))
	r.Group(func(r chi.Router) {
		r.Use(apiMiddleware.ApiKeyAuthHandler(apiKey))
		r.Post("/api/v1/connections", schemaConnectionHandler(inspector))
		r.Get("/api/v1/connections", listDatabaseConnections(inspector))
		r.Post("/api/v1/connections/{id}/test", testConnection(inspector))
	})

	return r
}
