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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/ratifydata/ratify/internal/access"
	sqlc "github.com/ratifydata/ratify/internal/db/generated"
	"github.com/stretchr/testify/require"
)

func TestTeamMemberHandlers(t *testing.T) {
	q, ctx, orgID := initQueryContext(t)
	team, err := q.CreateTeam(ctx, sqlc.CreateTeamParams{OrgID: orgID, Name: "TEAM", EmailAddress: "team@example.com"})
	require.NoError(t, err)
	member := access.NewMember(q)
	router := chi.NewRouter()
	router.Post("/teams/{id}/members", addTeamMemberHandler(member))
	router.Get("/teams/{id}/members", listTeamMembersHandler(member))
	router.Delete("/teams/{id}/members/{userID}", removeTeamMemberHandler(member))
	path := "/teams/" + team.ID.String() + "/members"
	request := func(method, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body)).WithContext(ctx)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}
	rec := request(http.MethodPost, path, `{"email":"member@example.com"}`)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	members, err := member.ListTeamMembers(ctx, team.ID)
	require.NoError(t, err)
	require.Len(t, members, 1)
	assertTeamResponse(t, rec, http.StatusCreated, Response{Status: "ok", Body: access.TeamMember{TeamID: team.ID, UserID: members[0].UserID, Role: "member"}})
	require.Equal(t, "member@example.com", members[0].Email)
	assertTeamResponse(t, request(http.MethodGet, path, ""), http.StatusOK, Response{Status: "ok", Body: members})
	assertTeamResponse(t, request(http.MethodPost, path, `{"email":"member@example.com"}`), http.StatusConflict, Response{Status: "error", Message: "team member already exists"})
	require.Equal(t, http.StatusBadRequest, request(http.MethodPost, path, `{`).Code)
	require.Equal(t, http.StatusBadRequest, request(http.MethodPost, path, `{"email":"invalid"}`).Code)
	require.Equal(t, http.StatusBadRequest, request(http.MethodGet, "/teams/invalid/members", "").Code)
	memberPath := path + "/" + members[0].UserID.String()
	require.Equal(t, http.StatusNoContent, request(http.MethodDelete, memberPath, "").Code)
	assertTeamResponse(t, request(http.MethodGet, path, ""), http.StatusOK, Response{Status: "ok", Body: []access.TeamMember{}})
}
