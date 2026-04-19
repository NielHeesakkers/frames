// internal/fsops/fsops_test.go
package fsops

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/NielHeesakkers/frames/internal/db"
)

func setup(t *testing.T) (string, *db.DB, *Ops) {
	t.Helper()
	root := t.TempDir()
	d, err := db.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	if err := d.Migrate(); err != nil {
		t.Fatal(err)
	}
	_, _ = d.UpsertFolder(db.Folder{Path: "", Name: "", Mtime: 1})
	return root, d, &Ops{DB: d, Root: root}
}

func TestMkdir(t *testing.T) {
	root, d, ops := setup(t)
	if err := ops.Mkdir("Vakantie"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "Vakantie")); err != nil {
		t.Fatal("folder not on disk")
	}
	f, err := d.FolderByPath("Vakantie")
	if err != nil {
		t.Fatalf("folder not in db: %v", err)
	}
	if f.Name != "Vakantie" {
		t.Errorf("name=%q", f.Name)
	}
}

func TestRenameFile(t *testing.T) {
	root, d, ops := setup(t)
	if err := os.WriteFile(filepath.Join(root, "old.jpg"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	rf, _ := d.FolderByPath("")
	id, _ := d.InsertFile(db.File{
		FolderID: rf.ID, Filename: "old.jpg", RelativePath: "old.jpg",
		Size: 1, Mtime: 1, Kind: "image", MimeType: "image/jpeg",
	})
	if err := ops.RenameFile(id, "new.jpg"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "new.jpg")); err != nil {
		t.Fatalf("new not on disk: %v", err)
	}
	got, _ := d.FileByID(id)
	if got.Filename != "new.jpg" || got.RelativePath != "new.jpg" {
		t.Errorf("db not updated: %+v", got)
	}
}
