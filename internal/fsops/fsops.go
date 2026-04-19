// internal/fsops/fsops.go
package fsops

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/NielHeesakkers/frames/internal/db"
	"github.com/NielHeesakkers/frames/internal/upload"
)

type Ops struct {
	DB    *db.DB
	Root  string
	Cache interface {
		RemoveDerivatives(id int64)
	}
}

func (o *Ops) Mkdir(relPath string) error {
	abs, err := upload.SafeJoin(o.Root, relPath)
	if err != nil {
		return err
	}
	if err := os.Mkdir(abs, 0o755); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	parentRel := filepath.Dir(relPath)
	if parentRel == "." {
		parentRel = ""
	}
	parent, err := o.DB.FolderByPath(parentRel)
	if err != nil {
		return err
	}
	fi, err := os.Stat(abs)
	if err != nil {
		return err
	}
	_, err = o.DB.UpsertFolder(db.Folder{
		ParentID: &parent.ID, Path: relPath, Name: filepath.Base(relPath), Mtime: fi.ModTime().Unix(),
	})
	return err
}

func (o *Ops) RenameFile(id int64, newName string) error {
	if strings.ContainsRune(newName, '/') || strings.ContainsRune(newName, '\\') {
		return fmt.Errorf("invalid name %q", newName)
	}
	f, err := o.DB.FileByID(id)
	if err != nil {
		return err
	}
	folder, err := o.DB.FolderByID(f.FolderID)
	if err != nil {
		return err
	}
	oldAbs, err := upload.SafeJoin(o.Root, f.RelativePath)
	if err != nil {
		return err
	}
	newRel := filepath.Join(folder.Path, newName)
	newAbs, err := upload.SafeJoin(o.Root, newRel)
	if err != nil {
		return err
	}
	if _, err := os.Stat(newAbs); err == nil {
		return fmt.Errorf("destination exists")
	}
	if err := os.Rename(oldAbs, newAbs); err != nil {
		return err
	}
	_, err = o.DB.Exec(`UPDATE files SET filename=?, relative_path=? WHERE id=?`,
		newName, newRel, id)
	return err
}

func (o *Ops) MoveFile(id, newFolderID int64) error {
	f, err := o.DB.FileByID(id)
	if err != nil {
		return err
	}
	dst, err := o.DB.FolderByID(newFolderID)
	if err != nil {
		return err
	}
	oldAbs, err := upload.SafeJoin(o.Root, f.RelativePath)
	if err != nil {
		return err
	}
	newRel := filepath.Join(dst.Path, f.Filename)
	newAbs, err := upload.SafeJoin(o.Root, newRel)
	if err != nil {
		return err
	}
	if _, err := os.Stat(newAbs); err == nil {
		return fmt.Errorf("destination exists")
	}
	if err := os.Rename(oldAbs, newAbs); err != nil {
		return err
	}
	_, err = o.DB.Exec(`UPDATE files SET folder_id=?, relative_path=? WHERE id=?`,
		newFolderID, newRel, id)
	return err
}

func (o *Ops) DeleteFile(id int64) error {
	f, err := o.DB.FileByID(id)
	if err != nil {
		return err
	}
	abs, err := upload.SafeJoin(o.Root, f.RelativePath)
	if err != nil {
		return err
	}
	if err := os.Remove(abs); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := o.DB.DeleteFile(id); err != nil {
		return err
	}
	if o.Cache != nil {
		o.Cache.RemoveDerivatives(id)
	}
	return nil
}

func (o *Ops) DeleteFolder(id int64) error {
	f, err := o.DB.FolderByID(id)
	if err != nil {
		return err
	}
	if f.Path == "" {
		return fmt.Errorf("refusing to delete root")
	}
	abs, err := upload.SafeJoin(o.Root, f.Path)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(abs); err != nil {
		return err
	}
	// Collect child file IDs before cascading delete so we can scrub cache.
	var childIDs []int64
	if rows, err := o.DB.Query(
		`SELECT id FROM files WHERE folder_id = ? OR relative_path LIKE ?`,
		id, f.Path+"/%",
	); err == nil {
		for rows.Next() {
			var fid int64
			if err := rows.Scan(&fid); err == nil {
				childIDs = append(childIDs, fid)
			}
		}
		rows.Close()
	}
	if err := o.DB.DeleteFolder(id); err != nil {
		return err
	}
	if o.Cache != nil {
		for _, cid := range childIDs {
			o.Cache.RemoveDerivatives(cid)
		}
	}
	return nil
}

func (o *Ops) RenameFolder(id int64, newName string) error {
	if strings.ContainsRune(newName, '/') || strings.ContainsRune(newName, '\\') {
		return fmt.Errorf("invalid name %q", newName)
	}
	f, err := o.DB.FolderByID(id)
	if err != nil {
		return err
	}
	if f.Path == "" {
		return fmt.Errorf("refusing to rename root")
	}
	oldAbs, err := upload.SafeJoin(o.Root, f.Path)
	if err != nil {
		return err
	}
	parent := filepath.Dir(f.Path)
	if parent == "." {
		parent = ""
	}
	newRel := filepath.Join(parent, newName)
	newAbs, err := upload.SafeJoin(o.Root, newRel)
	if err != nil {
		return err
	}
	if err := os.Rename(oldAbs, newAbs); err != nil {
		return err
	}
	// Update folder row and cascade descendants' relative_path values.
	_, err = o.DB.Exec(`UPDATE folders SET path=?, name=? WHERE id=?`, newRel, newName, id)
	if err != nil {
		return err
	}
	// Update descendant folder paths and file relative_paths with the new prefix.
	oldPrefix := f.Path + "/"
	newPrefix := newRel + "/"
	_, err = o.DB.Exec(`UPDATE folders SET path = ? || substr(path, ?) WHERE path LIKE ?`,
		newPrefix, len(oldPrefix)+1, oldPrefix+"%")
	if err != nil {
		return err
	}
	_, err = o.DB.Exec(`UPDATE files SET relative_path = ? || substr(relative_path, ?) WHERE relative_path LIKE ?`,
		newPrefix, len(oldPrefix)+1, oldPrefix+"%")
	return err
}
