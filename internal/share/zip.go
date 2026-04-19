// internal/share/zip.go
package share

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/NielHeesakkers/frames/internal/db"
)

// StreamFolderZip writes a ZIP archive of folder + all descendants into w, streaming.
// It does not buffer the whole archive in memory.
func StreamFolderZip(w io.Writer, d *db.DB, root, rootPath string) error {
	zw := zip.NewWriter(w)
	defer zw.Close()

	// Walk files in DB rooted at rootPath.
	rows, err := d.Query(`
		SELECT id, relative_path FROM files
		WHERE relative_path = ? OR relative_path LIKE ?
		ORDER BY relative_path
	`, rootPath, rootPath+"/%")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var rel string
		if err := rows.Scan(&id, &rel); err != nil {
			return err
		}
		abs := filepath.Join(root, rel)
		// Strip the rootPath prefix from entry name for nicer ZIP layout.
		entryName := strings.TrimPrefix(rel, rootPath+"/")
		if entryName == rel {
			entryName = filepath.Base(rel)
		}
		if err := addFile(zw, abs, entryName); err != nil {
			return err
		}
	}
	return rows.Err()
}

func addFile(zw *zip.Writer, abs, entryName string) error {
	fh, err := os.Open(abs)
	if err != nil {
		return err
	}
	defer fh.Close()
	fi, err := fh.Stat()
	if err != nil {
		return err
	}
	hdr, err := zip.FileInfoHeader(fi)
	if err != nil {
		return err
	}
	hdr.Name = entryName
	hdr.Method = zip.Deflate
	iw, err := zw.CreateHeader(hdr)
	if err != nil {
		return err
	}
	_, err = io.Copy(iw, fh)
	return err
}
