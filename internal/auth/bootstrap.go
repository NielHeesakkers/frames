// internal/auth/bootstrap.go
package auth

import (
	"errors"

	"github.com/NielHeesakkers/frames/internal/db"
)

// BootstrapAdmin creates the admin user if none exists. Returns (created, error).
// Returns an error if there is no admin AND the supplied creds are empty.
func BootstrapAdmin(d *db.DB, username, password string) (bool, error) {
	users, err := d.ListUsers()
	if err != nil {
		return false, err
	}
	for _, u := range users {
		if u.IsAdmin {
			return false, nil
		}
	}
	if username == "" || password == "" {
		return false, errors.New("no admin exists; set FRAMES_ADMIN_USERNAME + FRAMES_ADMIN_PASSWORD on first run")
	}
	hash, err := HashPassword(password)
	if err != nil {
		return false, err
	}
	if _, err := d.CreateUser(username, hash, true); err != nil {
		return false, err
	}
	return true, nil
}
