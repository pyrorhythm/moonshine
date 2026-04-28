package commands

import (
	"context"
	"os"

	"github.com/urfave/cli/v3"
	"pyrorhythm.dev/moonshine/internal/ui"
)

func diffCommand() *cli.Command {
	return &cli.Command{
		Name:  "diff",
		Usage: "show what apply would change",
		Action: func(ctx context.Context, c *cli.Command) error {
			ac, err := loadContext(c)
			if err != nil {
				return err
			}
			plan, err := computePlan(ctx, ac)
			if err != nil {
				return err
			}
			ui.PrintDiff(os.Stdout, plan)
			return nil
		},
	}
}
