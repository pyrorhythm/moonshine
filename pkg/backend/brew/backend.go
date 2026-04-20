package brew

import (
	"context"
	"fmt"

	"github.com/pyrorhythm/moonshine/pkg/backend"
)

// Backend implements backend.Backend for Homebrew.
type Backend struct {
	runner   IRunner
	localTap string
}

// New returns a brew Backend using the real Runner.
func New(localTap string, verbose bool) (*Backend, error) {
	r, err := NewRunner(verbose)
	if err != nil {
		return nil, err
	}
	return &Backend{runner: r, localTap: localTap}, nil
}

// NewWithRunner creates a Backend with a custom runner (for testing).
func NewWithRunner(r IRunner, localTap string) *Backend {
	return &Backend{runner: r, localTap: localTap}
}

func (b *Backend) Name() string { return "brew" }

func (b *Backend) Available() bool {
	_, err := NewRunner(false)
	return err == nil
}

func (b *Backend) ListInstalled(ctx context.Context) ([]backend.InstalledPackage, error) {
	entries, err := b.runner.ListInstalled(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]backend.InstalledPackage, len(entries))
	for i, e := range entries {
		out[i] = backend.InstalledPackage{Name: e.Name, Version: e.Version, Source: "brew"}
	}
	return out, nil
}

func (b *Backend) Install(ctx context.Context, pkg backend.Package) error {
	formula, err := b.resolveFormula(ctx, pkg)
	if err != nil {
		return err
	}
	return b.runner.Install(ctx, formula)
}

func (b *Backend) Uninstall(ctx context.Context, pkg backend.Package) error {
	return b.runner.Uninstall(ctx, pkg.Get("name"))
}

func (b *Backend) Upgrade(ctx context.Context, pkg backend.Package) error {
	return b.runner.Upgrade(ctx, pkg.Get("name"))
}

// resolveFormula returns the formula string to install.
// If a tap is specified it is used directly; otherwise version pinning via brew extract is applied.
func (b *Backend) resolveFormula(ctx context.Context, pkg backend.Package) (string, error) {
	name := pkg.Get("name")
	version := pkg.Get("version")
	tap := pkg.Get("tap")

	if tap != "" && version == "" {
		return tap + "/" + name, nil
	}
	if !pkg.IsPinned() {
		if tap != "" {
			return tap + "/" + name, nil
		}
		return name, nil
	}

	candidate := name + "@" + version
	exists, err := b.runner.FormulaExists(ctx, candidate)
	if err != nil {
		return "", err
	}
	if exists {
		return candidate, nil
	}
	if err := b.runner.TapCreate(ctx, b.localTap); err != nil {
		return "", fmt.Errorf("creating local tap %q: %w", b.localTap, err)
	}
	if err := b.runner.Extract(ctx, name, version, b.localTap); err != nil {
		return "", fmt.Errorf("extracting %s@%s: %w", name, version, err)
	}
	return b.localTap + "/" + candidate, nil
}

// InstalledVersion returns the installed version of name, or "" if not installed.
func (b *Backend) InstalledVersion(ctx context.Context, name string) (string, error) {
	entries, err := b.runner.ListInstalled(ctx)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if e.Name == name {
			return e.Version, nil
		}
	}
	return "", nil
}
