package backend

import (
	"context"
	"strings"
)

// InstalledPackage represents a package currently present on the system.
type InstalledPackage struct {
	Name    string
	Version string
	Source  string
}

// Package is the desired state passed to backend operations.
// PackageManager identifies the backend; Meta holds backend-specific metadata.
type Package struct {
	PackageManager string
	Meta           map[string]string
}

// Get returns a metadata value.
func (p Package) Get(key string) string { return p.Meta[key] }

// IsPinned reports whether a specific version was requested.
func (p Package) IsPinned() bool { return p.Meta["version"] != "" }

// Name returns the binary/formula name used to identify this package.
// For brew packages with brew_version set this is "name@brew_version".
// For go packages this is the last path segment of the install link.
func (p Package) Name() string {
	switch p.PackageManager {
	case "brew":
		name := p.Meta["name"]
		if bv := p.Meta["brew_version"]; bv != "" {
			return name + "@" + bv
		}
		return name
	case "go":
		parts := strings.Split(p.Meta["link"], "/")
		return parts[len(parts)-1]
	default:
		return p.Meta["name"]
	}
}

// SearchResult is a package found during a search.
type SearchResult struct {
	Name        string
	Version     string
	Description string
	Backend     string
}

// Searcher is an optional interface backends may implement to support package search.
type Searcher interface {
	Search(ctx context.Context, query string) ([]SearchResult, error)
}

// Backend is implemented by every package manager integration.
type Backend interface {
	// Name returns the lowercase identifier for this backend, e.g. "brew".
	Name() string

	// Available reports whether the underlying tool is installed and on PATH.
	Available() bool

	// ListInstalled returns all packages currently managed by this backend.
	ListInstalled(ctx context.Context) ([]InstalledPackage, error)

	// Install installs or upgrades pkg to the requested version (or latest).
	Install(ctx context.Context, pkg Package) error

	// Uninstall removes pkg from the system.
	Uninstall(ctx context.Context, pkg Package) error

	// Upgrade upgrades pkg to the latest available version.
	// For pinned packages this is a no-op; the reconciler handles version changes via Install.
	Upgrade(ctx context.Context, pkg Package) error
}
