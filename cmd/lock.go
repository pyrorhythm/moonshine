package main

import (
	"fmt"
	"time"

	"github.com/pyrorhythm/moonshine/internal/lockfile"
	"github.com/pyrorhythm/moonshine/internal/state"
	"github.com/pyrorhythm/moonshine/internal/ui"
	"github.com/urfave/cli/v2"
)

func lockCommand() *cli.Command {
	return &cli.Command{
		Name:  "lock",
		Usage: "regenerate moonshine.lock from current installed state",
		Action: func(c *cli.Context) error {
			ac, err := loadContext(c)
			if err != nil {
				return err
			}
			ss, err := state.Snapshot(c.Context, ac.registry)
			if err != nil {
				return fmt.Errorf("snapshot: %w", err)
			}

			newLock := lockfile.New(string(ac.moonfile.Mode))
			for _, dp := range ac.moonfile.Packages {
				binaryName := dp.BinaryName()
				if installed, ok := ss.Get(dp.PackageManager, binaryName); ok {
					newLock.Upsert(dp.PackageManager, lockfile.LockedPackage{
						Name:        binaryName,
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
