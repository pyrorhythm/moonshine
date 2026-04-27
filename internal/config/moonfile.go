package config

import (
	"fmt"
	"os"
	"path/filepath"

	"pyrorhythm.dev/moonshine/internal/packages"
)

// Moonfile bundles the global config with the package list.
type Moonfile struct {
	Moonshine

	Packages packages.List
}

// packagesPath returns the packages file path, preferring packages.yml over the legacy name.
func packagesPath(configPath string) string {
	dir := filepath.Dir(configPath)
	preferred := filepath.Join(dir, "packages.yml")
	if _, err := os.Stat(preferred); err == nil {
		return preferred
	}
	return filepath.Join(dir, "moonpackages.yml")
}

// LoadBundle loads config.yml (or moonconfig.yml) plus the adjacent packages file.
func LoadBundle(configPath string) (*Moonfile, error) {
	cfg, err := Load(configPath)
	if err != nil {
		return nil, err
	}
	pkgs, err := packages.LoadMoonpackages(packagesPath(configPath))
	if err != nil {
		return nil, fmt.Errorf("loading packages: %w", err)
	}
	return &Moonfile{Moonshine: *cfg, Packages: pkgs}, nil
}

func SaveConfig(configPath string, mf *Moonfile) error {
	return Save(configPath, &mf.Moonshine)
}

func SavePackages(configPath string, list packages.List) error {
	return packages.SaveMoonpackages(packagesPath(configPath), list)
}

func NewMoonfile(opMode string) *Moonfile {
	return &Moonfile{
		Moonshine: *New(opMode),
		Packages:  packages.List{},
	}
}
