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
		Name:     "moonshine",
		Usage:    "declarative package manager",
		Version:  fmt.Sprintf("%s (commit %s, built %s)", version, commit, date),
		Flags:    commands.Flags(),
		Commands: commands.Commands(),
		ExitErrHandler: func(_ *cli.Context, err error) {
			if err != nil {
				fmt.Fprintln(os.Stderr, "error: "+err.Error())
				os.Exit(1)
			}
		},
	}).Run(os.Args)
}
