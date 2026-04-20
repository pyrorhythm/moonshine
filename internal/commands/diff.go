package commands

import (
	"fmt"
	"os"

	"github.com/pyrorhythm/moonshine/internal/reconciler"
	"github.com/pyrorhythm/moonshine/internal/state"
	"github.com/pyrorhythm/moonshine/internal/ui"
	"github.com/urfave/cli/v2"
)

func diffCommand() *cli.Command {
	return &cli.Command{
		Name:  "diff",
		Usage: "show what apply would change",
		Action: func(c *cli.Context) error {
			ac, err := loadContext(c)
			if err != nil {
				return err
			}
			ss, err := state.Snapshot(c.Context, ac.registry)
			if err != nil {
				return fmt.Errorf("snapshot: %w", err)
			}
			plan := reconciler.Diff(ac.moonfile, ss, ac.lock)
			ui.PrintDiff(os.Stdout, plan)
			return nil
		},
	}
}
