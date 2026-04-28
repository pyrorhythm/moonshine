package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/urfave/cli/v3"
	"pyrorhythm.dev/moonshine/internal/config"
	"pyrorhythm.dev/moonshine/internal/ui"
	"pyrorhythm.dev/moonshine/pkg/backend"
)

func addCommand() *cli.Command {
	return &cli.Command{
		Name:      "add",
		Usage:     "search, install, and add a package to packages.yml",
		ArgsUsage: "[backend#]package[@version]",
		Action: func(ctx context.Context, c *cli.Command) error {
			if c.NArg() == 0 {
				return errors.New(
					"package required — format: [backend#]name[@version]  e.g. brew#node@22, go#golang.org/x/tools/gopls",
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

			for _, p := range ac.moonfile.Packages {
				if p.PackageManager == ref.backend && p.BinaryName() == ref.name {
					return fmt.Errorf(
						"package %q already in packages.yml under %s",
						ref.name,
						ref.backend,
					)
				}
			}

			b, ok := ac.registry.Get(ref.backend)
			if !ok {
				return fmt.Errorf("unknown backend %q", ref.backend)
			}

			if searchBackend(ctx, b, ref.backend, ref.name) {
				return installAndAdd(ctx, ac, b, ref)
			}

			// Not found in preferred backend — search all others.
			ui.Warn(
				fmt.Sprintf(
					"%q not found in %s, searching other backends...",
					ref.name,
					ref.backend,
				),
			)

			var crossResults []backend.SearchResult
			for _, other := range ac.registry.All() {
				if other.Name() == ref.backend || !other.Available() {
					continue
				}
				s, ok := other.(backend.Searcher)
				if !ok {
					continue
				}
				results, err := s.Search(ctx, ref.name)
				if err != nil {
					continue
				}
				crossResults = append(crossResults, results...)
			}

			if len(crossResults) == 0 {
				return fmt.Errorf("package %q not found in any available backend", ref.name)
			}

			chosen := ui.PickSearchResult(crossResults)
			if chosen == nil {
				ui.Info("aborted")
				return nil
			}

			selectedRef := packageRef{
				backend: chosen.Backend,
				name:    chosen.Name,
				version: ref.version,
			}
			selectedB, _ := ac.registry.Get(chosen.Backend)
			return installAndAdd(ctx, ac, selectedB, selectedRef)
		},
	}
}

// searchBackend checks if name exists in b. Returns true if found or if b
// doesn't implement Searcher (optimistic — let install fail naturally).
func searchBackend(
	ctx context.Context,
	b backend.Backend,
	backendName, name string,
) bool {
	s, ok := b.(backend.Searcher)
	if !ok {
		return true
	}
	ui.Info(fmt.Sprintf("searching %s for %q...", backendName, name))
	results, err := s.Search(ctx, name)
	if err != nil {
		ui.Warn(fmt.Sprintf("search error: %s", err))
		return true
	}
	for _, r := range results {
		if r.Name == name {
			return true
		}
	}
	return false
}

// installAndAdd installs pkg via b then appends it to packages.yml.
func installAndAdd(
	ctx context.Context,
	ac *appContext,
	b backend.Backend,
	ref packageRef,
) error {
	pkg := refToPackage(ref)
	bpkg := backend.Package{PackageManager: pkg.PackageManager, Meta: pkg.Meta}

	ui.Info(fmt.Sprintf("installing %s/%s...", ref.backend, ref.name))
	if err := b.Install(ctx, bpkg); err != nil {
		return fmt.Errorf("install failed: %w", err)
	}

	ac.moonfile.Packages = append(ac.moonfile.Packages, pkg)
	if err := config.SavePackages(ac.configPath, ac.moonfile.Packages); err != nil {
		return fmt.Errorf("saving packages.yml: %w", err)
	}
	ui.Success(fmt.Sprintf("added %s/%s to packages.yml", ref.backend, ref.name))
	return nil
}
