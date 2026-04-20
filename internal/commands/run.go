package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"
)

const Version = "0.1.0"

const (
	configFlag  = "config"
	verboseFlag = "verbose"
	dryRunFlag  = "dry-run"
	modeFlag    = "mode"
)

// defaultConfigPath returns $XDG_CONFIG_HOME/moonshine/moonconfig.yml,
// falling back to ~/.config/moonshine/moonconfig.yml per the XDG spec.
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

// Run executes the moonshine CLI with the given args (typically os.Args).
func Run(args []string) error {
	return newApp().Run(args)
}

func newApp() *cli.App {
	app := &cli.App{
		Name:    "moonshine",
		Usage:   "declarative package manager",
		Version: Version,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    configFlag,
				Aliases: []string{"c"},
				Value:   defaultConfigPath(),
				Usage:   "path to moonconfig.yml",
				EnvVars: []string{"MOONCONFIG"},
			},
			&cli.BoolFlag{Name: verboseFlag, Usage: "verbose output"},
			&cli.BoolFlag{Name: dryRunFlag, Usage: "show what would happen without making changes"},
			&cli.StringFlag{Name: modeFlag, Usage: "override operating mode (standalone|companion)"},
		},
		Commands: []*cli.Command{
			applyCommand(),
			diffCommand(),
			statusCommand(),
			addCommand(),
			removeCommand(),
			lockCommand(),
			updateCommand(),
			tapCommand(),
			snapshotCommand(),
			doctorCommand(),
			initCommand(),
			daemonCommand(),
			hookCommand(),
		},
		ExitErrHandler: func(_ *cli.Context, err error) {
			if err != nil {
				fmt.Fprintln(os.Stderr, "error: "+err.Error())
				os.Exit(1)
			}
		},
	}
	return app
}
