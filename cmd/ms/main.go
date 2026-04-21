package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"

	"pyrorhythm.dev/moonshine/internal/commands"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	(&cli.App{
		Name:    "moonshine",
		Usage:   "declarative package manager",
		Version: fmt.Sprintf("%s (commit %s, built %s)", version, commit, date),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    commands.ConfigFlag,
				Aliases: []string{"c"},
				Value:   commands.DefaultConfigPath(),
				Usage:   "path to moonconfig.yml",
				EnvVars: []string{"MOONCONFIG"},
			},
			&cli.BoolFlag{Name: commands.VerboseFlag, Usage: "verbose output"},
			&cli.BoolFlag{Name: commands.DryRunFlag, Usage: "show what would happen without making changes"},
			&cli.StringFlag{Name: commands.ModeFlag, Usage: "override operating mode (standalone|companion)"},
		},
		Commands: commands.All(),
		ExitErrHandler: func(_ *cli.Context, err error) {
			if err != nil {
				fmt.Fprintln(os.Stderr, "error: "+err.Error())
				os.Exit(1)
			}
		},
	}).Run(os.Args)
}
