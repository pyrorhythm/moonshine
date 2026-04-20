package main

import (
	"context"
	"fmt"
	"os"

	"github.com/pyrorhythm/moonshine/internal/lockfile"
	"github.com/pyrorhythm/moonshine/internal/reconciler"
	"github.com/pyrorhythm/moonshine/internal/state"
	"github.com/pyrorhythm/moonshine/internal/ui"
)

// applyAC runs the full apply flow using an already-loaded appContext.
func applyAC(ctx context.Context, ac *appContext) error {
	ss, err := state.Snapshot(ctx, ac.registry)
	if err != nil {
		return fmt.Errorf("snapshot: %w", err)
	}

	plan := reconciler.Diff(ac.moonfile, ss, ac.lock)
	if !plan.HasChanges() {
		ui.Success("Already up to date — nothing to do.")
		return nil
	}

	ui.PrintDiff(os.Stdout, plan)
	fmt.Println()

	if ac.dryRun {
		ui.Warn("dry-run: no changes made")
		return nil
	}

	opts := reconciler.ApplyOptions{
		DryRun:  ac.dryRun,
		Verbose: ac.verbose,
		Hooks:   ac.moonfile.Hooks,
		Mode:    string(ac.moonfile.Mode),
	}
	if err := reconciler.Apply(ctx, plan, ac.registry, ac.moonfile, ac.lock, opts); err != nil {
		return err
	}

	if err := lockfile.Save(ac.lockPath, ac.lock); err != nil {
		ui.Warn("failed to save lockfile: " + err.Error())
	}

	ui.Success("Apply complete.")
	return nil
}
