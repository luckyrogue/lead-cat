package authweb

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

func NewPKCE(readFull func([]byte) (int, error)) (verifier, challenge string, err error) {
	b := make([]byte, 32)
	if readFull == nil {
		readFull = rand.Read
	}
	if _, err = readFull(b); err != nil {
		return "", "", err
	}
	verifier = base64.RawURLEncoding.EncodeToString(b)
	return verifier, Challenge(verifier), nil
}

func Challenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func NewState(readFull func([]byte) (int, error)) (string, error) {
	b := make([]byte, 32)
	if readFull == nil {
		readFull = rand.Read
	}
	if _, err := readFull(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func HashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}
