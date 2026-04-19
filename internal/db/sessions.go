// internal/db/sessions.go
package db

import (
	"database/sql"
	"errors"
	"time"
)

type Session struct {
	Token     string
	UserID    int64
	ExpiresAt time.Time
}

func (d *DB) CreateSession(token string, userID int64, expiresAt time.Time) error {
	_, err := d.Exec(`INSERT INTO sessions(token,user_id,expires_at) VALUES(?,?,?)`,
		token, userID, expiresAt.UTC())
	return err
}

func (d *DB) SessionByToken(token string) (*Session, error) {
	row := d.QueryRow(`SELECT token,user_id,expires_at FROM sessions WHERE token=? AND expires_at > CURRENT_TIMESTAMP`, token)
	s := &Session{}
	err := row.Scan(&s.Token, &s.UserID, &s.ExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return s, err
}

func (d *DB) DeleteSession(token string) error {
	_, err := d.Exec(`DELETE FROM sessions WHERE token=?`, token)
	return err
}

func (d *DB) CleanupExpiredSessions() (int64, error) {
	res, err := d.Exec(`DELETE FROM sessions WHERE expires_at <= CURRENT_TIMESTAMP`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
