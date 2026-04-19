// internal/db/folder_shares.go
package db

import "database/sql"

type FolderShare struct {
	FolderID         int64
	SharedWithUserID int64
	SharedBy         int64
}

func (d *DB) AddFolderShare(folderID, sharedWith, sharedBy int64) error {
	_, err := d.Exec(`
		INSERT OR IGNORE INTO folder_shares(folder_id, shared_with_user_id, shared_by)
		VALUES(?,?,?)
	`, folderID, sharedWith, sharedBy)
	return err
}

func (d *DB) RemoveFolderShare(folderID, sharedWith int64) error {
	_, err := d.Exec(`DELETE FROM folder_shares WHERE folder_id=? AND shared_with_user_id=?`, folderID, sharedWith)
	return err
}

func (d *DB) FoldersSharedWith(userID int64) ([]Folder, error) {
	rows, err := d.Query(`
		SELECT f.id, f.parent_id, f.path, f.name, f.mtime, f.item_count, f.last_scanned_at
		FROM folders f
		JOIN folder_shares s ON s.folder_id = f.id
		WHERE s.shared_with_user_id = ? ORDER BY f.name
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Folder
	for rows.Next() {
		var f Folder
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
