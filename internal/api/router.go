package api

import (
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/ratifydata/ratify/internal/access"
	apiMiddleware "github.com/ratifydata/ratify/internal/api/middleware"
	"github.com/ratifydata/ratify/internal/auth"
	"github.com/ratifydata/ratify/internal/config"
	"github.com/ratifydata/ratify/internal/contract"
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
	orgTeamAccess := access.NewTeam(queries)
	members := access.NewMember(queries)

	contractApi := contract.New(pool, cfg.EncryptionKey)

	// Health check endpoint.
	r.Get("/health", healthHandler(pool))
	r.Post("/auth/login", loginHandler(authLogin))

	// Both transports use the same handlers and services with server-authenticated context.
	register := func(r chi.Router) {
		r.Post("/connections", schemaConnectionHandler(inspector))
		r.Get("/connections", listDatabaseConnections(inspector))
		r.Post("/connections/{id}/test", testConnection(inspector))
		r.Post("/teams", createTeamsConnectionHandler(orgTeamAccess))
		r.Get("/teams/{id}", getTeamConnectionHandler(orgTeamAccess))
		r.Get("/teams", listOrgTeamsConnectionHandler(orgTeamAccess))
		r.Post("/teams/{id}/members", addTeamMemberHandler(members))
		r.Get("/teams/{id}/members", listTeamMembersHandler(members))
		r.Delete("/teams/{id}/members/{userID}", removeTeamMemberHandler(members))
		r.Post("/contracts", createContract(contractApi))
		r.Post("/contracts/{id}/activate", activateContract(contractApi))
	}
	r.Route("/api/cli/v1", func(r chi.Router) {
		r.Use(apiMiddleware.ApiKeyAuthHandler(apiKey))
		register(r)
	})
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(apiMiddleware.JwtAuthMiddleware(cfg))
		register(r)
	})
	return r
}
