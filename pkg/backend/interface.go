package backend

import (
	"context"
	"strings"
)

// InstalledPackage is the common interface for packages currently present on the system.
// Each backend provides a concrete type with backend-specific metadata.
type InstalledPackage interface {
	GetName() string
	GetVersion() string
	GetSource() string
}

// SimplePackage is a generic InstalledPackage used in tests and
// contexts where backend-specific information is not needed.
type SimplePackage struct {
	Name    string
	Version string
	Source  string
}

func (p SimplePackage) GetName() string    { return p.Name }
func (p SimplePackage) GetVersion() string { return p.Version }
func (p SimplePackage) GetSource() string  { return p.Source }

// Package is the desired state passed to backend operations.
// PackageManager identifies the backend; Meta holds backend-specific metadata.
type Package struct {
	PackageManager string
	Meta           map[string]string
}

func (p Package) Get(key string) string { return p.Meta[key] }

func (p Package) IsPinned() bool { return p.Meta["version"] != "" }

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
	Name() string
	Available() bool
	ListInstalled(ctx context.Context) ([]InstalledPackage, error)
	Install(ctx context.Context, pkg Package) error
	Uninstall(ctx context.Context, pkg Package) error
	Upgrade(ctx context.Context, pkg Package) error
}
