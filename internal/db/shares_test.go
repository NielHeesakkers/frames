// internal/db/shares_test.go
package db

import (
	"testing"
	"time"
)

func TestShareCRUD(t *testing.T) {
	d := setupDB(t)
	uid, _ := d.CreateUser("alice", "h", false)
	root, _ := d.UpsertFolder(Folder{Path: "", Name: "", Mtime: 1})

	exp := time.Now().Add(30 * 24 * time.Hour)
	id, err := d.CreateShare(Share{
		Token: "abc", FolderID: root.ID, CreatedBy: uid,
		ExpiresAt: &exp, AllowDownload: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if id == 0 {
		t.Fatal("zero id")
	}
	got, err := d.ShareByToken("abc")
	if err != nil {
		t.Fatal(err)
	}
	if !got.AllowDownload {
		t.Error("allow_download not set")
	}
	if err := d.RevokeShare(id); err != nil {
		t.Fatal(err)
	}
	got, _ = d.ShareByToken("abc")
	if got.RevokedAt == nil {
		t.Error("RevokedAt not set")
	}
}
