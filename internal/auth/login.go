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
