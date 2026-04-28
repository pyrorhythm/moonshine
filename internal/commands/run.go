package commands

import (
	"os"
	"path/filepath"

	"github.com/urfave/cli/v3"
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
			return "config.yml"
		}
		xdg = filepath.Join(home, ".config")
	}
	return filepath.Join(xdg, "moonshine", "config.yml")
}

func Commands() []*cli.Command {
	return []*cli.Command{
		applyCommand(),
		diffCommand(),
		statusCommand(),
		listCommand(),
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

func Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    configFlag,
			Aliases: []string{"c"},
			Value:   defaultConfigPath(),
			Usage:   "path to config.yml",
			Sources: cli.EnvVars("MOONCONFIG"),
		},
		&cli.BoolFlag{Name: verboseFlag, Usage: "verbose output"},
		&cli.BoolFlag{Name: dryRunFlag, Usage: "show what would happen without making changes"},
		&cli.StringFlag{Name: modeFlag, Usage: "override operating mode (standalone|companion)"},
	}
}
