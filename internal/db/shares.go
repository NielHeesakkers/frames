// internal/db/shares.go
package db

import (
	"database/sql"
	"encoding/json"
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
	// FileIDs narrows a share to a specific set of files (by ID). When nil/
	// empty the share covers the whole folder subtree as before.
	FileIDs []int64
}

// encodeFileIDs returns NULL when the slice is empty, otherwise a JSON array.
func encodeFileIDs(ids []int64) any {
	if len(ids) == 0 {
		return nil
	}
	b, _ := json.Marshal(ids)
	return string(b)
}

func (d *DB) CreateShare(s Share) (int64, error) {
	res, err := d.Exec(`
		INSERT INTO shares(token,folder_id,created_by,expires_at,password_hash,
		                   allow_download,allow_upload,file_ids)
		VALUES(?,?,?,?,?,?,?,?)
	`, s.Token, s.FolderID, s.CreatedBy, s.ExpiresAt, s.PasswordHash,
		s.AllowDownload, s.AllowUpload, encodeFileIDs(s.FileIDs))
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

const shareSelect = `
SELECT id,token,folder_id,created_by,created_at,expires_at,password_hash,
       allow_download,allow_upload,revoked_at,file_ids
FROM shares`

func (d *DB) ShareByToken(token string) (*Share, error) {
	row := d.QueryRow(shareSelect+` WHERE token=?`, token)
	return scanShare(row)
}

func (d *DB) ShareByID(id int64) (*Share, error) {
	row := d.QueryRow(shareSelect+` WHERE id=?`, id)
	return scanShare(row)
}

func (d *DB) SharesByUser(userID int64) ([]Share, error) {
	rows, err := d.Query(shareSelect+` WHERE created_by=? ORDER BY created_at DESC`, userID)
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
	rows, err := d.Query(shareSelect + ` ORDER BY created_at DESC`)
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
	var fids sql.NullString
	err := r.Scan(&s.ID, &s.Token, &s.FolderID, &s.CreatedBy, &s.CreatedAt,
		&exp, &pw, &s.AllowDownload, &s.AllowUpload, &rev, &fids)
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
	if fids.Valid && fids.String != "" {
		var ids []int64
		if jErr := json.Unmarshal([]byte(fids.String), &ids); jErr == nil {
			s.FileIDs = ids
		}
	}
	return s, nil
}
