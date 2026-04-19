package config

// Package is a single entry in the packages map.
type Package struct {
	Name    string   `yaml:"name"`
	Version string   `yaml:"version,omitempty"`
	Options []string `yaml:"options,omitempty"`
}

// Pinned reports whether this package declares a specific version.
func (p Package) Pinned() bool { return p.Version != "" }

// FQN returns the canonical install target, e.g. "git@2.41.0" or "git".
func (p Package) FQN() string {
	if p.Pinned() {
		return p.Name + "@" + p.Version
	}
	return p.Name
}
