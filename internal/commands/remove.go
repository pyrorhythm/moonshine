package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/urfave/cli/v3"
	"pyrorhythm.dev/moonshine/internal/config"
	"pyrorhythm.dev/moonshine/internal/ui"
)

func removeCommand() *cli.Command {
	return &cli.Command{
		Name:      "remove",
		Aliases:   []string{"rm"},
		Usage:     "remove a package from moonpackages and uninstall",
		ArgsUsage: "[backend#]package",
		Action: func(ctx context.Context, c *cli.Command) error {
			if c.NArg() == 0 {
				return errors.New(
					"package required — format: [backend#]name  e.g. brew#node, go#gopls",
				)
			}

			ref, err := parsePackageRef(c.Args().First())
			if err != nil {
				return err
			}

			ac, err := loadContext(c)
			if err != nil {
				return err
			}

			found := false
			updated := ac.moonfile.Packages[:0]
			for _, p := range ac.moonfile.Packages {
				if p.PackageManager == ref.backend && p.BinaryName() == ref.name {
					found = true
					continue
				}
				updated = append(updated, p)
			}
			if !found {
				return fmt.Errorf(
					"package %q not found in moonpackages.yml under %s",
					ref.name,
					ref.backend,
				)
			}

			ac.moonfile.Packages = updated
			if err := config.SavePackages(ac.configPath, ac.moonfile.Packages); err != nil {
				return fmt.Errorf("saving moonpackages.yml: %w", err)
			}
			ui.Info(fmt.Sprintf("removed %s/%s from moonpackages.yml", ref.backend, ref.name))

			return applyAC(ctx, ac)
		},
	}
}
