package ids

import (
	"crypto/rand"
	"encoding/base64"
)

// NewID returns a URL-safe random identifier.
func NewID() (string, error) {
	return NewToken(16)
}

// NewToken returns a URL-safe random token.
func NewToken(nbytes int) (string, error) {
	b := make([]byte, nbytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
