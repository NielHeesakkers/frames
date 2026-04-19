// internal/auth/session.go
package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"time"
)

type ctxKey int

const userCtxKey ctxKey = 1

const (
	SessionCookieName = "frames_session"
	SessionTTL        = 30 * 24 * time.Hour
)

// NewToken returns 32 random bytes as URL-safe base64.
func NewToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func SetSessionCookie(w http.ResponseWriter, token string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(SessionTTL),
	})
}

func ClearSessionCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

type CurrentUser struct {
	ID       int64
	Username string
	IsAdmin  bool
}

func WithUser(ctx context.Context, u CurrentUser) context.Context {
	return context.WithValue(ctx, userCtxKey, u)
}

func UserFromContext(ctx context.Context) (CurrentUser, bool) {
	u, ok := ctx.Value(userCtxKey).(CurrentUser)
	return u, ok
}
