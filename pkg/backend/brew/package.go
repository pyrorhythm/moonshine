package brew

import (
	"strings"

	"pyrorhythm.dev/moonshine/pkg/backend"
)

// Package is the canonical Homebrew formula representation, used for both
// desired state (install/uninstall/upgrade) and installed state (ListInstalled).
type Package struct {
	Name        string
	Tap         string
	Version     string
	BrewVersion string
	Description string
}

// GetName returns the fully-qualified formula name.
// For tapped formulae it is "tap/name"; for core formulae it is just the name.
func (p Package) GetName() string {
	if p.Tap != "" && p.Tap != "homebrew/core" {
		return p.Tap + "/" + p.Name
	}
	return p.Name
}

func (p Package) GetVersion() string { return p.Version }

func (p Package) GetSource() string {
	if p.Tap != "" {
		return p.Tap
	}
	return "homebrew/core"
}

// FormulaRef returns the formula identifier used with brew install/uninstall/upgrade.
// It includes the tap prefix and brew_version suffix when present.
func (p Package) FormulaRef() string {
	name := p.Name
	if p.BrewVersion != "" {
		name += "@" + p.BrewVersion
	}
	if p.Tap != "" && p.Tap != "homebrew/core" {
		return p.Tap + "/" + name
	}
	return name
}

// fromBackend converts a generic backend.Package to a brew Package.
func fromBackend(pkg backend.Package) Package {
	return Package{
		Name:        pkg.Get("name"),
		Tap:         pkg.Get("tap"),
		Version:     pkg.Get("version"),
		BrewVersion: pkg.Get("brew_version"),
	}
}

// fromLeavesName parses a brew leaves output line into a Package.
// Tapped formulae appear as "org/tap/name"; core formulae as plain "name".
func fromLeavesName(fullName string) Package {
	if strings.Count(fullName, "/") >= 2 {
		idx := strings.LastIndex(fullName, "/")
		return Package{Tap: fullName[:idx], Name: fullName[idx+1:]}
	}
	return Package{Name: fullName}
}

var _ backend.InstalledPackage = Package{}
