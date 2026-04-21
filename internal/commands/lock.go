package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/urfave/cli/v3"
	"pyrorhythm.dev/moonshine/internal/lockfile"
	"pyrorhythm.dev/moonshine/internal/state"
	"pyrorhythm.dev/moonshine/internal/ui"
)

func lockCommand() *cli.Command {
	return &cli.Command{
		Name:  "lock",
		Usage: "regenerate moonshine.lock from current installed state",
		Action: func(ctx context.Context, c *cli.Command) error {
			ac, err := loadContext(ctx, c)
			if err != nil {
				return err
			}
			ss, err := state.Snapshot(ctx, ac.registry)
			if err != nil {
				return fmt.Errorf("snapshot: %w", err)
			}

			newLock := lockfile.New(string(ac.moonfile.Mode))
			for _, dp := range ac.moonfile.Packages {
				if installed, ok := ss.Get(dp.PackageManager, dp.BinaryName()); ok {
					newLock.Upsert(dp.PackageManager, lockfile.LockedPackage{
						Name:        dp.BinaryName(),
						Version:     installed.Version,
						Source:      installed.Source,
						InstalledAt: time.Now().UTC(),
					})
				}
			}

			if err := lockfile.Save(ac.lockPath, newLock); err != nil {
				return fmt.Errorf("saving lockfile: %w", err)
			}
			ui.Success(fmt.Sprintf("lockfile written to %s", ac.lockPath))
			return nil
		},
	}
}
