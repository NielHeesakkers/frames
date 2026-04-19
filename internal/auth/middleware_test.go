// internal/auth/middleware_test.go
package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/NielHeesakkers/frames/internal/db"
)

func TestRequireLogin(t *testing.T) {
	d, err := db.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if err := d.Migrate(); err != nil {
		t.Fatal(err)
	}
	uid, _ := d.CreateUser("alice", "h", false)
	_ = d.CreateSession("tok", uid, time.Now().Add(time.Hour))

	h := RequireLogin(d)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, ok := UserFromContext(r.Context())
		if !ok || u.Username != "alice" {
			t.Errorf("no user in context")
		}
		w.WriteHeader(200)
	}))

	// missing cookie
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Errorf("no-cookie: code=%d", w.Code)
	}

	// valid cookie
	req = httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "tok"})
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("valid cookie: code=%d", w.Code)
	}
}
