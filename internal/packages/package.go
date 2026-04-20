package packages

import "strings"

// Package is a single package entry.
// PackageManager identifies the backend; Meta holds backend-specific metadata.
type Package struct {
	PackageManager string
	Meta           map[string]string
}

// Get returns a metadata value.
func (p Package) Get(key string) string { return p.Meta[key] }

// BinaryName returns the binary/formula name used to match against installed packages.
// For go packages this is the last path segment of the install target.
func (p Package) BinaryName() string {
	switch p.PackageManager {
	case "go":
		target := p.Meta["module"]
		if path := p.Meta["path"]; path != "" {
			target = target + "/" + path
		}
		parts := strings.Split(target, "/")
		return parts[len(parts)-1]
	default:
		return p.Meta["name"]
	}
}

// Version returns the pinned version, or empty for latest.
func (p Package) Version() string { return p.Meta["version"] }

// Pinned reports whether a specific version is requested.
func (p Package) Pinned() bool { return p.Meta["version"] != "" }
