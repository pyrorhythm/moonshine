package reconciler

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"pyrorhythm.dev/moonshine/internal/hooks"
	"pyrorhythm.dev/moonshine/internal/lockfile"
	"pyrorhythm.dev/moonshine/internal/registry"
	"pyrorhythm.dev/moonshine/pkg/backend"
)

// ProgressReporter receives live events during Apply.
// All methods must be safe for concurrent calls.
type ProgressReporter interface {
	OnStart(pkg string)
	OnLog(pkg, line string)
	OnDone(pkg string, err error)
	OnAllDone()
}

// ApplyOptions configures the Apply call.
type ApplyOptions struct {
	DryRun   bool
	Verbose  bool
	Hooks    hooks.Hooks
	Mode     string
	Progress ProgressReporter // nil = no live progress reporting
}

// progressWriter buffers subprocess output and forwards complete lines to a ProgressReporter.
type progressWriter struct {
	pkg string
	r   ProgressReporter
	buf []byte
}

func (w *progressWriter) Write(p []byte) (int, error) {
	w.buf = append(w.buf, p...)
	for {
		idx := bytes.IndexByte(w.buf, '\n')
		if idx < 0 {
			break
		}
		line := strings.TrimRight(string(w.buf[:idx]), "\r")
		if line != "" {
			w.r.OnLog(w.pkg, line)
		}
		w.buf = w.buf[idx+1:]
	}
	return len(p), nil
}

func (w *progressWriter) flush() {
	if line := strings.TrimRight(string(w.buf), "\r\n"); line != "" {
		w.r.OnLog(w.pkg, line)
	}
	w.buf = nil
}

// Apply executes the reconciliation plan, calling the appropriate backend for each action.
// Install and upgrade actions run in parallel; uninstalls run serially afterward.
func Apply(
	ctx context.Context,
	plan DiffResult,
	reg *registry.Registry,
	lf *lockfile.LockFile,
	opts ApplyOptions,
) error {
	if opts.Progress != nil {
		defer opts.Progress.OnAllDone()
	}
	if err := hooks.Run(ctx, opts.Hooks.PreApply, hooks.Env{Action: "apply", Mode: opts.Mode}); err != nil {
		return fmt.Errorf("pre_apply hook: %w", err)
	}

	sorted := sortActions(plan.Actions)

	var installs, uninstalls []PackageAction
	for _, a := range sorted {
		switch a.Kind {
		case ActionInstall, ActionUpgrade:
			installs = append(installs, a)
		case ActionUninstall:
			uninstalls = append(uninstalls, a)
		case ActionNone:
		}
	}

	if err := runInstallsParallel(ctx, installs, reg, lf, opts); err != nil {
		return err
	}

	for _, action := range uninstalls {
		if err := runUninstall(ctx, action, reg, lf, opts); err != nil {
			return err
		}
	}

	if err := hooks.Run(ctx, opts.Hooks.PostApply, hooks.Env{Action: "apply", Mode: opts.Mode}); err != nil {
		return fmt.Errorf("post_apply hook: %w", err)
	}
	return nil
}

func runInstallsParallel(
	ctx context.Context,
	actions []PackageAction,
	reg *registry.Registry,
	lf *lockfile.LockFile,
	opts ApplyOptions,
) error {
	if len(actions) == 0 {
		return nil
	}
	errs := make([]error, len(actions))
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, action := range actions {
		wg.Go(func() {
			errs[i] = runInstall(ctx, action, reg, lf, opts, &mu)
		})
	}
	wg.Wait()

	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func runInstall(
	ctx context.Context,
	action PackageAction,
	reg *registry.Registry,
	lf *lockfile.LockFile,
	opts ApplyOptions,
	mu *sync.Mutex,
) (retErr error) {
	b, ok := reg.Get(action.BackendName)
	if !ok {
		return fmt.Errorf("backend %q not registered", action.BackendName)
	}

	name := action.DisplayName()
	if opts.Progress != nil {
		opts.Progress.OnStart(name)
		pw := &progressWriter{pkg: name, r: opts.Progress}
		ctx = backend.WithOutput(ctx, pw)
		defer func() {
			pw.flush()
			opts.Progress.OnDone(name, retErr)
		}()
	}

	env := actionEnv(action, opts)
	env.Action = action.Kind.String()

	if err := hooks.Run(ctx, opts.Hooks.PreInstall, env); err != nil {
		return fmt.Errorf("pre_install hook: %w", err)
	}
	if !opts.DryRun {
		if err := b.Install(ctx, action.Package); err != nil {
			return fmt.Errorf("installing %s/%s: %w", action.BackendName, action.Package.Name(), err)
		}
		mu.Lock()
		lf.Upsert(action.BackendName, lockfile.LockedPackage{
			Name:        action.Package.Name(),
			Version:     action.Package.Get("version"),
			Source:      action.BackendName,
			InstalledAt: time.Now().UTC(),
		})
		mu.Unlock()
	}
	if err := hooks.Run(ctx, opts.Hooks.PostInstall, env); err != nil {
		return fmt.Errorf("post_install hook: %w", err)
	}
	return nil
}

func runUninstall(
	ctx context.Context,
	action PackageAction,
	reg *registry.Registry,
	lf *lockfile.LockFile,
	opts ApplyOptions,
) (retErr error) {
	b, ok := reg.Get(action.BackendName)
	if !ok {
		return fmt.Errorf("backend %q not registered", action.BackendName)
	}

	name := action.DisplayName()
	if opts.Progress != nil {
		opts.Progress.OnStart(name)
		pw := &progressWriter{pkg: name, r: opts.Progress}
		ctx = backend.WithOutput(ctx, pw)
		defer func() {
			pw.flush()
			opts.Progress.OnDone(name, retErr)
		}()
	}

	env := actionEnv(action, opts)
	env.Action = "uninstall"

	if err := hooks.Run(ctx, opts.Hooks.PreRemove, env); err != nil {
		return fmt.Errorf("pre_remove hook: %w", err)
	}
	if !opts.DryRun {
		pkg := backend.Package{
			PackageManager: action.BackendName,
			Meta:           map[string]string{"name": action.Current.GetName()},
		}
		if err := b.Uninstall(ctx, pkg); err != nil {
			return fmt.Errorf("uninstalling %s/%s: %w", action.BackendName, action.Current.GetName(), err)
		}
		lf.Remove(action.BackendName, action.Current.GetName())
	}
	if err := hooks.Run(ctx, opts.Hooks.PostRemove, env); err != nil {
		return fmt.Errorf("post_remove hook: %w", err)
	}
	return nil
}

func actionEnv(action PackageAction, opts ApplyOptions) hooks.Env {
	env := hooks.Env{Backend: action.BackendName, Mode: opts.Mode}
	if action.Package.Meta != nil {
		env.Package = action.Package.Name()
		env.Version = action.Package.Get("version")
	} else if action.Current != nil {
		env.Package = action.Current.GetName()
		env.Version = action.Current.GetVersion()
	}
	return env
}

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
	case ActionNone:
		return 3
	}
	return 3
}
