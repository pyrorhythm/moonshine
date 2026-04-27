package brew

import (
	"bufio"
	"bytes"
	"strings"

	"pyrorhythm.dev/moonshine/pkg/backend"
)

// InstalledPackage is the brew-specific installed package record.
type InstalledPackage struct {
	Name        string
	Version     string
	Tap         string
	Description string
}

func (p InstalledPackage) GetName() string    { return p.Name }
func (p InstalledPackage) GetVersion() string { return p.Version }
func (p InstalledPackage) GetSource() string {
	if p.Tap != "" {
		return p.Tap
	}
	return "homebrew/core"
}

var _ backend.InstalledPackage = InstalledPackage{}

// InfoEntry is a formula record from `brew info --json`.
type InfoEntry struct {
	Name      string `json:"name"`
	FullName  string `json:"full_name"`
	Tap       string `json:"tap"`
	Desc      string `json:"desc"`
	Installed []struct {
		Version string `json:"version"`
	} `json:"installed"`
}

// parseLeaves parses `brew leaves` output: one formula name per line.
func parseLeaves(data []byte) []string {
	var names []string
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		if name := strings.TrimSpace(scanner.Text()); name != "" {
			names = append(names, name)
		}
	}
	return names
}
