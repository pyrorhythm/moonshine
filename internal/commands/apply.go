package commands

import (
	"context"

	"github.com/urfave/cli/v3"
	"pyrorhythm.dev/moonshine/internal/ui"
)

func applyCommand() *cli.Command {
	return &cli.Command{
		Name:  "apply",
		Usage: "reconcile system state with packages.yml",
		Action: func(ctx context.Context, c *cli.Command) error {
			ac, err := loadContext(c)
			if err != nil {
				return err
			}
			ui.Banner()
			ui.Info("taking system snapshot…")
			return applyAC(ctx, ac)
		},
	}
}
