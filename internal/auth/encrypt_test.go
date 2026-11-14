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
	"bytes"
	"strings"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := bytes.Repeat([]byte{0x42}, KEY_SIZE)

	tests := []struct {
		name      string
		plainText string
	}{
		{name: "regular text", plainText: "a secret message"},
		{name: "empty text", plainText: ""},
		{name: "unicode text", plainText: "Ratify 🔐 — siri"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := Encrypt(key, tt.plainText)
			if err != nil {
				t.Fatalf("Encrypt() error = %v", err)
			}
			if encrypted == nil {
				t.Fatal("Encrypt() returned nil data")
			}
			if len(encrypted.Nonce) == 0 {
				t.Fatal("Encrypt() returned an empty nonce")
			}
			if bytes.Equal(encrypted.CipherText, []byte(tt.plainText)) {
				t.Fatal("Encrypt() ciphertext matches plaintext")
			}

			decrypted, err := Decrypt(encrypted.Nonce, encrypted.CipherText, key)
			if err != nil {
				t.Fatalf("Decrypt() error = %v", err)
			}
			if decrypted != tt.plainText {
				t.Errorf("Decrypt() = %q, want %q", decrypted, tt.plainText)
			}
		})
	}
}

func TestEncryptInvalidKeySize(t *testing.T) {
	tests := []struct {
		name string
		key  []byte
	}{
		{name: "empty key", key: nil},
		{name: "short key", key: make([]byte, KEY_SIZE-1)},
		{name: "long key", key: make([]byte, KEY_SIZE+1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := Encrypt(tt.key, "secret")
			if err == nil {
				t.Fatal("Encrypt() error = nil, want an invalid key size error")
			}
			if encrypted != nil {
				t.Errorf("Encrypt() = %#v, want nil data", encrypted)
			}
			if !strings.Contains(err.Error(), "32 bytes") {
				t.Errorf("Encrypt() error = %q, want it to mention 32 bytes", err)
			}
		})
	}
}

func TestDecryptInvalidKeySize(t *testing.T) {
	plainText, err := Decrypt(make([]byte, 12), []byte("ciphertext"), []byte("short"))
	if err == nil {
		t.Fatal("Decrypt() error = nil, want an invalid key size error")
	}
	if plainText != "" {
		t.Errorf("Decrypt() = %q, want an empty string", plainText)
	}
}

func TestEncryptUsesUniqueNonces(t *testing.T) {
	key := bytes.Repeat([]byte{0x42}, KEY_SIZE)

	first, err := Encrypt(key, "same plaintext")
	if err != nil {
		t.Fatalf("first Encrypt() error = %v", err)
	}
	second, err := Encrypt(key, "same plaintext")
	if err != nil {
		t.Fatalf("second Encrypt() error = %v", err)
	}

	if bytes.Equal(first.Nonce, second.Nonce) {
		t.Fatal("Encrypt() reused a nonce")
	}
	if bytes.Equal(first.CipherText, second.CipherText) {
		t.Fatal("Encrypt() produced identical ciphertext for distinct nonces")
	}
}

func TestDecryptRejectsTamperedCiphertext(t *testing.T) {
	key := bytes.Repeat([]byte{0x42}, KEY_SIZE)
	encrypted, err := Encrypt(key, "a secret message")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	tampered := append([]byte(nil), encrypted.CipherText...)
	tampered[0] ^= 0xff

	plainText, err := Decrypt(encrypted.Nonce, tampered, key)
	if err == nil {
		t.Fatal("Decrypt() error = nil for tampered ciphertext")
	}
	if plainText != "" {
		t.Errorf("Decrypt() = %q, want an empty string", plainText)
	}
}

func TestDecryptRejectsWrongKey(t *testing.T) {
	key := bytes.Repeat([]byte{0x42}, KEY_SIZE)
	encrypted, err := Encrypt(key, "a secret message")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	wrongKey := bytes.Repeat([]byte{0x24}, KEY_SIZE)
	plainText, err := Decrypt(encrypted.Nonce, encrypted.CipherText, wrongKey)
	if err == nil {
		t.Fatal("Decrypt() error = nil for the wrong key")
	}
	if plainText != "" {
		t.Errorf("Decrypt() = %q, want an empty string", plainText)
	}
}
