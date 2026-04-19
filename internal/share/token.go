// internal/share/token.go
package share

import (
	"crypto/rand"
	"encoding/base64"
)

func NewToken() (string, error) {
	b := make([]byte, 24) // 32 base64url chars
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
