package version

import (
	_ "embed"
	"regexp"
	"strings"
)

// Current is the human-readable version string. Bump by 0.1 per shipped change
// (matches the entries in CHANGELOG.md).
const Current = "0.17.0"

//go:embed changelog.md
var rawChangelog string

// Changelog returns the full changelog as markdown.
func Changelog() string { return rawChangelog }

// Latest returns the most recent changelog entry (header + body until the next `## `).
func Latest() string {
	lines := strings.Split(rawChangelog, "\n")
	var out []string
	inEntry := false
	headerRE := regexp.MustCompile(`^## `)
	for _, l := range lines {
		if headerRE.MatchString(l) {
			if inEntry {
				break
			}
			inEntry = true
		}
		if inEntry {
			out = append(out, l)
		}
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}
