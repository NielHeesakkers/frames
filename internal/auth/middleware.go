// internal/auth/middleware.go
package auth

import (
	"net/http"

	"github.com/NielHeesakkers/frames/internal/db"
)

type UserLookup interface {
	SessionByToken(string) (*db.Session, error)
	UserByID(int64) (*db.User, error)
}

// RequireLogin loads the session and puts the user into the request context.
// Returns 401 if no valid session.
func RequireLogin(lookup UserLookup) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, err := r.Cookie(SessionCookieName)
			if err != nil || c.Value == "" {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			sess, err := lookup.SessionByToken(c.Value)
			if err != nil || sess == nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			u, err := lookup.UserByID(sess.UserID)
			if err != nil || u == nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			ctx := WithUser(r.Context(), CurrentUser{ID: u.ID, Username: u.Username, IsAdmin: u.IsAdmin})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAdmin must come after RequireLogin.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, ok := UserFromContext(r.Context())
		if !ok || !u.IsAdmin {
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
