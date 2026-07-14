package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgtype"
	sqlc "github.com/ratifydata/ratify/internal/db/generated"
	"golang.org/x/crypto/bcrypt"
)

const (
	KeyLength = 32
	// Introduce versioning for future case
)

type APIKey struct {
	db *sqlc.Queries
}

func NewAPIKey(db *sqlc.Queries) *APIKey {
	return &APIKey{db: db}
}

func (api *APIKey) ApiKeyAuthentication(ctx context.Context, prefix, keyHash string) (*sqlc.ApiKey, error) {
	params := sqlc.GetAPIKeyByPrefixParams{
		KeyPrefix: prefix,
		IsActive:  true,
	}
	key, err := api.db.GetAPIKeyByPrefix(ctx, params)
	if err != nil {
		slog.Error("failed to get api key by prefix")
		return nil, err
	}

	if err = VerifyAPIKey(keyHash, key.KeyHash); err != nil {
		return nil, err
	}

	return &key, nil
}

func (api *APIKey) UpdateVerificationTimestamp(ctx context.Context, id pgtype.UUID) error {
	_, err := api.db.UpdateAPIKeyLastUsed(ctx, id)
	if err != nil {
		slog.Error("failed to update api key last used")
		return err
	}
	return nil
}

func GenerateAPIKey() (string, error) {
	key := make([]byte, KeyLength)
	if _, err := rand.Read(key); err != nil {
		return "", err
	}
	//Encode the key to URL-safe base64
	return base64.RawURLEncoding.EncodeToString(key), nil
}

func Hash(key string) (string, error) {
	//Generate random salt
	hashedKey, err := bcrypt.GenerateFromPassword([]byte(key), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("error hashing password: %w", err)
	}
	return string(hashedKey), nil

}

func VerifyAPIKey(key, secured string) error {
	//Hash Key them compare
	err := bcrypt.CompareHashAndPassword([]byte(secured), []byte(key))
	if err != nil {
		return err
	}
	return nil
}
