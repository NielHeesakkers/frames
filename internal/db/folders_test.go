// internal/db/folders_test.go
package db

import "testing"

func TestFolderUpsertAndTree(t *testing.T) {
	d := setupDB(t)

	root, err := d.UpsertFolder(Folder{Path: "", Name: "", Mtime: 1, ParentID: nil})
	if err != nil {
		t.Fatal(err)
	}
	kid, err := d.UpsertFolder(Folder{Path: "2024", Name: "2024", Mtime: 2, ParentID: &root.ID})
	if err != nil {
		t.Fatal(err)
	}

	// Lookup.
	got, err := d.FolderByPath("2024")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != kid.ID || *got.ParentID != root.ID {
		t.Errorf("tree mismatch: %+v", got)
	}

	// Update path via upsert (same path keeps id).
	same, err := d.UpsertFolder(Folder{Path: "2024", Name: "2024", Mtime: 99, ParentID: &root.ID})
	if err != nil {
		t.Fatal(err)
	}
	if same.ID != kid.ID {
		t.Errorf("expected same id after upsert; got %d vs %d", same.ID, kid.ID)
	}

	// List children.
	children, err := d.ChildFolders(root.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(children) != 1 {
		t.Errorf("children=%d", len(children))
	}
}

func TestDeleteFolderCascades(t *testing.T) {
	d := setupDB(t)
	root, _ := d.UpsertFolder(Folder{Path: "", Name: "", Mtime: 1})
	_, _ = d.UpsertFolder(Folder{Path: "A", Name: "A", Mtime: 1, ParentID: &root.ID})
	if err := d.DeleteFolder(root.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := d.FolderByPath("A"); err == nil {
		t.Error("expected child to be deleted via cascade")
	}
}
