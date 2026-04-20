package commands

import (
	"fmt"

	"github.com/urfave/cli/v2"
	"pyrorhythm.dev/moonshine/internal/ui"
	brewbackend "pyrorhythm.dev/moonshine/pkg/backend/brew"
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
					if err := runner.TapCreate(c.Context, ac.moonfile.LocalTap); err != nil {
						return fmt.Errorf("creating tap %q: %w", ac.moonfile.LocalTap, err)
					}
					ui.Success(fmt.Sprintf("tap %q initialised", ac.moonfile.LocalTap))
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
					exists, err := runner.TapExists(c.Context, ac.moonfile.LocalTap)
					if err != nil {
						return err
					}
					if !exists {
						ui.Warn(
							fmt.Sprintf(
								"tap %q does not exist; run 'ms tap init'",
								ac.moonfile.LocalTap,
							),
						)
						return nil
					}
					ui.Success(fmt.Sprintf("tap %q is registered", ac.moonfile.LocalTap))
					return nil
				},
			},
		},
	}
}
