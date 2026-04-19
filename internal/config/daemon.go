package config

import "time"

// DaemonConfig controls background daemon behaviour.
type DaemonConfig struct {
	Enabled       bool          `yaml:"enabled"`
	CheckInterval time.Duration `yaml:"check_interval"`
	AutoApply     bool          `yaml:"auto_apply"`
	Notify        bool          `yaml:"notify"`
}
