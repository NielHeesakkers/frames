// internal/auth/csrf.go
package auth

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
)

const (
	CSRFCookieName = "frames_csrf"
	CSRFHeaderName = "X-CSRF-Token"
)

// CSRF enforces the double-submit cookie pattern:
//   - Safe methods (GET/HEAD/OPTIONS) always pass and seed the cookie if missing.
//   - Unsafe methods (POST/PUT/PATCH/DELETE) require the cookie value to
//     equal the value sent in the X-CSRF-Token header.
func CSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, _ := r.Cookie(CSRFCookieName)
		if c == nil || c.Value == "" {
			tok, _ := newCSRFToken()
			http.SetCookie(w, &http.Cookie{
				Name: CSRFCookieName, Value: tok,
				Path: "/", SameSite: http.SameSiteLaxMode,
				// NOT HttpOnly — the JS client must read it to set the header
			})
			c = &http.Cookie{Value: tok}
		}

		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}

		hdr := r.Header.Get(CSRFHeaderName)
		if hdr == "" || hdr != c.Value {
			http.Error(w, `{"error":"csrf mismatch"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func newCSRFToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
