// internal/scanner/scanner.go
package scanner

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sync/atomic"

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

// ProgressTracker exposes live counters that handleDir will bump during the
// walk. Fields are atomic pointers so an HTTP handler can read them while
// the scan is still running.
type ProgressTracker struct {
	Folders *atomic.Int64
	Scanned *atomic.Int64
	Added   *atomic.Int64
	Updated *atomic.Int64
	Removed *atomic.Int64
}

// Scan performs one pass. If full is true, the mtime short-circuit is disabled.
func (s *Scanner) Scan(ctx context.Context, full bool) (Stats, error) {
	return s.ScanWithProgress(ctx, full, nil)
}

// seen is populated during the walk so we can garbage-collect DB folders
// that disappeared from disk (e.g. after the /photos mount changed).
type walkSeen map[string]bool

// ScanWithProgress is like Scan but reports live counters into `tracker` as
// the walk proceeds. Pass nil to disable progress tracking.
func (s *Scanner) ScanWithProgress(ctx context.Context, full bool, tracker *ProgressTracker) (Stats, error) {
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
	seen := walkSeen{}
	err = WalkDirs(ctx, s.Root, func(dir dirEntry, files []fileEntry) error {
		if tracker != nil {
			tracker.Folders.Add(1)
		}
		seen[dir.RelPath] = true
		return s.handleDir(dir, files, full, &stats, tracker)
	})

	// Garbage-collect folders that were in the DB but no longer on disk.
	// Only do this on a successful walk so partial failures don't wipe data.
	if err == nil {
		if gcRemoved, gerr := s.gcOrphanedFolders(seen, tracker); gerr == nil {
			stats.Removed += gcRemoved
		} else {
			s.Log.Warn("orphan-folder GC failed", "err", gerr)
		}
	}

	emsg := ""
	if err != nil {
		emsg = err.Error()
	}
	if fErr := s.DB.FinishScanJob(jobID, stats.Scanned, stats.Added, stats.Updated, stats.Removed, emsg); fErr != nil {
		s.Log.Warn("failed to finish scan job", "err", fErr)
	}
	return stats, err
}

// gcOrphanedFolders removes DB rows for folders not visited on this walk.
// Cascading FKs handle files + shares within them. Returns the number of
// files that were deleted via cascade (best-effort).
func (s *Scanner) gcOrphanedFolders(seen walkSeen, tracker *ProgressTracker) (int64, error) {
	rows, err := s.DB.Query(`SELECT id, path FROM folders`)
	if err != nil {
		return 0, err
	}
	type orph struct {
		id   int64
		path string
	}
	var toDelete []orph
	for rows.Next() {
		var id int64
		var path string
		if err := rows.Scan(&id, &path); err != nil {
			continue
		}
		if !seen[path] {
			toDelete = append(toDelete, orph{id, path})
		}
	}
	rows.Close()

	var removed int64
	for _, o := range toDelete {
		// Count files first so we can report what the cascade wipes out.
		var n int64
		_ = s.DB.QueryRow(`SELECT COUNT(*) FROM files WHERE folder_id=?`, o.id).Scan(&n)
		if _, err := s.DB.Exec(`DELETE FROM folders WHERE id=?`, o.id); err == nil {
			removed += n
			if tracker != nil && n > 0 {
				tracker.Removed.Add(n)
			}
			s.Log.Info("removed orphan folder", "path", o.path, "files", n)
		}
	}
	return removed, nil
}

func (s *Scanner) handleDir(dir dirEntry, files []fileEntry, full bool, stats *Stats, tracker *ProgressTracker) error {
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
	var toInsert []db.File
	var toUpdate []db.FileStatUpdate
	for _, fe := range files {
		stats.Scanned++
		if tracker != nil { tracker.Scanned.Add(1) }
		keep = append(keep, fe.Name)
		old, exists := byName[fe.Name]
		kind, mime := Classify(fe.Name)
		rel := filepath.Join(dir.RelPath, fe.Name)
		if !exists {
			toInsert = append(toInsert, db.File{
				FolderID: folderID, Filename: fe.Name, RelativePath: rel,
				Size: fe.Size, Mtime: fe.Mtime, MimeType: mime, Kind: kind,
			})
			stats.Added++
			if tracker != nil { tracker.Added.Add(1) }
			continue
		}
		if old.Mtime != fe.Mtime || old.Size != fe.Size {
			toUpdate = append(toUpdate, db.FileStatUpdate{ID: old.ID, Mtime: fe.Mtime, Size: fe.Size})
			stats.Updated++
			if tracker != nil { tracker.Updated.Add(1) }
		}
	}
	if err := s.DB.BulkInsertFiles(toInsert); err != nil {
		return err
	}
	if err := s.DB.BulkUpdateFileStats(toUpdate); err != nil {
		return err
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
	if tracker != nil && removed > 0 { tracker.Removed.Add(removed) }

	// Mark folder scanned with new mtime + count.
	return s.DB.SetFolderScanned(folderID, dir.Mtime, int64(len(keep)))
}
