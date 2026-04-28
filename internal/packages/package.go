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

// BinaryName returns the identifier used to match against installed packages.
// For brew packages with brew_version set, this is "name@brew_version" (e.g. openssl@3).
// For go packages this is the last path segment of the install link.
func (p Package) BinaryName() string {
	switch p.PackageManager {
	case "brew":
		name := p.Meta["name"]
		if bv := p.Meta["brew_version"]; bv != "" {
			name = name + "@" + bv
		}
		if tap := p.Meta["tap"]; tap != "" {
			return tap + "/" + name
		}
		return name
	case "go":
		link := p.Meta["link"]
		parts := strings.Split(link, "/")
		return parts[len(parts)-1]
	default:
		return p.Meta["name"]
	}
}

// Version returns the pinned version, or empty for latest.
func (p Package) Version() string { return p.Meta["version"] }

// Pinned reports whether a specific version is requested.
func (p Package) Pinned() bool { return p.Meta["version"] != "" }
