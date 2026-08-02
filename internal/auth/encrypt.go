package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"log/slog"
)

const KEY_SIZE = 32

type EncData struct {
	CipherText []byte
	Nonce      []byte
}

func Encrypt(secret []byte, plainText string) (*EncData, error) {

	gcm, err := generateCipherBlock(secret)
	if err != nil {
		slog.Error("error creating Cipher block:")
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		slog.Error("error creating nonce")
		return nil, err
	}

	cipherText := gcm.Seal(nil, nonce, []byte(plainText), nil)
	return &EncData{
		CipherText: cipherText,
		Nonce:      nonce,
	}, nil
}

func Decrypt(nonce, encryptedText, key []byte) (string, error) {
	gcm, err := generateCipherBlock(key)
	if err != nil {
		slog.Error("error creating GCM")
		return "", err
	}
	plainText, err := gcm.Open(nil, nonce, encryptedText, nil)
	if err != nil {
		slog.Error("error decrypting ciphertext")
		return "", err
	}
	return string(plainText), nil
}

func generateCipherBlock(key []byte) (cipher.AEAD, error) {
	if len(key) != KEY_SIZE {
		return nil, fmt.Errorf("secret must be 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		slog.Error("error creating AES cipher")
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		slog.Error("error creating GCM")
		return nil, err
	}
	return gcm, nil
}
