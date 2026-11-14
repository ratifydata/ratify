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
	"encoding/base64"
	"strings"
	"testing"
)

func TestGenerateAPIKey(t *testing.T) {
	key, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	decoded, err := base64.RawURLEncoding.DecodeString(key)
	if err != nil {
		t.Fatalf("expected key to be hex encoded, got error: %v", err)
	}

	if len(decoded) != KeyLength {
		t.Errorf("expected decoded key length %d, got %d", KeyLength, len(decoded))
	}
}

func TestGenerateAPIKey_ReturnsUniqueKeys(t *testing.T) {
	firstKey, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("expected no error for first key, got %v", err)
	}

	secondKey, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("expected no error for second key, got %v", err)
	}

	if firstKey == secondKey {
		t.Fatal("expected generated API keys to be unique")
	}
}

func TestHash(t *testing.T) {
	key := "test-api-key"

	hashedKey, err := Hash(key)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if hashedKey == "" {
		t.Fatal("expected hashed key to not be empty")
	}

	if hashedKey == key {
		t.Fatal("expected hashed key to differ from original key")
	}

	if !strings.HasPrefix(hashedKey, "$2") {
		t.Errorf("expected bcrypt hash prefix, got %q", hashedKey)
	}
}

func TestVerifyAPIKey_ValidKey(t *testing.T) {
	key := "test-api-key"

	hashedKey, err := Hash(key)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if err := VerifyAPIKey(key, hashedKey); err != nil {
		t.Fatalf("expected API key to verify, got %v", err)
	}
}

func TestVerifyAPIKey_InvalidKey(t *testing.T) {
	key := "test-api-key"
	hashedKey, err := Hash(key)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if err := VerifyAPIKey("wrong-api-key", hashedKey); err == nil {
		t.Fatal("expected invalid API key to return an error")
	}
}

func TestVerifyAPIKey_InvalidHash(t *testing.T) {
	if err := VerifyAPIKey("test-api-key", "not-a-bcrypt-hash"); err == nil {
		t.Fatal("expected invalid hash to return an error")
	}
}
