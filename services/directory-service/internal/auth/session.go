package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

const sessionTokenBytes = 32

func GenerateSessionToken() (plain string, hash string, err error) {
	buffer := make([]byte, sessionTokenBytes)
	if _, err = rand.Read(buffer); err != nil {
		return "", "", err
	}

	plain = base64.RawURLEncoding.EncodeToString(buffer)
	return plain, HashSessionToken(plain), nil
}

func HashSessionToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
