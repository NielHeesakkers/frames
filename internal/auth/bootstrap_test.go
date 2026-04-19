// internal/auth/bootstrap_test.go
package auth

import (
	"testing"

	"github.com/NielHeesakkers/frames/internal/db"
)

func TestBootstrapAdmin(t *testing.T) {
	d, err := db.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if err := d.Migrate(); err != nil {
		t.Fatal(err)
	}

	created, err := BootstrapAdmin(d, "niel", "hunter2hunter2hunter2")
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Error("expected created=true on fresh db")
	}
	u, err := d.UserByUsername("niel")
	if err != nil || !u.IsAdmin {
		t.Errorf("admin missing/not-admin: %+v err=%v", u, err)
	}

	// Second call is a no-op.
	created, err = BootstrapAdmin(d, "niel", "hunter2hunter2hunter2")
	if err != nil {
		t.Fatal(err)
	}
	if created {
		t.Error("expected created=false second time")
	}
}

func TestBootstrapAdmin_EmptyCreds(t *testing.T) {
	d, _ := db.Open(t.TempDir())
	defer d.Close()
	_ = d.Migrate()
	_, err := BootstrapAdmin(d, "", "")
	if err == nil {
		t.Error("expected error when creds empty")
	}
}
