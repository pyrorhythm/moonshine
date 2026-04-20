package main

import (
	"context"
	"fmt"
	"time"

	"github.com/pyrorhythm/moonshine/internal/lockfile"
	"github.com/pyrorhythm/moonshine/internal/state"
	"github.com/pyrorhythm/moonshine/internal/ui"
	"github.com/pyrorhythm/moonshine/pkg/backend"
	"github.com/urfave/cli/v2"
)

func updateCommand() *cli.Command {
	return &cli.Command{
		Name:      "update",
		Usage:     "upgrade unpinned packages to latest",
		ArgsUsage: "[package]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "backend",
				Aliases: []string{"b"},
				Usage:   "limit to this backend",
			},
		},
		Action: func(c *cli.Context) error {
			target := c.Args().First()
			backendFilter := c.String("backend")
			ac, err := loadContext(c)
			if err != nil {
				return err
			}
			ss, err := state.Snapshot(c.Context, ac.registry)
			if err != nil {
				return fmt.Errorf("snapshot: %w", err)
			}
			return doUpdate(c.Context, ac, ss, target, backendFilter)
		},
	}
}

func doUpdate(
	ctx context.Context,
	ac *appContext,
	ss state.SystemState,
	target, backendFilter string,
) error {
	updated := 0
	for _, dp := range ac.moonfile.Packages {
		backendName := dp.PackageManager
		if backendFilter != "" && backendName != backendFilter {
			continue
		}
		b, ok := ac.registry.Get(backendName)
		if !ok || !b.Available() {
			continue
		}
		binaryName := dp.BinaryName()
		if target != "" && binaryName != target {
			continue
		}
		if dp.Pinned() {
			ui.Warn(fmt.Sprintf(
				"%s/%s is pinned at %s; use 'ms add %s --version <new>' to change",
				backendName, binaryName, dp.Version(), binaryName,
			))
			continue
		}
		installed, found := ss.Get(backendName, binaryName)
		if !found {
			ui.Warn(fmt.Sprintf("%s/%s not installed; run 'ms apply' first", backendName, binaryName))
			continue
		}
		ui.Info(fmt.Sprintf("upgrading %s/%s…", backendName, binaryName))
		if !ac.dryRun {
			bpkg := backend.Package{PackageManager: backendName, Meta: dp.Meta}
			if err := b.Upgrade(ctx, bpkg); err != nil {
				ui.Error(fmt.Sprintf("upgrading %s: %v", binaryName, err))
				continue
			}
			ac.lock.Upsert(backendName, lockfile.LockedPackage{
				Name:        binaryName,
				Version:     installed.Version,
				Source:      installed.Source,
				InstalledAt: time.Now().UTC(),
			})
		}
		updated++
	}

	if updated == 0 && target == "" {
		ui.Success("All unpinned packages are up to date.")
		return nil
	}
	if !ac.dryRun && updated > 0 {
		if err := lockfile.Save(ac.lockPath, ac.lock); err != nil {
			ui.Warn("failed to save lockfile: " + err.Error())
		}
		ui.Success(fmt.Sprintf("Updated %d package(s).", updated))
	}
	return nil
}
