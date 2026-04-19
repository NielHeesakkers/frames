// internal/scanner/scanner.go
package scanner

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/NielHeesakkers/frames/internal/db"
)

type Scanner struct {
	DB   *db.DB
	Log  *slog.Logger
	Root string
}

type Stats struct {
	Scanned, Added, Updated, Removed int64
}

// Scan performs one pass. If full is true, the mtime short-circuit is disabled.
func (s *Scanner) Scan(ctx context.Context, full bool) (Stats, error) {
	kind := "incremental"
	if full {
		kind = "full"
	}
	jobID, err := s.DB.StartScanJob(kind)
	if err != nil {
		return Stats{}, err
	}

	// Ensure the root folder row exists before walking.
	if _, err := s.DB.FolderByPath(""); err == db.ErrNotFound {
		fi, serr := os.Stat(s.Root)
		if serr != nil {
			_ = s.DB.FinishScanJob(jobID, 0, 0, 0, 0, serr.Error())
			return Stats{}, serr
		}
		if _, uerr := s.DB.UpsertFolder(db.Folder{Path: "", Name: "", Mtime: fi.ModTime().Unix()}); uerr != nil {
			_ = s.DB.FinishScanJob(jobID, 0, 0, 0, 0, uerr.Error())
			return Stats{}, uerr
		}
	}

	var stats Stats
	err = WalkDirs(ctx, s.Root, func(dir dirEntry, files []fileEntry) error {
		return s.handleDir(dir, files, full, &stats)
	})
	emsg := ""
	if err != nil {
		emsg = err.Error()
	}
	if fErr := s.DB.FinishScanJob(jobID, stats.Scanned, stats.Added, stats.Updated, stats.Removed, emsg); fErr != nil {
		s.Log.Warn("failed to finish scan job", "err", fErr)
	}
	return stats, err
}

func (s *Scanner) handleDir(dir dirEntry, files []fileEntry, full bool, stats *Stats) error {
	// Ensure folder row exists; determine parent.
	var parentID *int64
	if dir.RelPath != "" {
		parentRel := filepath.Dir(dir.RelPath)
		if parentRel == "." {
			parentRel = ""
		}
		parent, err := s.DB.FolderByPath(parentRel)
		if err != nil {
			return err
		}
		parentID = &parent.ID
	}
	existing, err := s.DB.FolderByPath(dir.RelPath)
	var folderID int64
	switch {
	case err == nil:
		folderID = existing.ID
		// Short-circuit if mtime unchanged AND not a full scan.
		if !full && existing.Mtime == dir.Mtime {
			return nil
		}
	case err == db.ErrNotFound:
		created, cerr := s.DB.UpsertFolder(db.Folder{
			ParentID: parentID, Path: dir.RelPath, Name: dir.Name, Mtime: dir.Mtime,
		})
		if cerr != nil {
			return cerr
		}
		folderID = created.ID
	default:
		return err
	}

	// Pull existing files for this folder, paginating so folders with
	// more than the batch size aren't silently truncated.
	const batch = 10000
	var existingFiles []db.File
	offset := 0
	for {
		chunk, err := s.DB.FilesInFolder(folderID, batch, offset, db.SortByName)
		if err != nil {
			return err
		}
		existingFiles = append(existingFiles, chunk...)
		if len(chunk) < batch {
			break
		}
		offset += batch
	}
	if len(existingFiles) > 50000 {
		s.Log.Warn("large folder", "path", dir.RelPath, "count", len(existingFiles))
	}
	byName := make(map[string]db.File, len(existingFiles))
	for _, f := range existingFiles {
		byName[f.Filename] = f
	}

	keep := make([]string, 0, len(files))
	for _, fe := range files {
		stats.Scanned++
		keep = append(keep, fe.Name)
		old, exists := byName[fe.Name]
		kind, mime := Classify(fe.Name)
		rel := filepath.Join(dir.RelPath, fe.Name)
		if !exists {
			_, err := s.DB.InsertFile(db.File{
				FolderID: folderID, Filename: fe.Name, RelativePath: rel,
				Size: fe.Size, Mtime: fe.Mtime, MimeType: mime, Kind: kind,
			})
			if err != nil {
				return err
			}
			stats.Added++
			continue
		}
		if old.Mtime != fe.Mtime || old.Size != fe.Size {
			if err := s.DB.UpdateFileStat(old.ID, fe.Mtime, fe.Size); err != nil {
				return err
			}
			stats.Updated++
		}
	}
	// Remove rows for files no longer present.
	keepSet := make(map[string]struct{}, len(keep))
	for _, n := range keep {
		keepSet[n] = struct{}{}
	}
	var removed int64
	for name := range byName {
		if _, ok := keepSet[name]; !ok {
			removed++
		}
	}
	if err := s.DB.DeleteFilesByFolder(folderID, keep); err != nil {
		return err
	}
	stats.Removed += removed

	// Mark folder scanned with new mtime + count.
	return s.DB.SetFolderScanned(folderID, dir.Mtime, int64(len(keep)))
}
