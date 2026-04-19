package backend

import "context"

// InstalledPackage represents a package currently present on the system.
type InstalledPackage struct {
	Name    string
	Version string
	Source  string // tap, registry URL, or similar
}

// Package is the desired state for one package entry.
type Package struct {
	Name    string
	Version string   // empty = latest
	Options []string // extra flags forwarded to the registry tool
}

// IsPinned reports whether a specific version is requested.
func (p Package) IsPinned() bool { return p.Version != "" }

// Backend is implemented by every package manager integration.
type Backend interface {
	// Name returns the lowercase identifier used in moonfile.yaml, e.g. "brew".
	Name() string

	// Available reports whether the underlying tool is installed and on PATH.
	Available() bool

	// ListInstalled returns all packages currently managed by this registry.
	ListInstalled(ctx context.Context) ([]InstalledPackage, error)

	// Install installs or upgrades pkg to the requested version (or latest).
	Install(ctx context.Context, pkg Package) error

	// Uninstall removes pkg from the system.
	Uninstall(ctx context.Context, pkg Package) error

	// Upgrade upgrades pkg to the latest available version.
	// For pinned packages this is a no-op; the reconciler handles version changes via Install.
	Upgrade(ctx context.Context, pkg Package) error
}
