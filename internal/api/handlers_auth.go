// internal/api/handlers_auth.go
package api

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/NielHeesakkers/frames/internal/auth"
	"github.com/NielHeesakkers/frames/internal/db"
)

type AuthDeps struct {
	DB      *db.DB
	Limiter *auth.LoginLimiter
	Secure  bool // set Secure cookies in prod
}

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (ad *AuthDeps) handleLogin(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	if !ad.Limiter.Allow(ip) {
		WriteError(w, http.StatusTooManyRequests, "too many attempts, try later")
		return
	}
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	u, err := ad.DB.UserByUsername(req.Username)
	if err != nil {
		WriteError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	ok, err := auth.VerifyPassword(u.PasswordHash, req.Password)
	if err != nil || !ok {
		WriteError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	tok, err := auth.NewToken()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "token error")
		return
	}
	if err := ad.DB.CreateSession(tok, u.ID, time.Now().Add(auth.SessionTTL)); err != nil {
		WriteError(w, http.StatusInternalServerError, "session error")
		return
	}
	auth.SetSessionCookie(w, tok, ad.Secure)
	WriteJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"id": u.ID, "username": u.Username, "is_admin": u.IsAdmin,
		},
	})
}

func (ad *AuthDeps) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(auth.SessionCookieName); err == nil && c.Value != "" {
		_ = ad.DB.DeleteSession(c.Value)
	}
	auth.ClearSessionCookie(w, ad.Secure)
	w.WriteHeader(http.StatusNoContent)
}

func handleMe(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"id": u.ID, "username": u.Username, "is_admin": u.IsAdmin,
		},
	})
}

func clientIP(r *http.Request) string {
	// Prefer X-Forwarded-For first entry when behind trusted proxy; fallback to remote addr.
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

var _ = errors.New // silence unused-import warnings in future edits
