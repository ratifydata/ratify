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
	exist, err := u.userExists(ctx, userParams.Email)
	if err != nil {
		return nil, err
	}
	if exist {
		user, err := u.db.FetchUserByEmail(ctx, userParams.Email)
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

	exist, err := u.userExists(ctx, userParams.Email)
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
func (u User) userExists(ctx context.Context, email string) (bool, error) {
	exist, err := u.db.CheckUserExistByEmail(ctx, email)
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
