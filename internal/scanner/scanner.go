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
	err = WalkDirs(ctx, s.Root, func(dir dirEntry, files []fileEntry) error {
		if tracker != nil {
			tracker.Folders.Add(1)
		}
		return s.handleDir(dir, files, full, &stats, tracker)
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
