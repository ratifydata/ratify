package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ratifydata/ratify/internal/access"
	sqlc "github.com/ratifydata/ratify/internal/db/generated"
	"github.com/ratifydata/ratify/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestCreateTeamsConnectionHandler(t *testing.T) {
	queries, ctx, orgID := initQueryContext(t)
	handler := createTeamsConnectionHandler(access.NewTeam(queries))
	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/teams", strings.NewReader(`{"name":" Data Platform ","description":"Platform team"}`)).WithContext(ctx)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		// Verify the request created exactly one correctly scoped database record.
		teams, err := queries.ListTeamsByOrg(ctx, orgID)
		require.NoError(t, err)
		require.Len(t, teams, 1)
		require.True(t, teams[0].ID.Valid)
		require.Equal(t, "DATAPLATFORM", teams[0].Name)
		require.Equal(t, pgtype.Text{String: "Platform team", Valid: true}, teams[0].Description)
		assertTeamResponse(t, rec, http.StatusCreated, Response{Status: "ok", Body: access.OrgTeam{ID: teams[0].ID, Name: "DATAPLATFORM", Description: "Platform team"}})
	})
	t.Run("duplicate", func(t *testing.T) {
		_, err := queries.CreateTeam(ctx, sqlc.CreateTeamParams{OrgID: orgID, Name: "EXISTING"})
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/teams", strings.NewReader(`{"name":"Existing"}`)).WithContext(ctx)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assertTeamResponse(t, rec, http.StatusBadRequest, Response{Status: "error", Message: "team already exists"})
		teams, err := queries.ListTeamsByOrg(ctx, orgID)
		require.NoError(t, err)
		count := 0
		for _, team := range teams {
			if team.Name == "EXISTING" {
				count++
			}
		}
		require.Equal(t, 1, count)
	})
	t.Run("invalid JSON", func(t *testing.T) {
		before, err := queries.ListTeamsByOrg(ctx, orgID)
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/teams", strings.NewReader(`{`)).WithContext(ctx)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assertTeamResponse(t, rec, http.StatusBadRequest, Response{Status: "error", Message: "invalid request body"})
		after, err := queries.ListTeamsByOrg(ctx, orgID)
		require.NoError(t, err)
		require.Equal(t, before, after)
	})
	t.Run("missing organization", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/teams", strings.NewReader(`{"name":"Platform"}`))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assertTeamResponse(t, rec, http.StatusBadRequest, Response{Status: "error", Message: "OrgID missing from context"})
	})
}

func TestGetTeamConnectionHandler(t *testing.T) {
	queries, ctx, orgID := initQueryContext(t)
	saved, err := queries.CreateTeam(ctx, sqlc.CreateTeamParams{OrgID: orgID, Name: "PLATFORM", Description: pgtype.Text{String: "Platform team", Valid: true}})
	require.NoError(t, err)
	handler := getTeamConnectionHandler(access.NewTeam(queries))
	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/teams/"+saved.ID.String(), nil).WithContext(ctx)
		req.SetPathValue("id", saved.ID.String())
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assertTeamResponse(t, rec, http.StatusOK, Response{Status: "ok", Body: access.OrgTeam{ID: saved.ID, Name: saved.Name, Description: saved.Description.String}})
	})
	t.Run("missing ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/teams/", nil).WithContext(ctx)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assertTeamResponse(t, rec, http.StatusBadRequest, Response{Status: "error", Message: "invalid request"})
	})
	t.Run("invalid ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/teams/not-a-uuid", nil).WithContext(ctx)
		req.SetPathValue("id", "not-a-uuid")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assertTeamResponse(t, rec, http.StatusBadRequest, Response{Status: "error", Message: "wrong id format"})
	})
	t.Run("not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/teams/00000000-0000-0000-0000-000000000000", nil).WithContext(ctx)
		req.SetPathValue("id", "00000000-0000-0000-0000-000000000000")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assertTeamResponse(t, rec, http.StatusInternalServerError, Response{Status: "error", Message: pgx.ErrNoRows.Error()})
	})
	t.Run("missing organization", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/teams/"+saved.ID.String(), nil)
		req.SetPathValue("id", saved.ID.String())
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assertTeamResponse(t, rec, http.StatusInternalServerError, Response{Status: "error", Message: "OrgID missing from context"})
	})
}

func TestListOrgTeamsConnectionHandler(t *testing.T) {
	queries, ctx, orgID := initQueryContext(t)
	handler := listOrgTeamsConnectionHandler(access.NewTeam(queries))
	t.Run("empty", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/teams", nil).WithContext(ctx)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assertTeamResponse(t, rec, http.StatusOK, Response{Status: "ok", Body: []access.OrgTeam{}})
	})
	t.Run("success", func(t *testing.T) {
		platform, err := queries.CreateTeam(ctx, sqlc.CreateTeamParams{OrgID: orgID, Name: "PLATFORM", Description: pgtype.Text{String: "Platform team", Valid: true}})
		require.NoError(t, err)
		analytics, err := queries.CreateTeam(ctx, sqlc.CreateTeamParams{OrgID: orgID, Name: "ANALYTICS"})
		require.NoError(t, err)
		other, err := queries.CreateOrganization(ctx, sqlc.CreateOrganizationParams{Name: "Other", Slug: "other"})
		require.NoError(t, err)
		_, err = queries.CreateTeam(ctx, sqlc.CreateTeamParams{OrgID: other.ID, Name: "OTHER"})
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/teams", nil).WithContext(ctx)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assertTeamResponse(t, rec, http.StatusOK, Response{Status: "ok", Body: []access.OrgTeam{
			{ID: analytics.ID, Name: "ANALYTICS"},
			{ID: platform.ID, Name: "PLATFORM", Description: "Platform team"},
		}})
	})
	t.Run("missing organization", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/teams", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assertTeamResponse(t, rec, http.StatusInternalServerError, Response{Status: "error", Message: "OrgID missing from context"})
	})
	t.Run("database error", func(t *testing.T) {
		canceled, cancel := context.WithCancel(ctx)
		cancel()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/teams", nil).WithContext(canceled)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assertTeamResponse(t, rec, http.StatusInternalServerError, Response{Status: "error", Message: context.Canceled.Error()})
	})
}

func initQueryContext(t *testing.T) (*sqlc.Queries, context.Context, pgtype.UUID) {
	t.Helper()
	if testing.Short() {
		t.Skip("requires Postgres containers")
	}
	containers, err := testutil.InitializePostgresContainer()
	require.NoError(t, err)
	t.Cleanup(func() {
		containers.Internal.Pool.Close()
		if err := containers.External.DB.Close(); err != nil {
			t.Errorf("close external database: %v", err)
		}
		testutil.TerminateContainer(containers.Internal.Container, containers.External.Container)
	})
	queries := sqlc.New(containers.Internal.Pool)
	org, err := queries.CreateOrganization(t.Context(), sqlc.CreateOrganizationParams{Name: t.Name(), Slug: "handler-test-org"})
	require.NoError(t, err)
	return queries, context.WithValue(t.Context(), "OrgID", org.ID), org.ID
}

func assertTeamResponse(t *testing.T, rec *httptest.ResponseRecorder, status int, want Response) {
	t.Helper()
	if rec.Code != status {
		t.Errorf("status = %d, want %d", rec.Code, status)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
	// Unmarshal the whole body to reject trailing or multiple JSON responses.
	var gotJSON, wantJSON any
	if err := json.Unmarshal(rec.Body.Bytes(), &gotJSON); err != nil {
		t.Fatalf("invalid JSON response %q: %v", rec.Body.String(), err)
	}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &wantJSON); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotJSON, wantJSON) {
		t.Errorf("response = %s, want %s", rec.Body.String(), data)
	}
}
