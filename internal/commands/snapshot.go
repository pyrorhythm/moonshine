package commands

import (
	"fmt"
	"os"

	"github.com/pyrorhythm/moonshine/internal/config"
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
				Value:   "moonconfig.yml",
				Usage:   "output moonconfig.yml path (moonpackages.yml written alongside)",
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

			for backendName, pm := range ss {
				if backendFilter != "" && backendName != backendFilter {
					continue
				}
				for name, installed := range pm {
					meta := map[string]string{"name": name}
					if installed.Version != "" {
						meta["version"] = installed.Version
					}
					mf.Packages = append(mf.Packages, packages.Package{
						PackageManager: backendName,
						Meta:           meta,
					})
				}
			}

			if _, err := os.Stat(output); err == nil {
				ui.Warn(fmt.Sprintf("%s already exists; overwrite? (ctrl-c to abort)", output))
			}

			if err := config.SaveMoonfile(output, mf); err != nil {
				return fmt.Errorf("writing moonfile: %w", err)
			}
			ui.Success(fmt.Sprintf("snapshot written to %s and moonpackages.yml", output))
			return nil
		},
	}
}
