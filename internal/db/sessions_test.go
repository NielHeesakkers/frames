// internal/db/sessions_test.go
package db

import (
	"testing"
	"time"
)

func TestSessionCRUD(t *testing.T) {
	d := setupDB(t)
	uid, _ := d.CreateUser("alice", "h", false)

	tok := "tok-abc"
	exp := time.Now().Add(time.Hour)
	if err := d.CreateSession(tok, uid, exp); err != nil {
		t.Fatal(err)
	}
	s, err := d.SessionByToken(tok)
	if err != nil {
		t.Fatal(err)
	}
	if s.UserID != uid {
		t.Fatalf("uid=%d want %d", s.UserID, uid)
	}
	if err := d.DeleteSession(tok); err != nil {
		t.Fatal(err)
	}
	if _, err := d.SessionByToken(tok); err == nil {
		t.Fatal("expected not found")
	}
}

func TestSession_Expired(t *testing.T) {
	d := setupDB(t)
	uid, _ := d.CreateUser("alice", "h", false)
	tok := "tok-expired"
	if err := d.CreateSession(tok, uid, time.Now().Add(-time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := d.SessionByToken(tok); err == nil {
		t.Fatal("expected not-found for expired session")
	}
}
