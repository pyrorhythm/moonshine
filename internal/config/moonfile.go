package config

import (
	"fmt"
	"path/filepath"

	"pyrorhythm.dev/moonshine/internal/packages"
)

// Moonfile bundles the global config with the package list.
type Moonfile struct {
	Moonshine

	Packages packages.List
}

// packagesPath returns the packages.yml path adjacent to configPath.
func packagesPath(configPath string) string {
	return filepath.Join(filepath.Dir(configPath), "packages.yml")
}

// LoadBundle loads config.yml plus the adjacent packages.yml.
func LoadBundle(configPath string) (*Moonfile, error) {
	cfg, err := Load(configPath)
	if err != nil {
		return nil, err
	}
	pkgs, err := packages.LoadPackages(packagesPath(configPath))
	if err != nil {
		return nil, fmt.Errorf("loading packages: %w", err)
	}
	return &Moonfile{Moonshine: *cfg, Packages: pkgs}, nil
}

func SaveConfig(configPath string, mf *Moonfile) error {
	return Save(configPath, &mf.Moonshine)
}

func SavePackages(configPath string, list packages.List) error {
	return packages.SavePackages(packagesPath(configPath), list)
}

func NewMoonfile(opMode string) *Moonfile {
	return &Moonfile{
		Moonshine: *New(opMode),
		Packages:  packages.List{},
	}
}
