// Package authweb holds pure helpers for the web auth flow: PKCE, CSRF state,
// and one-way token hashing for storage. No I/O.
package authweb

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

// NewPKCE returns (verifier, challenge). readFull defaults to crypto/rand.Read.
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

// Challenge is the S256 PKCE challenge of a verifier.
func Challenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// NewState returns a url-safe random CSRF state token.
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

// HashToken returns the SHA-256 of a secret for at-rest storage (sessions,
// magic-link, invites). Compare hashes, never store raw tokens.
func HashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}
