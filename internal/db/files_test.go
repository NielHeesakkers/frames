// internal/db/files_test.go
package db

import "testing"

func TestFileCRUD(t *testing.T) {
	d := setupDB(t)
	root, _ := d.UpsertFolder(Folder{Path: "", Name: "", Mtime: 1})

	f := File{
		FolderID: root.ID, Filename: "a.jpg", RelativePath: "a.jpg",
		Size: 1000, Mtime: 123, MimeType: "image/jpeg", Kind: "image",
	}
	id, err := d.InsertFile(f)
	if err != nil {
		t.Fatal(err)
	}
	if id == 0 {
		t.Fatal("zero id")
	}

	files, err := d.FilesInFolder(root.ID, 100, 0, SortByName)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("count=%d", len(files))
	}
	if files[0].Filename != "a.jpg" {
		t.Error("wrong file returned")
	}

	// Update by upsert-like API (for scanner diffs).
	f.Size = 2000
	f.Mtime = 456
	if err := d.UpdateFileStat(id, f.Mtime, f.Size); err != nil {
		t.Fatal(err)
	}

	// Delete.
	if err := d.DeleteFile(id); err != nil {
		t.Fatal(err)
	}
	files, _ = d.FilesInFolder(root.ID, 100, 0, SortByName)
	if len(files) != 0 {
		t.Error("expected no files after delete")
	}
}

func TestFile_UniquePath(t *testing.T) {
	d := setupDB(t)
	root, _ := d.UpsertFolder(Folder{Path: "", Name: "", Mtime: 1})
	_, err := d.InsertFile(File{FolderID: root.ID, Filename: "a.jpg", RelativePath: "a.jpg", Size: 1, Mtime: 1, Kind: "image"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = d.InsertFile(File{FolderID: root.ID, Filename: "a.jpg", RelativePath: "a.jpg", Size: 1, Mtime: 1, Kind: "image"})
	if err == nil {
		t.Fatal("expected unique violation")
	}
}
