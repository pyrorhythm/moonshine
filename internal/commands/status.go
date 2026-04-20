package commands

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
	"pyrorhythm.dev/moonshine/internal/reconciler"
	"pyrorhythm.dev/moonshine/internal/state"
	"pyrorhythm.dev/moonshine/internal/ui"
)

func statusCommand() *cli.Command {
	return &cli.Command{
		Name:  "status",
		Usage: "show current state vs declared state",
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
			fmt.Fprintf(os.Stdout, "mode: %s\n\n", ac.moonfile.Mode)
			ui.PrintStatus(os.Stdout, plan)
			return nil
		},
	}
}
