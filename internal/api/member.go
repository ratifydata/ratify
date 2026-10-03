package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/ratifydata/ratify/internal/access"
)

func addTeamMemberHandler(member *access.Member) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := memberPathID(w, r, "id")
		if !ok {
			return
		}
		var params access.MemberParams
		if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
			writeJSONResponse(w, http.StatusBadRequest, Response{Status: "error", Message: "invalid request body"})
			return
		}
		added, err := member.AddTeamMember(r.Context(), id, params)
		if err != nil {
			writeMemberError(w, err)
			return
		}
		writeJSONResponse(w, http.StatusCreated, Response{Status: "ok", Body: added})
	}
}
func removeTeamMemberHandler(member *access.Member) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		teamID, ok := memberPathID(w, r, "id")
		if !ok {
			return
		}
		userID, ok := memberPathID(w, r, "userID")
		if !ok {
			return
		}
		if err := member.RemoveTeamMember(r.Context(), teamID, userID); err != nil {
			writeMemberError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
func listTeamMembersHandler(member *access.Member) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := memberPathID(w, r, "id")
		if !ok {
			return
		}
		members, err := member.ListTeamMembers(r.Context(), id)
		if err != nil {
			writeMemberError(w, err)
			return
		}
		writeJSONResponse(w, http.StatusOK, Response{Status: "ok", Body: members})
	}
}
func memberPathID(w http.ResponseWriter, r *http.Request, key string) (pgtype.UUID, bool) {
	value := chi.URLParam(r, key)
	if value == "" {
		value = r.PathValue(key)
	}
	var id pgtype.UUID
	if err := id.Scan(value); err != nil || !id.Valid {
		writeJSONResponse(w, http.StatusBadRequest, Response{Status: "error", Message: "invalid " + key})
		return id, false
	}
	return id, true
}
func writeMemberError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	message := "internal server error"
	switch {
	case errors.Is(err, access.ErrMemberExists):
		status = http.StatusConflict
		message = err.Error()
	case errors.Is(err, access.ErrTeamNotFound), errors.Is(err, access.ErrMemberNotFound):
		status = http.StatusNotFound
		message = err.Error()
	case errors.Is(err, access.ErrMemberEmail), errors.Is(err, access.ErrMemberOrganization):
		status = http.StatusBadRequest
		message = err.Error()
	}
	writeJSONResponse(w, status, Response{Status: "error", Message: message})
}
