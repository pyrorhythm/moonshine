package config

import (
	"fmt"
	"time"

	"github.com/pyrorhythm/moonshine/internal/config/mode"
	"github.com/pyrorhythm/moonshine/internal/hooks"
)

// MoonshineConfig holds global moonshine settings.
type MoonshineConfig struct {
	Mode     mode.OperatingMode `yaml:"mode"`
	LocalTap string             `yaml:"local_tap"`
}

// Manifest is the parsed representation of moonfile.yaml.
type Manifest struct {
	Moonshine MoonshineConfig      `yaml:"moonshine"`
	Daemon    DaemonConfig         `yaml:"daemon"`
	Hooks     hooks.Hooks          `yaml:"hooks"`
	Custom    []ShellBackendConfig `yaml:"custom_backends"`
	Packages  map[string][]Package `yaml:"packages"`
}

func (m *Manifest) applyDefaults() {
	if m.Moonshine.Mode == "" {
		m.Moonshine.Mode = mode.Default
	}
	if m.Moonshine.LocalTap == "" {
		m.Moonshine.LocalTap = "moonshine-local"
	}
	if m.Daemon.CheckInterval.Seconds() == 0 {
		m.Daemon.CheckInterval = 6 * time.Hour
	}
	if m.Packages == nil {
		m.Packages = make(map[string][]Package)
	}
}

func (m *Manifest) validate() error {
	if !m.Moonshine.Mode.Valid() {
		return fmt.Errorf("invalid mode %q: must be %q or %q", m.Moonshine.Mode, mode.Standalone, mode.Companion)
	}
	for backend, pkgs := range m.Packages {
		seen := make(map[string]bool)
		for _, p := range pkgs {
			if p.Name == "" {
				return fmt.Errorf("registry %q: package missing name", backend)
			}
			if seen[p.Name] {
				return fmt.Errorf("registry %q: duplicate package %q", backend, p.Name)
			}
			seen[p.Name] = true
		}
	}
	return nil
}
