package api

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ratifydata/ratify/internal/access"
)

func createTeamsConnectionHandler(team *access.Team) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var params access.TeamParams
		if json.NewDecoder(r.Body).Decode(&params) != nil {
			writeJSONResponse(w, http.StatusBadRequest, Response{Status: "error", Message: "invalid request body"})
			return
		}

		createdTeam, err := team.CreateTeam(r.Context(), params)
		if err != nil {
			writeJSONResponse(w, http.StatusBadRequest, Response{Status: "error", Message: err.Error()})
			return
		}

		writeJSONResponse(w, http.StatusCreated, Response{Status: "ok", Body: createdTeam})
	}

}

func getTeamConnectionHandler(team *access.Team) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			writeJSONResponse(w, http.StatusBadRequest, Response{Status: "error", Message: "invalid request"})
			return
		}
		var teamId pgtype.UUID
		err := teamId.Scan(id)
		if err != nil {
			writeJSONResponse(w, http.StatusBadRequest, Response{Status: "error", Message: "wrong id format"})
			return
		}

		orgTeam, err := team.GetTeam(r.Context(), teamId)
		if err != nil {
			writeJSONResponse(w, http.StatusInternalServerError, Response{Status: "error", Message: err.Error()})
			return
		}
		writeJSONResponse(w, http.StatusOK, Response{Status: "ok", Body: orgTeam})

	}
}

func listOrgTeamsConnectionHandler(team *access.Team) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		teams, err := team.ListTeams(r.Context())
		if err != nil {
			writeJSONResponse(w, http.StatusInternalServerError, Response{Status: "error", Message: err.Error()})
			return
		}
		writeJSONResponse(w, http.StatusOK, Response{Status: "ok", Body: teams})
	}
}
