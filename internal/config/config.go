package config

import (
	"fmt"
	"os"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/pyrorhythm/moonshine/internal/config/mode"
	"github.com/pyrorhythm/moonshine/internal/hooks"
)

// Moonshine holds global moonshine settings.
type Moonshine struct {
	Mode     mode.OperatingMode   `yaml:"mode"`
	LocalTap string               `yaml:"local_tap"`
	Daemon   DaemonConfig         `yaml:"daemon"`
	Hooks    hooks.Hooks          `yaml:"hooks"`
	Shell    []ShellBackendConfig `yaml:"shell_backends"`
}

func (m *Moonshine) applyDefaults() {
	if m.Mode == "" {
		m.Mode = mode.Default
	}
	if m.LocalTap == "" {
		m.LocalTap = "moonshine-local"
	}
	if m.Daemon.CheckInterval.Seconds() == 0 {
		m.Daemon.CheckInterval = 6 * time.Hour
	}
}

func (m *Moonshine) validate() error {
	if !m.Mode.Valid() {
		return fmt.Errorf(
			"invalid mode %q: must be %q or %q",
			m.Mode,
			mode.Standalone,
			mode.Companion,
		)
	}
	return nil
}

// Load reads and parses a moonfile.yaml at path.
func Load(path string) (*Moonshine, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading moonfile: %w", err)
	}
	var m Moonshine
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
func Save(path string, m *Moonshine) error {
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

// New returns a Moonshine populated with sensible defaults.
func New(opMode string) *Moonshine {
	m := &Moonshine{
		Mode:     mode.OperatingMode(opMode),
		LocalTap: "moonshine-local",
		Daemon: DaemonConfig{
			Enabled:       false,
			CheckInterval: 6 * time.Hour,
			AutoApply:     false,
			Notify:        true,
		},
	}
	return m
}
