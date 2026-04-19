// internal/share/validate.go
package share

import (
	"time"

	"github.com/NielHeesakkers/frames/internal/db"
)

type Status int

const (
	StatusActive Status = iota
	StatusExpired
	StatusRevoked
)

func Validate(s *db.Share) Status {
	if s.RevokedAt != nil {
		return StatusRevoked
	}
	if s.ExpiresAt != nil && !s.ExpiresAt.After(time.Now()) {
		return StatusExpired
	}
	return StatusActive
}

// IsUnderFolder reports whether childPath is the same as or under rootPath.
func IsUnderFolder(rootPath, childPath string) bool {
	if rootPath == "" {
		return true // root contains everything
	}
	if childPath == rootPath {
		return true
	}
	return len(childPath) > len(rootPath) && childPath[:len(rootPath)+1] == rootPath+"/"
}
