package commands

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
		ArgsUsage: "[[backend#]package]",
		Action: func(c *cli.Context) error {
			ac, err := loadContext(c)
			if err != nil {
				return err
			}
			ss, err := state.Snapshot(c.Context, ac.registry)
			if err != nil {
				return fmt.Errorf("snapshot: %w", err)
			}

			var targetBackend, targetName string
			if arg := c.Args().First(); arg != "" {
				ref, err := parsePackageRef(arg)
				if err != nil {
					return err
				}
				targetBackend, targetName = ref.backend, ref.name
			}

			return doUpdate(c.Context, ac, ss, targetBackend, targetName)
		},
	}
}

func doUpdate(
	ctx context.Context,
	ac *appContext,
	ss state.SystemState,
	targetBackend, targetName string,
) error {
	updated := 0
	for _, dp := range ac.moonfile.Packages {
		backendName := dp.PackageManager
		binaryName := dp.BinaryName()

		if targetBackend != "" && backendName != targetBackend {
			continue
		}
		if targetName != "" && binaryName != targetName {
			continue
		}

		b, ok := ac.registry.Get(backendName)
		if !ok || !b.Available() {
			continue
		}
		if dp.Pinned() {
			ui.Warn(fmt.Sprintf(
				"%s/%s is pinned at %s; use 'ms add %s#%s@<new>' to change",
				backendName, binaryName, dp.Version(), backendName, binaryName,
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

	if updated == 0 && targetName == "" {
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
