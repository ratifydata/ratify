package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

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

type LoginParams struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func NewAPIKey(db *sqlc.Queries) *APIKey {
	return &APIKey{db: db}
}

func (api *APIKey) ApiKeyAuthentication(ctx context.Context, prefix, keyHash string) (*sqlc.ApiKey, error) {
	key, err := api.db.GetAPIKeyByPrefix(ctx, prefix)
	if err != nil {
		fmt.Printf("error getting api key prefix: %v\n", err)
		return nil, err
	}

	if err = VerifyAPIKey(keyHash, key.KeyHash); err != nil {
		return nil, err
	}

	return &key, nil
}

func (api *APIKey) Login(ctx context.Context, params LoginParams) (string, error) {
	return "", fmt.Errorf("login not implemented")
}

func GenerateAPIKey() (string, error) {
	key := make([]byte, KeyLength)
	if _, err := rand.Read(key); err != nil {
		return "", err
	}
	//Encode the key to URL-safe base64
	return hex.EncodeToString(key), nil
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
