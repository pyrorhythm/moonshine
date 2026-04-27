package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/urfave/cli/v3"

	"pyrorhythm.dev/moonshine/internal/commands"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGABRT,
	)
	defer cancel()

	app := &cli.Command{
		Name:                       "moonshine",
		EnableShellCompletion:      true,
		ShellCompletionCommandName: "completion",
		Usage:                      "declarative package manager",
		Version:                    fmt.Sprintf("%s (commit %s, built %s)", version, commit, date),
		Flags:                      commands.Flags(),
		Commands:                   commands.Commands(),
		ExitErrHandler: func(_ context.Context, c *cli.Command, err error) {
			if err != nil {
				fmt.Fprintf(os.Stderr, "command %s - error: %s", c.Name, err.Error())
				os.Exit(1)
			}
		},
	}
	return app.Run(ctx, os.Args)
}
