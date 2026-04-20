package reconciler

import (
	"context"
	"fmt"
	"sort"
	"time"

	"pyrorhythm.dev/moonshine/internal/config"
	"pyrorhythm.dev/moonshine/internal/hooks"
	"pyrorhythm.dev/moonshine/internal/lockfile"
	"pyrorhythm.dev/moonshine/internal/registry"
	"pyrorhythm.dev/moonshine/pkg/backend"
)

// ApplyOptions configures the Apply call.
type ApplyOptions struct {
	DryRun  bool
	Verbose bool
	Hooks   hooks.Hooks
	Mode    string
}

// Apply executes the reconciliation plan, calling the appropriate backend for each action.
func Apply(
	ctx context.Context,
	plan DiffResult,
	reg *registry.Registry,
	mf *config.Moonfile,
	lf *lockfile.LockFile,
	opts ApplyOptions,
) error {
	if err := hooks.Run(
		ctx,
		opts.Hooks.PreApply,
		hooks.Env{Action: "apply", Mode: opts.Mode},
	); err != nil {
		return fmt.Errorf("pre_apply hook: %w", err)
	}

	sorted := sortActions(plan.Actions)

	for _, action := range sorted {
		if action.Kind == ActionNone {
			continue
		}

		b, ok := reg.Get(action.BackendName)
		if !ok {
			return fmt.Errorf("backend %q not registered", action.BackendName)
		}

		env := hooks.Env{
			Backend: action.BackendName,
			Mode:    opts.Mode,
		}
		if action.Package.Meta != nil {
			env.Package = action.Package.Name()
			env.Version = action.Package.Get("version")
		} else if action.Current != nil {
			env.Package = action.Current.Name
			env.Version = action.Current.Version
		}

		switch action.Kind {
		case ActionInstall, ActionUpgrade:
			env.Action = action.Kind.String()
			if err := hooks.Run(ctx, opts.Hooks.PreInstall, env); err != nil {
				return fmt.Errorf("pre_install hook: %w", err)
			}
			if !opts.DryRun {
				if err := b.Install(ctx, action.Package); err != nil {
					return fmt.Errorf(
						"installing %s/%s: %w",
						action.BackendName,
						action.Package.Name(),
						err,
					)
				}
				lf.Upsert(action.BackendName, lockfile.LockedPackage{
					Name:        action.Package.Name(),
					Version:     action.Package.Get("version"),
					Source:      action.BackendName,
					InstalledAt: time.Now().UTC(),
				})
			}
			if err := hooks.Run(ctx, opts.Hooks.PostInstall, env); err != nil {
				return fmt.Errorf("post_install hook: %w", err)
			}

		case ActionUninstall:
			env.Action = "uninstall"
			if err := hooks.Run(ctx, opts.Hooks.PreRemove, env); err != nil {
				return fmt.Errorf("pre_remove hook: %w", err)
			}
			if !opts.DryRun {
				pkg := backend.Package{
					PackageManager: action.BackendName,
					Meta:           map[string]string{"name": action.Current.Name},
				}
				if err := b.Uninstall(ctx, pkg); err != nil {
					return fmt.Errorf(
						"uninstalling %s/%s: %w",
						action.BackendName,
						action.Current.Name,
						err,
					)
				}
				lf.Remove(action.BackendName, action.Current.Name)
			}
			if err := hooks.Run(ctx, opts.Hooks.PostRemove, env); err != nil {
				return fmt.Errorf("post_remove hook: %w", err)
			}
		}
	}

	_ = mf

	if err := hooks.Run(
		ctx,
		opts.Hooks.PostApply,
		hooks.Env{Action: "apply", Mode: opts.Mode},
	); err != nil {
		return fmt.Errorf("post_apply hook: %w", err)
	}
	return nil
}

// sortActions orders actions: pinned installs/upgrades first, then unpinned, then uninstalls.
func sortActions(actions []PackageAction) []PackageAction {
	sorted := make([]PackageAction, len(actions))
	copy(sorted, actions)
	sort.SliceStable(sorted, func(i, j int) bool {
		return priority(sorted[i]) < priority(sorted[j])
	})
	return sorted
}

func priority(a PackageAction) int {
	switch a.Kind {
	case ActionInstall, ActionUpgrade:
		if a.Package.IsPinned() {
			return 0
		}
		return 1
	case ActionUninstall:
		return 2
	default:
		return 3
	}
}
