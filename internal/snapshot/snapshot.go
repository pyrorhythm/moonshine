package snapshot

import (
	"context"
	"strings"
	"time"

	"pyrorhythm.dev/moonshine/internal/lockfile"
	"pyrorhythm.dev/moonshine/internal/packages"
	"pyrorhythm.dev/moonshine/internal/state"
)

type Result struct {
	Packages []packages.Package
	Lockfile *lockfile.LockFile
}

func Capture(ctx context.Context, st state.SystemState, backendFilter []string) *Result {
	res := &Result{
		Packages: make([]packages.Package, 0),
		Lockfile: lockfile.New("companion"),
	}

	filterMap := make(map[string]bool)
	for _, b := range backendFilter {
		filterMap[b] = true
	}

	for backendName, pm := range st {
		if len(filterMap) > 0 && !filterMap[backendName] {
			continue
		}

		for name, pkg := range pm {
			meta := map[string]string{"name": name}

			// brew tap awareness
			if backendName == "brew" && pkg.GetSource() != "homebrew/core" {
				meta["tap"] = pkg.GetSource()
				// remove tap from name if present
				parts := strings.Split(name, "/")
				if len(parts) > 2 {
					// acme/tools/foo -> foo
					meta["name"] = parts[len(parts)-1]
				}
			}

			res.Packages = append(res.Packages, packages.Package{
				PackageManager: backendName,
				Meta:           meta,
			})

			res.Lockfile.Upsert(backendName, lockfile.LockedPackage{
				Name:        name,
				Version:     pkg.GetVersion(),
				Source:      pkg.GetSource(),
				InstalledAt: time.Now().UTC(),
			})
		}
	}

	return res
}
