// internal/scanner/walker.go
package scanner

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type dirEntry struct {
	AbsPath string
	RelPath string
	Name    string
	Mtime   int64
}

type fileEntry struct {
	AbsPath string
	Name    string
	Size    int64
	Mtime   int64
}

// ignoredPrefixes lists name prefixes we skip entirely (OS dotfiles, thumbnail sidecars).
var ignoredPrefixes = []string{".DS_Store", ".", "@eaDir", "Thumbs.db"}

// ignoredSuffixes lists file-name suffixes (case-insensitive) we hide from users.
// XMP sidecars sit next to RAW files and are metadata, not photos.
var ignoredSuffixes = []string{".xmp", ".thm"}

func isIgnored(name string) bool {
	for _, p := range ignoredPrefixes {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	lower := strings.ToLower(name)
	for _, s := range ignoredSuffixes {
		if strings.HasSuffix(lower, s) {
			return true
		}
	}
	return false
}

// WalkDirs invokes onDir for every directory under root (including root).
// onDir receives the directory entry and a listing of its immediate files.
// Returns early on ctx cancellation.
func WalkDirs(ctx context.Context, root string, onDir func(dirEntry, []fileEntry) error) error {
	return filepath.WalkDir(root, func(p string, de fs.DirEntry, werr error) error {
		if werr != nil {
			return werr
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if isIgnored(de.Name()) && p != root {
			if de.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !de.IsDir() {
			return nil
		}
		info, err := os.Stat(p)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		if rel == "." {
			rel = ""
		}
		// List files in this dir.
		entries, err := os.ReadDir(p)
		if err != nil {
			return err
		}
		var files []fileEntry
		for _, e := range entries {
			if e.IsDir() || isIgnored(e.Name()) {
				continue
			}
			fi, err := e.Info()
			if err != nil {
				continue
			}
			files = append(files, fileEntry{
				AbsPath: filepath.Join(p, e.Name()),
				Name:    e.Name(),
				Size:    fi.Size(),
				Mtime:   fi.ModTime().Unix(),
			})
		}
		return onDir(dirEntry{
			AbsPath: p, RelPath: rel, Name: de.Name(), Mtime: info.ModTime().UnixNano(),
		}, files)
	})
}
