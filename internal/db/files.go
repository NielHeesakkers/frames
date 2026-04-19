// internal/db/files.go
package db

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type File struct {
	ID              int64
	FolderID        int64
	Filename        string
	RelativePath    string
	Size            int64
	Mtime           int64
	MimeType        string
	Kind            string // image | raw | video | other
	TakenAt         *time.Time
	Width           *int
	Height          *int
	CameraMake      *string
	CameraModel     *string
	Orientation     *int
	DurationMs      *int64
	ThumbStatus     string
	ThumbAttempts   int
	PreviewStatus   string
	PreviewAttempts int
}

type SortMode int

const (
	SortByName SortMode = iota
	SortByTakenAt
	SortBySize
)

func (d *DB) InsertFile(f File) (int64, error) {
	res, err := d.Exec(`
		INSERT INTO files(folder_id,filename,relative_path,size,mtime,mime_type,kind)
		VALUES(?,?,?,?,?,?,?)
	`, f.FolderID, f.Filename, f.RelativePath, f.Size, f.Mtime, f.MimeType, f.Kind)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DB) UpdateFileStat(id, mtime, size int64) error {
	_, err := d.Exec(`UPDATE files SET mtime=?, size=?, thumb_status='pending', preview_status='pending', thumb_attempts=0, preview_attempts=0 WHERE id=?`,
		mtime, size, id)
	return err
}

func (d *DB) UpdateFileMetadata(id int64, m MetadataUpdate) error {
	_, err := d.Exec(`
		UPDATE files SET
		  taken_at=?, width=?, height=?, camera_make=?, camera_model=?,
		  orientation=?, duration_ms=?, mime_type=COALESCE(?, mime_type)
		WHERE id=?
	`, m.TakenAt, m.Width, m.Height, m.CameraMake, m.CameraModel, m.Orientation, m.DurationMs, m.MimeType, id)
	return err
}

type MetadataUpdate struct {
	TakenAt     *time.Time
	Width       *int
	Height      *int
	CameraMake  *string
	CameraModel *string
	Orientation *int
	DurationMs  *int64
	MimeType    *string
}

func (d *DB) FileByID(id int64) (*File, error) {
	row := d.QueryRow(fileSelect + ` WHERE id=?`, id)
	return scanFile(row)
}

func (d *DB) FilesInFolder(folderID int64, limit, offset int, sort SortMode) ([]File, error) {
	order := "filename"
	switch sort {
	case SortByTakenAt:
		order = "COALESCE(taken_at, datetime(mtime, 'unixepoch')) DESC, filename"
	case SortBySize:
		order = "size DESC, filename"
	}
	q := fileSelect + ` WHERE folder_id=? ORDER BY ` + order + ` LIMIT ? OFFSET ?`
	rows, err := d.Query(q, folderID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []File
	for rows.Next() {
		f, err := scanFileRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *f)
	}
	return out, rows.Err()
}

func (d *DB) PendingThumbs(limit int) ([]File, error) {
	rows, err := d.Query(fileSelect+` WHERE thumb_status='pending' AND thumb_attempts < 3 ORDER BY id LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []File
	for rows.Next() {
		f, err := scanFileRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *f)
	}
	return out, rows.Err()
}

func (d *DB) SetThumbStatus(id int64, status string, bumpAttempts bool) error {
	if bumpAttempts {
		_, err := d.Exec(`UPDATE files SET thumb_status=?, thumb_attempts=thumb_attempts+1 WHERE id=?`, status, id)
		return err
	}
	_, err := d.Exec(`UPDATE files SET thumb_status=? WHERE id=?`, status, id)
	return err
}

func (d *DB) SetPreviewStatus(id int64, status string, bumpAttempts bool) error {
	if bumpAttempts {
		_, err := d.Exec(`UPDATE files SET preview_status=?, preview_attempts=preview_attempts+1 WHERE id=?`, status, id)
		return err
	}
	_, err := d.Exec(`UPDATE files SET preview_status=? WHERE id=?`, status, id)
	return err
}

func (d *DB) DeleteFile(id int64) error {
	_, err := d.Exec(`DELETE FROM files WHERE id=?`, id)
	return err
}

func (d *DB) DeleteFilesByFolder(folderID int64, keepFilenames []string) error {
	// Delete files not in keep list. Use IN with rebound parameters; at scale we
	// expect keepFilenames to fit (per-folder, not per-library).
	if len(keepFilenames) == 0 {
		_, err := d.Exec(`DELETE FROM files WHERE folder_id=?`, folderID)
		return err
	}
	placeholders := ""
	args := []any{folderID}
	for i, n := range keepFilenames {
		if i > 0 {
			placeholders += ","
		}
		placeholders += "?"
		args = append(args, n)
	}
	q := fmt.Sprintf(`DELETE FROM files WHERE folder_id=? AND filename NOT IN (%s)`, placeholders)
	_, err := d.Exec(q, args...)
	return err
}

const fileSelect = `
SELECT id, folder_id, filename, relative_path, size, mtime, mime_type, kind,
       taken_at, width, height, camera_make, camera_model, orientation, duration_ms,
       thumb_status, thumb_attempts, preview_status, preview_attempts
FROM files`

func scanFile(r rowScanner) (*File, error) {
	f := &File{}
	var takenAt sql.NullTime
	var w, h, orient sql.NullInt64
	var make_, model sql.NullString
	var dur sql.NullInt64
	err := r.Scan(&f.ID, &f.FolderID, &f.Filename, &f.RelativePath, &f.Size, &f.Mtime, &f.MimeType, &f.Kind,
		&takenAt, &w, &h, &make_, &model, &orient, &dur,
		&f.ThumbStatus, &f.ThumbAttempts, &f.PreviewStatus, &f.PreviewAttempts)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if takenAt.Valid {
		f.TakenAt = &takenAt.Time
	}
	if w.Valid {
		v := int(w.Int64)
		f.Width = &v
	}
	if h.Valid {
		v := int(h.Int64)
		f.Height = &v
	}
	if orient.Valid {
		v := int(orient.Int64)
		f.Orientation = &v
	}
	if make_.Valid {
		v := make_.String
		f.CameraMake = &v
	}
	if model.Valid {
		v := model.String
		f.CameraModel = &v
	}
	if dur.Valid {
		v := dur.Int64
		f.DurationMs = &v
	}
	return f, nil
}

func scanFileRows(r *sql.Rows) (*File, error) { return scanFile(r) }
