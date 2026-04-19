// internal/db/folders.go
package db

import (
	"database/sql"
	"errors"
	"time"
)

type Folder struct {
	ID            int64
	ParentID      *int64
	Path          string // relative to PhotosRoot; root = ""
	Name          string
	Mtime         int64
	ItemCount     int64
	LastScannedAt *time.Time
}

func (d *DB) UpsertFolder(f Folder) (*Folder, error) {
	// Use ON CONFLICT(path) DO UPDATE pattern.
	_, err := d.Exec(`
		INSERT INTO folders(parent_id,path,name,mtime)
		VALUES(?,?,?,?)
		ON CONFLICT(path) DO UPDATE SET
		  parent_id=excluded.parent_id,
		  name=excluded.name,
		  mtime=excluded.mtime
	`, f.ParentID, f.Path, f.Name, f.Mtime)
	if err != nil {
		return nil, err
	}
	return d.FolderByPath(f.Path)
}

func (d *DB) FolderByPath(path string) (*Folder, error) {
	row := d.QueryRow(`SELECT id,parent_id,path,name,mtime,item_count,last_scanned_at FROM folders WHERE path=?`, path)
	return scanFolder(row)
}

func (d *DB) FolderByID(id int64) (*Folder, error) {
	row := d.QueryRow(`SELECT id,parent_id,path,name,mtime,item_count,last_scanned_at FROM folders WHERE id=?`, id)
	return scanFolder(row)
}

func (d *DB) ChildFolders(parentID int64) ([]Folder, error) {
	rows, err := d.Query(`SELECT id,parent_id,path,name,mtime,item_count,last_scanned_at FROM folders WHERE parent_id=? ORDER BY name`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Folder
	for rows.Next() {
		f := Folder{}
		var pid sql.NullInt64
		var ls sql.NullTime
		if err := rows.Scan(&f.ID, &pid, &f.Path, &f.Name, &f.Mtime, &f.ItemCount, &ls); err != nil {
			return nil, err
		}
		if pid.Valid {
			v := pid.Int64
			f.ParentID = &v
		}
		if ls.Valid {
			f.LastScannedAt = &ls.Time
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (d *DB) DeleteFolder(id int64) error {
	_, err := d.Exec(`DELETE FROM folders WHERE id=?`, id)
	return err
}

func (d *DB) SetFolderScanned(id int64, mtime int64, count int64) error {
	_, err := d.Exec(`UPDATE folders SET mtime=?, item_count=?, last_scanned_at=CURRENT_TIMESTAMP WHERE id=?`,
		mtime, count, id)
	return err
}

func scanFolder(r rowScanner) (*Folder, error) {
	f := &Folder{}
	var pid sql.NullInt64
	var ls sql.NullTime
	err := r.Scan(&f.ID, &pid, &f.Path, &f.Name, &f.Mtime, &f.ItemCount, &ls)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if pid.Valid {
		v := pid.Int64
		f.ParentID = &v
	}
	if ls.Valid {
		f.LastScannedAt = &ls.Time
	}
	return f, nil
}
