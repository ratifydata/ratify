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
