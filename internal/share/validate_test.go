// internal/share/validate_test.go
package share

import (
	"testing"
	"time"

	"github.com/NielHeesakkers/frames/internal/db"
)

func TestValidateActive(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	cases := []struct {
		name   string
		s      db.Share
		wantOK bool
		want   Status
	}{
		{"active", db.Share{}, true, StatusActive},
		{"revoked", db.Share{RevokedAt: &past}, false, StatusRevoked},
		{"expired", db.Share{ExpiresAt: &past}, false, StatusExpired},
		{"future-expiry", db.Share{ExpiresAt: &future}, true, StatusActive},
	}
	for _, c := range cases {
		st := Validate(&c.s)
		if (st == StatusActive) != c.wantOK || st != c.want {
			t.Errorf("%s: got %v", c.name, st)
		}
	}
}
