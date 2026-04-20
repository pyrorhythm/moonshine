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

// Name returns the primary identifier used for this package by the backend.
// For go packages this is the binary name (last path segment of the install target).
func (p Package) Name() string {
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
