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
// localTap is the tap name used for version-pinned formulas.
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
	return b.runner.Install(ctx, formula, pkg.Options...)
}

func (b *Backend) Uninstall(ctx context.Context, pkg backend.Package) error {
	return b.runner.Uninstall(ctx, pkg.Name)
}

func (b *Backend) Upgrade(ctx context.Context, pkg backend.Package) error {
	return b.runner.Upgrade(ctx, pkg.Name)
}

// resolveFormula returns the formula to install for pkg.
// For pinned packages it uses brew extract if the versioned formula doesn't exist yet.
func (b *Backend) resolveFormula(ctx context.Context, pkg backend.Package) (string, error) {
	if !pkg.IsPinned() {
		return pkg.Name, nil
	}
	candidate := pkg.Name + "@" + pkg.Version
	exists, err := b.runner.FormulaExists(ctx, candidate)
	if err != nil {
		return "", err
	}
	if exists {
		return candidate, nil
	}
	// Need to extract the versioned formula into the local tap.
	if err := b.runner.TapCreate(ctx, b.localTap); err != nil {
		return "", fmt.Errorf("creating local tap %q: %w", b.localTap, err)
	}
	if err := b.runner.Extract(ctx, pkg.Name, pkg.Version, b.localTap); err != nil {
		return "", fmt.Errorf("extracting %s@%s: %w", pkg.Name, pkg.Version, err)
	}
	return b.localTap + "/" + candidate, nil
}

// InstalledVersion returns the version of name currently installed, or "" if not installed.
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
