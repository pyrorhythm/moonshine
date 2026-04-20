package commands

import (
	"fmt"

	"github.com/pyrorhythm/moonshine/internal/config"
	"github.com/pyrorhythm/moonshine/internal/ui"
	"github.com/urfave/cli/v2"
)

func addCommand() *cli.Command {
	return &cli.Command{
		Name:      "add",
		Usage:     "add a package to moonpackages and apply",
		ArgsUsage: "[backend#]package[@version]",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("package required — format: [backend#]name[@version]  e.g. brew#node@22, go#golang.org/x/tools/gopls")
			}

			ref, err := parsePackageRef(c.Args().First())
			if err != nil {
				return err
			}

			ac, err := loadContext(c)
			if err != nil {
				return err
			}

			for _, p := range ac.moonfile.Packages {
				if p.PackageManager == ref.backend && p.BinaryName() == ref.name {
					return fmt.Errorf("package %q already in moonpackages under %s", ref.name, ref.backend)
				}
			}

			pkg := refToPackage(ref)
			ac.moonfile.Packages = append(ac.moonfile.Packages, pkg)

			if err := config.SavePackages(ac.configPath, ac.moonfile.Packages); err != nil {
				return fmt.Errorf("saving moonpackages.yml: %w", err)
			}
			ui.Info(fmt.Sprintf("added %s/%s to moonpackages.yml", ref.backend, ref.name))

			return applyAC(c.Context, ac)
		},
	}
}
