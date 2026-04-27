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

func doctorCommand() *cli.Command {
	return &cli.Command{
		Name:  "doctor",
		Usage: "diagnose drift, conflicts, and configuration issues",
		Action: func(ctx context.Context, c *cli.Command) error {
			ac, err := loadContext(c)
			if err != nil {
				return err
			}

			issues := 0

			for _, b := range ac.registry.All() {
				if !b.Available() {
					ui.Warn(fmt.Sprintf("backend %q: tool not found on PATH", b.Name()))
					issues++
				}
			}

			ss, err := state.Snapshot(ctx, ac.registry)
			if err != nil {
				ui.Error("could not take system snapshot: " + err.Error())
				issues++
			} else {
				plan := reconciler.Diff(ac.moonfile, ss, ac.lock)
				if plan.HasChanges() {
					fmt.Fprintln(os.Stdout, "\ndrift detected:")
					for _, a := range plan.Actions {
						if a.Kind != reconciler.ActionNone {
							ui.Warn(fmt.Sprintf(
								"  %s %s/%s: %s",
								a.Kind, a.BackendName, actionPkgName(a), a.Reason,
							))
						}
					}
					issues++
				} else {
					ui.Success("no drift detected")
				}
			}

			if ac.lock == nil {
				ui.Warn("no lockfile found; run 'ms lock' to generate one")
				issues++
			}

			if issues == 0 {
				ui.Success("everything looks good!")
			} else {
				fmt.Fprintf(os.Stdout, "\n%d issue(s) found\n", issues)
			}
			return nil
		},
	}
}

func actionPkgName(a reconciler.PackageAction) string {
	if a.Package.Meta != nil {
		if n := a.Package.Name(); n != "" {
			return n
		}
	}
	if a.Current != nil {
		return a.Current.GetName()
	}
	return "?"
}
