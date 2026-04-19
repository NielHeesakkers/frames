// internal/db/shares.go
package db

import (
	"database/sql"
	"errors"
	"time"
)

type Share struct {
	ID            int64
	Token         string
	FolderID      int64
	CreatedBy     int64
	CreatedAt     time.Time
	ExpiresAt     *time.Time
	PasswordHash  *string
	AllowDownload bool
	AllowUpload   bool
	RevokedAt     *time.Time
}

func (d *DB) CreateShare(s Share) (int64, error) {
	res, err := d.Exec(`
		INSERT INTO shares(token,folder_id,created_by,expires_at,password_hash,allow_download,allow_upload)
		VALUES(?,?,?,?,?,?,?)
	`, s.Token, s.FolderID, s.CreatedBy, s.ExpiresAt, s.PasswordHash, s.AllowDownload, s.AllowUpload)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DB) ShareByToken(token string) (*Share, error) {
	row := d.QueryRow(`
		SELECT id,token,folder_id,created_by,created_at,expires_at,password_hash,
		       allow_download,allow_upload,revoked_at
		FROM shares WHERE token=?
	`, token)
	return scanShare(row)
}

func (d *DB) ShareByID(id int64) (*Share, error) {
	row := d.QueryRow(`
		SELECT id,token,folder_id,created_by,created_at,expires_at,password_hash,
		       allow_download,allow_upload,revoked_at
		FROM shares WHERE id=?
	`, id)
	return scanShare(row)
}

func (d *DB) SharesByUser(userID int64) ([]Share, error) {
	rows, err := d.Query(`
		SELECT id,token,folder_id,created_by,created_at,expires_at,password_hash,
		       allow_download,allow_upload,revoked_at
		FROM shares WHERE created_by=? ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Share
	for rows.Next() {
		s, err := scanShare(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

func (d *DB) AllShares() ([]Share, error) {
	rows, err := d.Query(`
		SELECT id,token,folder_id,created_by,created_at,expires_at,password_hash,
		       allow_download,allow_upload,revoked_at
		FROM shares ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Share
	for rows.Next() {
		s, err := scanShare(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

func (d *DB) RevokeShare(id int64) error {
	_, err := d.Exec(`UPDATE shares SET revoked_at=CURRENT_TIMESTAMP WHERE id=?`, id)
	return err
}

func (d *DB) DeleteShare(id int64) error {
	_, err := d.Exec(`DELETE FROM shares WHERE id=?`, id)
	return err
}

func scanShare(r rowScanner) (*Share, error) {
	s := &Share{}
	var exp sql.NullTime
	var pw sql.NullString
	var rev sql.NullTime
	err := r.Scan(&s.ID, &s.Token, &s.FolderID, &s.CreatedBy, &s.CreatedAt,
		&exp, &pw, &s.AllowDownload, &s.AllowUpload, &rev)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if exp.Valid {
		s.ExpiresAt = &exp.Time
	}
	if pw.Valid {
		v := pw.String
		s.PasswordHash = &v
	}
	if rev.Valid {
		s.RevokedAt = &rev.Time
	}
	return s, nil
}
