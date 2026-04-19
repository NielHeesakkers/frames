// internal/upload/chunked.go
package upload

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/NielHeesakkers/frames/internal/db"
)

type Service struct {
	DB   *db.DB
	Root string
}

// StoreFile writes an uploaded body into folderPath/filename and registers the DB row.
func (s *Service) StoreFile(folderPath, filename string, body io.Reader, kindFn func(string) (string, string)) (int64, error) {
	if folderPath == "" {
		// root
	}
	abs, err := SafeJoin(s.Root, filepath.Join(folderPath, filename))
	if err != nil {
		return 0, err
	}
	if _, err := os.Stat(abs); err == nil {
		return 0, errors.New("file exists")
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return 0, err
	}
	tmp, err := os.CreateTemp(filepath.Dir(abs), ".upload-*")
	if err != nil {
		return 0, err
	}
	defer os.Remove(tmp.Name())
	n, err := io.Copy(tmp, body)
	if err != nil {
		tmp.Close()
		return 0, err
	}
	if err := tmp.Close(); err != nil {
		return 0, err
	}
	if err := os.Rename(tmp.Name(), abs); err != nil {
		return 0, err
	}
	// Register in DB.
	folder, err := s.DB.FolderByPath(folderPath)
	if err != nil {
		return 0, err
	}
	kind, mime := kindFn(filename)
	fi, _ := os.Stat(abs)
	id, err := s.DB.InsertFile(db.File{
		FolderID: folder.ID, Filename: filename,
		RelativePath: filepath.Join(folderPath, filename),
		Size:         n, Mtime: fi.ModTime().Unix(),
		Kind: kind, MimeType: mime,
	})
	if err != nil {
		return 0, fmt.Errorf("db insert: %w", err)
	}
	return id, nil
}
