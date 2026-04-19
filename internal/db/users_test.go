// internal/db/users_test.go
package db

import "testing"

func setupDB(t *testing.T) *DB {
	t.Helper()
	d, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	if err := d.Migrate(); err != nil {
		t.Fatal(err)
	}
	return d
}

func TestUserCRUD(t *testing.T) {
	d := setupDB(t)

	id, err := d.CreateUser("alice", "hash", true)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero id")
	}
	u, err := d.UserByUsername("alice")
	if err != nil {
		t.Fatalf("by username: %v", err)
	}
	if u.ID != id || !u.IsAdmin || u.PasswordHash != "hash" {
		t.Errorf("unexpected user: %+v", u)
	}

	if _, err := d.CreateUser("alice", "x", false); err == nil {
		t.Fatal("expected unique violation")
	}

	if err := d.UpdateUserPassword(id, "newhash"); err != nil {
		t.Fatalf("update: %v", err)
	}
	u, _ = d.UserByUsername("alice")
	if u.PasswordHash != "newhash" {
		t.Errorf("password not updated")
	}

	users, err := d.ListUsers()
	if err != nil || len(users) != 1 {
		t.Errorf("list users: %v count=%d", err, len(users))
	}

	if err := d.DeleteUser(id); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := d.UserByUsername("alice"); err == nil {
		t.Fatal("expected not found after delete")
	}
}
