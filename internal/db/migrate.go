package db

import (
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
)

//go:embed migrations/*.sql
var migrations embed.FS

func (d *DB) Migrate() error {
	entries, err := fs.ReadDir(migrations, "migrations")
	if err != nil {
		return err
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)

	for _, n := range names {
		parts := strings.SplitN(n, "_", 2)
		ver, err := strconv.Atoi(parts[0])
		if err != nil {
			return fmt.Errorf("bad migration filename %q: %w", n, err)
		}
		var applied int
		_ = d.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version=?`, ver).Scan(&applied)
		if applied > 0 {
			continue
		}
		body, err := fs.ReadFile(migrations, "migrations/"+n)
		if err != nil {
			return err
		}
		if _, err := d.Exec(string(body)); err != nil {
			return fmt.Errorf("migration %s: %w", n, err)
		}
		if _, err := d.Exec(`INSERT INTO schema_migrations(version) VALUES (?)`, ver); err != nil {
			return fmt.Errorf("record migration %s: %w", n, err)
		}
	}
	return nil
}
