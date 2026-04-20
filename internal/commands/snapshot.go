package commands

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pyrorhythm/moonshine/internal/config"
	"github.com/pyrorhythm/moonshine/internal/lockfile"
	"github.com/pyrorhythm/moonshine/internal/packages"
	"github.com/pyrorhythm/moonshine/internal/state"
	"github.com/pyrorhythm/moonshine/internal/ui"
	"github.com/urfave/cli/v2"
)

func snapshotCommand() *cli.Command {
	return &cli.Command{
		Name:  "snapshot",
		Usage: "capture current installed packages into moonpackages.yml (companion bootstrap)",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Value:   defaultConfigPath(),
				Usage:   "output moonconfig.yml path (moonpackages.yml + .lock written alongside)",
			},
			&cli.StringFlag{
				Name:    "backend",
				Aliases: []string{"b"},
				Usage:   "snapshot only this backend",
			},
		},
		Action: func(c *cli.Context) error {
			output := c.String("output")
			backendFilter := c.String("backend")

			mf := config.NewMoonfile("companion")
			reg, err := buildDefaultRegistry(false)
			if err != nil {
				return err
			}

			ss, err := state.Snapshot(c.Context, reg)
			if err != nil {
				return fmt.Errorf("snapshot: %w", err)
			}

			// Lockfile records installed versions; moonpackages.yml only holds names.
			// This way packages float to latest on install, and the lock preserves what was
			// installed at snapshot time without pinning the moonpackages declaration.
			lockPath := strings.TrimSuffix(output, ".yml") + ".lock"
			lf := lockfile.New("companion")

			for backendName, pm := range ss {
				if backendFilter != "" && backendName != backendFilter {
					continue
				}
				for name, installed := range pm {
					// Moonpackages entry: name only, no version pin.
					mf.Packages = append(mf.Packages, packages.Package{
						PackageManager: backendName,
						Meta:           map[string]string{"name": name},
					})
					// Lockfile entry: records the version present at snapshot time.
					lf.Upsert(backendName, lockfile.LockedPackage{
						Name:        name,
						Version:     installed.Version,
						Source:      installed.Source,
						InstalledAt: time.Now().UTC(),
					})
				}
			}

			if _, err := os.Stat(output); err == nil {
				ui.Warn(fmt.Sprintf("%s already exists; overwrite? (ctrl-c to abort)", output))
			}

			if err := config.SaveMoonfile(output, mf); err != nil {
				return fmt.Errorf("writing moonconfig: %w", err)
			}
			if err := lockfile.Save(lockPath, lf); err != nil {
				return fmt.Errorf("writing lockfile: %w", err)
			}
			ui.Success(
				fmt.Sprintf("snapshot written to %s, moonpackages.yml, and %s", output, lockPath),
			)
			return nil
		},
	}
}
