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
package auth

import (
	"context"

	sqlc "github.com/ratifydata/ratify/internal/db/generated"
)

type LoginParams struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type UsernamePasswordAuth struct {
	db *sqlc.Queries
}

func NewUsernamePasswordAuth(db *sqlc.Queries) *UsernamePasswordAuth {
	return &UsernamePasswordAuth{db: db}
}

// AuthenticateUser TODO: Implement UsernamePassword Authentication. Returns a Token With User and OrgID
func (auth *UsernamePasswordAuth) AuthenticateUser(ctx context.Context, params LoginParams) (string, error) {
	return "", nil

}
