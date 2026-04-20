package commands

import (
	"github.com/urfave/cli/v2"
	"pyrorhythm.dev/moonshine/internal/ui"
)

func applyCommand() *cli.Command {
	return &cli.Command{
		Name:  "apply",
		Usage: "reconcile system state with moonpackages",
		Action: func(c *cli.Context) error {
			ac, err := loadContext(c)
			if err != nil {
				return err
			}
			ui.Banner()
			ui.Info("taking system snapshot…")
			return applyAC(c.Context, ac)
		},
	}
}
