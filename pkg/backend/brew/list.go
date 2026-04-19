package brew

import (
	"bufio"
	"bytes"
	"strings"
)

type ListEntry struct {
	Name    string
	Version string
}

// parseListOutput parses the plain-text output of `brew list --versions`.
// Each line is: "<name> <version> [<version2> ...]"; we take the last version.
func parseListOutput(data []byte) []ListEntry {
	var entries []ListEntry
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		entry := ListEntry{Name: fields[0]}
		if len(fields) >= 2 {
			entry.Version = fields[len(fields)-1] // latest installed version
		}
		entries = append(entries, entry)
	}
	return entries
}
