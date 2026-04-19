// internal/auth/csrf_test.go
package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCSRF_GetSetsCookie(t *testing.T) {
	h := CSRF(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("code=%d", w.Code)
	}
	if !strings.Contains(w.Header().Get("Set-Cookie"), CSRFCookieName) {
		t.Fatal("expected csrf cookie on GET")
	}
}

func TestCSRF_PostRequiresHeaderMatch(t *testing.T) {
	h := CSRF(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))

	// missing both
	req := httptest.NewRequest("POST", "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("missing: code=%d", w.Code)
	}

	// mismatched
	req = httptest.NewRequest("POST", "/", nil)
	req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: "a"})
	req.Header.Set(CSRFHeaderName, "b")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("mismatch: code=%d", w.Code)
	}

	// matched
	req = httptest.NewRequest("POST", "/", nil)
	req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: "abc"})
	req.Header.Set(CSRFHeaderName, "abc")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("match: code=%d", w.Code)
	}
}
