package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"
	"pyrorhythm.dev/moonshine/internal/reconciler"
	"pyrorhythm.dev/moonshine/internal/state"
	"pyrorhythm.dev/moonshine/internal/ui"
)

func diffCommand() *cli.Command {
	return &cli.Command{
		Name:  "diff",
		Usage: "show what apply would change",
		Action: func(ctx context.Context, c *cli.Command) error {
			ac, err := loadContext(ctx, c)
			if err != nil {
				return err
			}
			ss, err := state.Snapshot(ctx, ac.registry)
			if err != nil {
				return fmt.Errorf("snapshot: %w", err)
			}
			plan := reconciler.Diff(ac.moonfile, ss, ac.lock)
			ui.PrintDiff(os.Stdout, plan)
			return nil
		},
	}
}
