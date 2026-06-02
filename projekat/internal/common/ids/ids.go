package ids

import (
	"crypto/rand"
	"encoding/base64"
)

// NewToken returns a URL-safe random token.
func NewToken(nbytes int) (string, error) {
	b := make([]byte, nbytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
