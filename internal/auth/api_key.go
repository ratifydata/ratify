package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const (
	KeyLength = 32
	// Introduce versioning for future case
)

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
