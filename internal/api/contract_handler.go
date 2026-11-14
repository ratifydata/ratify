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

	"github.com/ratifydata/ratify/internal/contract"
)

func createContract(c *contract.ContractApi) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var params contract.ContractParams
		if json.NewDecoder(r.Body).Decode(&params) != nil {
			writeJSONResponse(w, http.StatusBadRequest, Response{Status: "error", Message: "invalid request body"})
			return
		}

		res, err := c.InitContract(r.Context(), params)
		if err != nil {
			writeHTTPError(w, err)
			return
		}

		writeJSONResponse(w, http.StatusCreated, Response{Status: "ok", Body: res})
	}
}
