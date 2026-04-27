package commands

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/urfave/cli/v3"
	"pyrorhythm.dev/moonshine/internal/config"
	"pyrorhythm.dev/moonshine/internal/lockfile"
	"pyrorhythm.dev/moonshine/internal/snapshot"
	"pyrorhythm.dev/moonshine/internal/state"
	"pyrorhythm.dev/moonshine/internal/ui"
)

func snapshotCommand() *cli.Command {
	return &cli.Command{
		Name:  "snapshot",
		Usage: "capture current installed packages into packages.yml",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Value:   defaultConfigPath(),
				Usage:   "output config.yml path (packages.yml + .lock written alongside)",
			},
			&cli.StringSliceFlag{
				Name:    "backend",
				Aliases: []string{"b"},
				Usage:   "snapshot only these backends",
			},
			&cli.BoolFlag{
				Name:    "overwrite-config",
				Usage:   "overwrite existing config",
				Aliases: []string{"f", "overwrite"},
				Value:   false,
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			output := c.String("output")
			backendFilter := c.StringSlice("backend")

			mf := config.NewMoonfile("companion")
			reg := buildDefaultRegistry(false)

			ss, err := state.Snapshot(ctx, reg)
			if err != nil {
				return fmt.Errorf("snapshot: %w", err)
			}

			lockPath := filepath.Join(filepath.Dir(output), "moonshine.lock")

			res := snapshot.Capture(ctx, ss, backendFilter)

			mf.Packages = res.Packages
			lf := res.Lockfile

			if c.Bool("overwrite-config") {
				if err := config.SaveConfig(output, mf); err != nil {
					return fmt.Errorf("writing moonconfig: %w", err)
				}

				ui.Success(
					fmt.Sprintf("overwritten config at %s", output),
				)
			}

			if err := config.SavePackages(output, mf.Packages); err != nil {
				return fmt.Errorf("writing packages: %w", err)
			}

			if err := lockfile.Save(lockPath, lf); err != nil {
				return fmt.Errorf("writing lockfile: %w", err)
			}

			ui.Success(
				fmt.Sprintf(
					"snapshot written to moonpackages.yml and %s",
					filepath.Base(lockPath),
				),
			)

			return nil
		},
	}
}
