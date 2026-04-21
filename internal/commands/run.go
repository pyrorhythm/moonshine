package commands

import (
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"
)

const (
	configFlag  = "config"
	verboseFlag = "verbose"
	dryRunFlag  = "dry-run"
	modeFlag    = "mode"
)

func defaultConfigPath() string {
	xdg := os.Getenv("XDG_CONFIG_HOME")
	if xdg == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "moonconfig.yml"
		}
		xdg = filepath.Join(home, ".config")
	}
	return filepath.Join(xdg, "moonshine", "moonconfig.yml")
}

// All returns every subcommand registered in this package.
func All() []*cli.Command {
	return []*cli.Command{
		applyCommand(),
		diffCommand(),
		statusCommand(),
		addCommand(),
		searchCommand(),
		removeCommand(),
		lockCommand(),
		updateCommand(),
		tapCommand(),
		snapshotCommand(),
		doctorCommand(),
		initCommand(),
		daemonCommand(),
		hookCommand(),
	}
}

// Flag names re-exported for cmd wiring.
const (
	ConfigFlag  = configFlag
	VerboseFlag = verboseFlag
	DryRunFlag  = dryRunFlag
	ModeFlag    = modeFlag
)

// DefaultConfigPath returns the default moonconfig.yml location.
func DefaultConfigPath() string { return defaultConfigPath() }
