package config

import (
	"fmt"
	"os"
	"time"

	"github.com/goccy/go-yaml"
	"pyrorhythm.dev/moonshine/internal/config/mode"
	"pyrorhythm.dev/moonshine/internal/hooks"
)

// Moonshine holds global moonshine settings.
type Moonshine struct {
	Mode     mode.OperatingMode   `yaml:"mode"`
	Backends []string             `yaml:"backends"`
	Daemon   DaemonConfig         `yaml:"daemon"`
	LocalTap string               `yaml:"local_tap,omitempty"`
	Hooks    hooks.Hooks          `yaml:"hooks,omitempty"`
	Shell    []ShellBackendConfig `yaml:"shell_backends,omitempty"`
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
	if len(m.Backends) == 0 {
		m.Backends = []string{"brew"}
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

// Load reads and parses a config file at path.
func Load(path string) (*Moonshine, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}
	var m Moonshine
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	m.applyDefaults()
	if err := m.validate(); err != nil {
		return nil, err
	}
	return &m, nil
}

// Save writes the config to path atomically.
func Save(path string, m *Moonshine) error {
	data, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshalling config: %w", err)
	}
	tmp, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return err
	}
	_ = tmp.Close()
	return os.Rename(tmp.Name(), path)
}

// New returns a Moonshine populated with sensible defaults.
func New(opMode string) *Moonshine {
	return &Moonshine{
		Mode:     mode.OperatingMode(opMode),
		LocalTap: "moonshine-local",
		Backends: []string{"brew"},
		Daemon: DaemonConfig{
			Enabled:       false,
			CheckInterval: 6 * time.Hour,
			AutoApply:     false,
			Notify:        true,
		},
	}
}
