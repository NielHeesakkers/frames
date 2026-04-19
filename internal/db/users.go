// internal/db/users.go
package db

import (
	"database/sql"
	"errors"
	"time"
)

type User struct {
	ID           int64
	Username     string
	PasswordHash string
	IsAdmin      bool
	CreatedAt    time.Time
}

var ErrNotFound = errors.New("not found")

func (d *DB) CreateUser(username, passwordHash string, isAdmin bool) (int64, error) {
	res, err := d.Exec(`INSERT INTO users(username,password_hash,is_admin) VALUES(?,?,?)`,
		username, passwordHash, isAdmin)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DB) UserByUsername(u string) (*User, error) {
	row := d.QueryRow(`SELECT id,username,password_hash,is_admin,created_at FROM users WHERE username=?`, u)
	return scanUser(row)
}

func (d *DB) UserByID(id int64) (*User, error) {
	row := d.QueryRow(`SELECT id,username,password_hash,is_admin,created_at FROM users WHERE id=?`, id)
	return scanUser(row)
}

func (d *DB) ListUsers() ([]User, error) {
	rows, err := d.Query(`SELECT id,username,password_hash,is_admin,created_at FROM users ORDER BY username`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []User
	for rows.Next() {
		u, err := scanUserRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *u)
	}
	return out, rows.Err()
}

func (d *DB) UpdateUserPassword(id int64, hash string) error {
	_, err := d.Exec(`UPDATE users SET password_hash=? WHERE id=?`, hash, id)
	return err
}

func (d *DB) DeleteUser(id int64) error {
	_, err := d.Exec(`DELETE FROM users WHERE id=?`, id)
	return err
}

type rowScanner interface {
	Scan(...any) error
}

func scanUser(r rowScanner) (*User, error) {
	u := &User{}
	err := r.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.IsAdmin, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return u, err
}

func scanUserRows(r *sql.Rows) (*User, error) {
	u := &User{}
	return u, r.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.IsAdmin, &u.CreatedAt)
}
