package commands

import (
	"context"
	"fmt"
	"os"
	"strings"

	"pyrorhythm.dev/moonshine/internal/lockfile"
	"pyrorhythm.dev/moonshine/internal/packages"
	"pyrorhythm.dev/moonshine/internal/reconciler"
	"pyrorhythm.dev/moonshine/internal/state"
	"pyrorhythm.dev/moonshine/internal/ui"
)

func applyAC(ctx context.Context, ac *appContext) error {
	ss, err := state.Snapshot(ctx, ac.registry)
	if err != nil {
		return fmt.Errorf("snapshot: %w", err)
	}

	plan := reconciler.Diff(ac.moonfile, ss, ac.lock)
	if !plan.HasChanges() {
		ui.Success("Already up to date — nothing to do.")
		return nil
	}

	ui.PrintDiff(os.Stdout, plan)
	fmt.Println()

	if ac.dryRun {
		ui.Warn("dry-run: no changes made")
		return nil
	}

	opts := reconciler.ApplyOptions{
		DryRun:  ac.dryRun,
		Verbose: ac.verbose,
		Hooks:   ac.moonfile.Hooks,
		Mode:    string(ac.moonfile.Mode),
	}
	if err := reconciler.Apply(ctx, plan, ac.registry, ac.lock, opts); err != nil {
		return err
	}

	if err := lockfile.Save(ac.lockPath, ac.lock); err != nil {
		ui.Warn("failed to save lockfile: " + err.Error())
	}

	ui.Success("Apply complete.")
	return nil
}

// packageRef is the parsed result of a [backend#]name[@version] argument.
type packageRef struct {
	backend string
	name    string
	version string
}

// parsePackageRef parses "[backend#]name[@version]".
// Default backend is "brew". For go packages, name is the full install link.
func parsePackageRef(s string) (packageRef, error) {
	ref := packageRef{backend: "brew"}

	if idx := strings.IndexByte(s, '#'); idx >= 0 {
		ref.backend = s[:idx]
		s = s[idx+1:]
	}

	// For version: split on last '@' to handle scoped npm packages like @scope/pkg@1.0.0
	if idx := strings.LastIndexByte(s, '@'); idx > 0 {
		ref.name = s[:idx]
		ref.version = s[idx+1:]
	} else {
		ref.name = s
	}

	if ref.name == "" {
		return ref, fmt.Errorf("empty package name in %q", s)
	}
	return ref, nil
}

// refToPackage converts a packageRef to a packages.Package for adding to moonpackages.
func refToPackage(ref packageRef) packages.Package {
	meta := make(map[string]string)
	meta["name"] = ref.name
	if ref.version != "" {
		meta["version"] = ref.version
	}

	switch ref.backend {
	case "brew":
		if strings.Count(ref.name, "/") == 2 {
			lbi := strings.LastIndexByte(ref.name, '/')
			meta["tap"] = ref.name[:lbi]
			meta["name"] = ref.name[lbi+1:]
		}
	case "go":
		meta["link"] = ref.name
		delete(meta, "name")
	}
	return packages.Package{PackageManager: ref.backend, Meta: meta}
}
