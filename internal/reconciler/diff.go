package reconciler

import (
	"log/slog"

	"pyrorhythm.dev/moonshine/internal/config"
	"pyrorhythm.dev/moonshine/internal/config/mode"
	"pyrorhythm.dev/moonshine/internal/lockfile"
	"pyrorhythm.dev/moonshine/internal/packages"
	"pyrorhythm.dev/moonshine/internal/state"
	"pyrorhythm.dev/moonshine/internal/version"
	"pyrorhythm.dev/moonshine/pkg/backend"
)

// ActionKind classifies what needs to happen to a package.
type ActionKind int

const (
	ActionNone      ActionKind = iota
	ActionInstall              // package not present; needs to be installed
	ActionUpgrade              // version mismatch on a pinned package
	ActionUninstall            // package not in manifest (standalone only)
)

func (k ActionKind) String() string {
	switch k {
	case ActionInstall:
		return "install"
	case ActionUpgrade:
		return "upgrade"
	case ActionUninstall:
		return "uninstall"
	case ActionNone:
		return "none"
	}
	return "none"
}

// PackageAction is one item in the reconciliation plan.
type PackageAction struct {
	Kind        ActionKind
	BackendName string
	Package     backend.Package
	Current     backend.InstalledPackage // nil if not yet installed
	Reason      string
}

// DiffResult is the full computed plan for all backends.
type DiffResult struct {
	Actions []PackageAction
}

// HasChanges reports whether any action requires work.
func (d DiffResult) HasChanges() bool {
	for _, a := range d.Actions {
		if a.Kind != ActionNone {
			return true
		}
	}
	return false
}

// ByKind returns all actions of the given kind.
func (d DiffResult) ByKind(k ActionKind) []PackageAction {
	var out []PackageAction
	for _, a := range d.Actions {
		if a.Kind == k {
			out = append(out, a)
		}
	}
	return out
}

// Diff computes the actions needed to reconcile current state with the moonfile.
func Diff(mf *config.Moonfile, current state.SystemState, lf *lockfile.LockFile) DiffResult {
	byBackend := make(map[string][]packages.Package)
	for _, pkg := range mf.Packages {
		byBackend[pkg.PackageManager] = append(byBackend[pkg.PackageManager], pkg)
	}

	var actions []PackageAction

	for backendName, desired := range byBackend {
		desiredSet := make(map[string]bool, len(desired))

		for _, dp := range desired {
			binaryName := dp.BinaryName()
			desiredSet[binaryName] = true

			bpkg := backend.Package{PackageManager: backendName, Meta: dp.Meta}

			installed, found := current.Get(backendName, binaryName)
			if !found {
				slog.Info("not found", "bpkg", bpkg, "binaryName", binaryName)

				actions = append(actions, PackageAction{
					Kind:        ActionInstall,
					BackendName: backendName,
					Package:     bpkg,
					Reason:      "not installed",
				})
				continue
			}

			if dp.Pinned() && !version.Equal(dp.Version(), installed.GetVersion()) {
				actions = append(actions, PackageAction{
					Kind:        ActionUpgrade,
					BackendName: backendName,
					Package:     bpkg,
					Current:     installed,
					Reason:      "version mismatch: have " + installed.GetVersion() + ", want " + dp.Version(),
				})
				continue
			}

			actions = append(actions, PackageAction{
				Kind:        ActionNone,
				BackendName: backendName,
				Package:     bpkg,
				Current:     installed,
			})
		}

		if mf.Mode == mode.Standalone {
			if pm, ok := current[backendName]; ok {
				for name, inst := range pm {
					if desiredSet[name] || !lf.Contains(backendName, name) {
						continue
					}
					actions = append(actions, PackageAction{
						Kind:        ActionUninstall,
						BackendName: backendName,
						Current:     inst,
						Reason:      "not in moonfile",
					})
				}
			}
		}
	}

	return DiffResult{Actions: actions}
}
