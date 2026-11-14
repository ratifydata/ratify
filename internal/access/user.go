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
package access

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgtype"
	sqlc "github.com/ratifydata/ratify/internal/db/generated"
	"github.com/ratifydata/ratify/internal/util"
)

type User struct {
	db *sqlc.Queries
}

type UserDetails struct {
	ID          pgtype.UUID
	Email       string
	DisplayName string
	Active      bool
}

type UserParams struct {
	Email       string `json:"email" required:"true"`
	DisplayName string `json:"display_name"`
}

func NewUser(db *sqlc.Queries) *User {
	return &User{db: db}
}

func (u User) GetUserViaMembership(ctx context.Context,
	orgId pgtype.UUID, userParams UserParams) (*UserDetails, error) {
	exist, err := u.userExists(ctx, userParams.Email, orgId)
	if err != nil {
		return nil, err
	}
	if exist {
		user, err := u.db.FetchUserByEmail(ctx, sqlc.FetchUserByEmailParams{
			Email: userParams.Email,
			OrgID: orgId,
		})
		if err != nil {
			slog.Error("failed to fetch user by email", slog.String("email", userParams.Email))
			return nil, err
		}
		return &UserDetails{
			ID:    user.ID,
			Email: user.Email,
		}, nil
	}

	return u.createUser(ctx, orgId, userParams)

}

func (u User) CreateUserViaEmail(ctx context.Context, userParams UserParams) (*UserDetails, error) {
	orgId, err := util.ValidateOrgId(ctx)
	if err != nil {
		return nil, err
	}

	exist, err := u.userExists(ctx, userParams.Email, orgId)
	if err != nil {
		return nil, err
	}
	if exist {
		return nil, fmt.Errorf("user with email %s already exists", userParams.Email)
	}

	return u.createUser(ctx, orgId, userParams)

}

func (u User) createUser(ctx context.Context, orgId pgtype.UUID, userParams UserParams) (*UserDetails, error) {

	//Create new user
	user, err := u.db.CreateUser(ctx, sqlc.CreateUserParams{
		Email:       userParams.Email,
		DisplayName: userParams.DisplayName,
		OrgID:       orgId,
		IsActive:    true,
	})

	if err != nil {
		slog.Error("error creating user")
		return nil, err
	}
	return &UserDetails{
		ID:          user.ID,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		Active:      user.IsActive,
	}, nil

}

// Checks if user exist in that Organization.
// Assumption is a user will NOT belong in multiple orgs at the same time using the same email address
func (u User) userExists(ctx context.Context, email string, orgId pgtype.UUID) (bool, error) {
	exist, err := u.db.CheckUserExistByEmail(ctx, sqlc.CheckUserExistByEmailParams{
		Email: email,
		OrgID: orgId,
	})
	if err != nil {
		slog.Error("error checking user")
		return false, err
	}
	if exist {
		slog.Info("user already exists")
		return true, nil
	}
	return false, nil
}
