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

// StreamFilesZip writes a ZIP with only the named files into w, streaming.
// Entry names are the basename of each file (no folder hierarchy is preserved
// because file-scoped shares aren't tied to a single parent).
func StreamFilesZip(w io.Writer, d *db.DB, root string, ids []int64) error {
	zw := zip.NewWriter(w)
	defer zw.Close()
	seen := make(map[string]int, len(ids))
	for _, id := range ids {
		var rel string
		if err := d.QueryRow(`SELECT relative_path FROM files WHERE id=?`, id).Scan(&rel); err != nil {
			continue
		}
		abs := filepath.Join(root, rel)
		entryName := filepath.Base(rel)
		// Avoid duplicate entry names when two selected files share a name
		// in different folders: append "(n)" to the second, "(2)" to the third, …
		if n := seen[entryName]; n > 0 {
			ext := filepath.Ext(entryName)
			base := strings.TrimSuffix(entryName, ext)
			entryName = base + " (" + itoa(n+1) + ")" + ext
		}
		seen[filepath.Base(rel)]++
		if err := addFile(zw, abs, entryName); err != nil {
			return err
		}
	}
	return nil
}

func itoa(n int) string {
	// tiny helper to avoid importing strconv just for this
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n/10) + string(rune('0'+n%10))
}

func addFile(zw *zip.Writer, abs, entryName string) error {
	fh, err := os.Open(abs)
	if err != nil {
		// File vanished between scan and zip; skip rather than abort archive.
		return nil
	}
	defer fh.Close()
	fi, err := fh.Stat()
	if err != nil {
		return nil
	}
	hdr, err := zip.FileInfoHeader(fi)
	if err != nil {
		return nil
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
