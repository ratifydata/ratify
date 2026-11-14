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
	"encoding/json"
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
			writeHTTPError(w, err)
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
			writeHTTPError(w, err)
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
			writeHTTPError(w, err)
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
