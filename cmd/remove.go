package main

import (
	"fmt"

	"github.com/pyrorhythm/moonshine/internal/config"
	"github.com/pyrorhythm/moonshine/internal/packages"
	"github.com/pyrorhythm/moonshine/internal/ui"
	"github.com/urfave/cli/v2"
)

func removeCommand() *cli.Command {
	return &cli.Command{
		Name:      "remove",
		Aliases:   []string{"rm"},
		Usage:     "remove a package from moonpackages and uninstall",
		ArgsUsage: "<package>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "backend",
				Aliases: []string{"b"},
				Value:   "brew",
				Usage:   "backend",
			},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("package name required")
			}
			pkgName := c.Args().First()
			backendName := c.String("backend")

			ac, err := loadContext(c)
			if err != nil {
				return err
			}

			found := false
			updated := ac.moonfile.Packages[:0]
			for _, p := range ac.moonfile.Packages {
				if p.PackageManager == backendName && p.BinaryName() == pkgName {
					found = true
					continue
				}
				updated = append(updated, p)
			}
			if !found {
				return fmt.Errorf("package %q not found in moonpackages under %s", pkgName, backendName)
			}

			ac.moonfile.Packages = packages.List(updated)
			if err := config.SavePackages(ac.configPath, ac.moonfile.Packages); err != nil {
				return fmt.Errorf("saving moonpackages: %w", err)
			}
			ui.Info(fmt.Sprintf("removed %s/%s from moonpackages", backendName, pkgName))

			return applyAC(c.Context, ac)
		},
	}
}
