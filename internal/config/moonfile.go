package config

import (
	"fmt"
	"path/filepath"

	"pyrorhythm.dev/moonshine/internal/packages"
)

// Moonfile is the combined view of moonconfig.yml + moonpackages.
type Moonfile struct {
	Moonshine
	Packages packages.List
}

// packagesPath derives the moonpackages.yml path from the moonconfig.yml path.
func packagesPath(configPath string) string {
	return filepath.Join(filepath.Dir(configPath), "moonpackages.yml")
}

// LoadMoonfile reads moonconfig.yml at configPath and moonpackages from the same directory.
func LoadMoonfile(configPath string) (*Moonfile, error) {
	cfg, err := Load(configPath)
	if err != nil {
		return nil, err
	}
	pkgs, err := packages.LoadMoonpackages(packagesPath(configPath))
	if err != nil {
		return nil, fmt.Errorf("loading moonpackages: %w", err)
	}
	return &Moonfile{Moonshine: *cfg, Packages: pkgs}, nil
}

// SaveMoonfile writes moonconfig.yml to configPath and moonpackages to the same directory.
func SaveMoonfile(configPath string, mf *Moonfile) error {
	if err := Save(configPath, &mf.Moonshine); err != nil {
		return err
	}
	return packages.SaveMoonpackages(packagesPath(configPath), mf.Packages)
}

// SavePackages writes only the moonpackages file for the given config path.
func SavePackages(configPath string, list packages.List) error {
	return packages.SaveMoonpackages(packagesPath(configPath), list)
}

// NewMoonfile returns a Moonfile with sensible defaults and the given operating mode.
func NewMoonfile(opMode string) *Moonfile {
	return &Moonfile{
		Moonshine: *New(opMode),
		Packages:  packages.List{},
	}
}
