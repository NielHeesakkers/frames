// internal/api/handlers_auth_test.go
package api

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/NielHeesakkers/frames/internal/auth"
	"github.com/NielHeesakkers/frames/internal/db"
)

func testDB(t *testing.T) *db.DB {
	t.Helper()
	d, err := db.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	if err := d.Migrate(); err != nil {
		t.Fatal(err)
	}
	return d
}

func TestLoginFlow(t *testing.T) {
	d := testDB(t)
	hash, _ := auth.HashPassword("hunter2")
	_, _ = d.CreateUser("alice", hash, false)

	r := NewRouter(Deps{
		Log:     slog.Default(),
		DB:      d,
		Limiter: auth.NewLoginLimiter(5, time.Minute),
	})

	// seed csrf with GET on an /api route (CSRF middleware is mounted on /api)
	req := httptest.NewRequest("GET", "/api/me", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	csrfCookie := extractCookie(w.Header().Values("Set-Cookie"), auth.CSRFCookieName)
	if csrfCookie == "" {
		t.Fatal("no csrf cookie set")
	}

	// wrong password
	body, _ := json.Marshal(map[string]string{"username": "alice", "password": "wrong"})
	req = httptest.NewRequest("POST", "/api/login", bytes.NewReader(body))
	req.AddCookie(&http.Cookie{Name: auth.CSRFCookieName, Value: csrfCookie})
	req.Header.Set(auth.CSRFHeaderName, csrfCookie)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Errorf("wrong-pw code=%d", w.Code)
	}

	// right password
	body, _ = json.Marshal(map[string]string{"username": "alice", "password": "hunter2"})
	req = httptest.NewRequest("POST", "/api/login", bytes.NewReader(body))
	req.AddCookie(&http.Cookie{Name: auth.CSRFCookieName, Value: csrfCookie})
	req.Header.Set(auth.CSRFHeaderName, csrfCookie)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("login code=%d body=%s", w.Code, w.Body.String())
	}
	sess := extractCookie(w.Header().Values("Set-Cookie"), auth.SessionCookieName)
	if sess == "" {
		t.Fatal("no session cookie")
	}

	// /api/me with session
	req = httptest.NewRequest("GET", "/api/me", nil)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: sess})
	req.AddCookie(&http.Cookie{Name: auth.CSRFCookieName, Value: csrfCookie})
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("me code=%d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"alice"`) {
		t.Errorf("body: %s", w.Body.String())
	}
}

func extractCookie(setCookies []string, name string) string {
	for _, raw := range setCookies {
		// crude parse: name=value;...
		parts := strings.SplitN(raw, ";", 2)
		if kv := strings.SplitN(parts[0], "=", 2); len(kv) == 2 && kv[0] == name {
			return kv[1]
		}
	}
	return ""
}
