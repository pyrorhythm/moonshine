package config

import (
	"fmt"
	"os"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/pyrorhythm/moonshine/internal/config/mode"
)

// Load reads and parses a moonfile.yaml at path.
func Load(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading moonfile: %w", err)
	}
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing moonfile: %w", err)
	}
	m.applyDefaults()
	if err := m.validate(); err != nil {
		return nil, err
	}
	return &m, nil
}

// Save writes the manifest to path atomically.
func Save(path string, m *Manifest) error {
	data, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshalling moonfile: %w", err)
	}
	tmp, err := os.CreateTemp("", "moonfile-*.yaml")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	tmp.Close()
	return os.Rename(tmp.Name(), path)
}

// New returns a Manifest populated with sensible defaults.
func New(opMode string) *Manifest {
	m := &Manifest{
		Moonshine: MoonshineConfig{
			Mode:     mode.OperatingMode(opMode),
			LocalTap: "moonshine-local",
		},
		Daemon: DaemonConfig{
			Enabled:       false,
			CheckInterval: 6 * time.Hour,
			AutoApply:     false,
			Notify:        true,
		},
		Packages: make(map[string][]Package),
	}
	return m
}
