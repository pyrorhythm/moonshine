package main

import (
	"fmt"

	"github.com/pyrorhythm/moonshine/internal/config"
	"github.com/pyrorhythm/moonshine/internal/packages"
	"github.com/pyrorhythm/moonshine/internal/ui"
	"github.com/urfave/cli/v2"
)

func addCommand() *cli.Command {
	return &cli.Command{
		Name:      "add",
		Usage:     "add a package to moonpackages and apply",
		ArgsUsage: "<package>",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "version", Aliases: []string{"V"}, Usage: "pin to a specific version"},
			&cli.StringFlag{Name: "backend", Aliases: []string{"b"}, Value: "brew", Usage: "backend to use (brew, go, cargo, npm, ...)"},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("package name required")
			}
			pkgName := c.Args().First()
			ver := c.String("version")
			backendName := c.String("backend")

			ac, err := loadContext(c)
			if err != nil {
				return err
			}

			for _, p := range ac.moonfile.Packages {
				if p.PackageManager == backendName && p.BinaryName() == pkgName {
					return fmt.Errorf("package %q already in moonpackages under %s", pkgName, backendName)
				}
			}

			meta := map[string]string{"name": pkgName}
			if ver != "" {
				meta["version"] = ver
			}
			ac.moonfile.Packages = append(ac.moonfile.Packages, packages.Package{
				PackageManager: backendName,
				Meta:           meta,
			})

			if err := config.SavePackages(ac.configPath, ac.moonfile.Packages); err != nil {
				return fmt.Errorf("saving moonpackages: %w", err)
			}
			ui.Info(fmt.Sprintf("added %s/%s to moonpackages", backendName, pkgName))

			return applyAC(c.Context, ac)
		},
	}
}
