package commands

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/urfave/cli/v3"
	"pyrorhythm.dev/moonshine/internal/config"
	"pyrorhythm.dev/moonshine/internal/packages"
	"pyrorhythm.dev/moonshine/internal/state"
	"pyrorhythm.dev/moonshine/internal/ui"
	"pyrorhythm.dev/moonshine/pkg/backend"
)

func listCommand() *cli.Command {
	return &cli.Command{
		Name:    "list",
		Aliases: []string{"ls"},
		Usage:   "browse and manage installed packages interactively",
		Action: func(ctx context.Context, c *cli.Command) error {
			ac, err := loadContext(c)
			if err != nil {
				return err
			}

			ss, err := state.Snapshot(ctx, ac.registry)
			if err != nil {
				return fmt.Errorf("snapshot: %w", err)
			}

			// Index moonfile packages for O(1) managed lookup.
			managed := make(map[string]bool, len(ac.moonfile.Packages))
			for _, pkg := range ac.moonfile.Packages {
				managed[pkg.PackageManager+"/"+pkg.BinaryName()] = true
			}

			var entries []ui.PackageEntry
			for backendName, pkgs := range ss {
				for name, pkg := range pkgs {
					entries = append(entries, ui.PackageEntry{
						Name:    name,
						Version: pkg.GetVersion(),
						Backend: backendName,
						Managed: managed[backendName+"/"+name],
					})
				}
			}

			sort.Slice(entries, func(i, j int) bool {
				if entries[i].Backend != entries[j].Backend {
					return entries[i].Backend < entries[j].Backend
				}
				return entries[i].Name < entries[j].Name
			})

			result, err := ui.RunPackagesList(entries)
			if err != nil {
				return err
			}

			modified := false
			for _, e := range result.Added {
				ac.moonfile.Packages = append(ac.moonfile.Packages, entryToPackage(e))
				ui.Success(fmt.Sprintf("added %s/%s to packages.yml", e.Backend, e.Name))
				modified = true
			}
			for _, e := range result.Removed {
				keep := ac.moonfile.Packages[:0]
				for _, p := range ac.moonfile.Packages {
					if p.PackageManager == e.Backend && p.BinaryName() == e.Name {
						continue
					}
					keep = append(keep, p)
				}
				ac.moonfile.Packages = keep
				ui.Info(fmt.Sprintf("removed %s/%s from packages.yml", e.Backend, e.Name))
				modified = true
			}
			if modified {
				if err := config.SavePackages(ac.configPath, ac.moonfile.Packages); err != nil {
					return fmt.Errorf("saving packages.yml: %w", err)
				}
			}

			if result.Upgrade != nil {
				return runUpgrade(ctx, ac, result.Upgrade)
			}
			return nil
		},
	}
}

// entryToPackage converts a PackageEntry back to a packages.Package for moonfile storage.
func entryToPackage(e ui.PackageEntry) packages.Package {
	meta := make(map[string]string)
	switch e.Backend {
	case "brew":
		// FQN "org/tap/name" → separate tap and name fields.
		if strings.Count(e.Name, "/") >= 2 {
			idx := strings.LastIndex(e.Name, "/")
			meta["tap"] = e.Name[:idx]
			meta["name"] = e.Name[idx+1:]
		} else {
			meta["name"] = e.Name
		}
	case "go":
		meta["link"] = e.Name
	default:
		meta["name"] = e.Name
	}
	return packages.Package{PackageManager: e.Backend, Meta: meta}
}

// runUpgrade upgrades a single package from a PackageEntry.
func runUpgrade(ctx context.Context, ac *appContext, e *ui.PackageEntry) error {
	b, ok := ac.registry.Get(e.Backend)
	if !ok {
		return fmt.Errorf("backend %q not registered", e.Backend)
	}

	// Find full package metadata from moonfile for a richer upgrade call.
	var pkg *packages.Package
	for _, p := range ac.moonfile.Packages {
		if p.PackageManager == e.Backend && p.BinaryName() == e.Name {
			cp := p
			pkg = &cp
			break
		}
	}
	if pkg == nil {
		ui.Warn(fmt.Sprintf("%s/%s not in packages.yml; cannot upgrade an unmanaged package", e.Backend, e.Name))
		return nil
	}

	ui.Info(fmt.Sprintf("upgrading %s/%s…", e.Backend, e.Name))
	bpkg := backend.Package{PackageManager: e.Backend, Meta: pkg.Meta}
	if err := b.Upgrade(ctx, bpkg); err != nil {
		return fmt.Errorf("upgrading %s: %w", e.Name, err)
	}
	ui.Success(fmt.Sprintf("upgraded %s/%s", e.Backend, e.Name))
	return nil
}
