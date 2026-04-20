package main

import (
	"fmt"

	"github.com/pyrorhythm/moonshine/internal/ui"
	brewbackend "github.com/pyrorhythm/moonshine/pkg/backend/brew"
	"github.com/urfave/cli/v2"
)

func tapCommand() *cli.Command {
	return &cli.Command{
		Name:  "tap",
		Usage: "manage the moonshine local brew tap",
		Subcommands: []*cli.Command{
			{
				Name:  "init",
				Usage: "create and register the local tap",
				Action: func(c *cli.Context) error {
					ac, err := loadContext(c)
					if err != nil {
						return err
					}
					runner, err := brewbackend.NewRunner(ac.verbose)
					if err != nil {
						return err
					}
					tapName := ac.moonfile.LocalTap
					if err := runner.TapCreate(c.Context, tapName); err != nil {
						return fmt.Errorf("creating tap %q: %w", tapName, err)
					}
					ui.Success(fmt.Sprintf("tap %q initialised", tapName))
					return nil
				},
			},
			{
				Name:  "status",
				Usage: "show local tap info",
				Action: func(c *cli.Context) error {
					ac, err := loadContext(c)
					if err != nil {
						return err
					}
					runner, err := brewbackend.NewRunner(ac.verbose)
					if err != nil {
						return err
					}
					tapName := ac.moonfile.LocalTap
					exists, err := runner.TapExists(c.Context, tapName)
					if err != nil {
						return err
					}
					if !exists {
						ui.Warn(fmt.Sprintf("tap %q does not exist; run 'ms tap init'", tapName))
						return nil
					}
					ui.Success(fmt.Sprintf("tap %q is registered", tapName))
					return nil
				},
			},
		},
	}
}
