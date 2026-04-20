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
	return b.runner.Uninstall(ctx, formulaBase(pkg))
}

func (b *Backend) Upgrade(ctx context.Context, pkg backend.Package) error {
	return b.runner.Upgrade(ctx, formulaBase(pkg))
}

// formulaBase returns the base formula name, incorporating brew_version variant if set.
// e.g. name=openssl brew_version=3 → "openssl@3"
func formulaBase(pkg backend.Package) string {
	name := pkg.Get("name")
	if bv := pkg.Get("brew_version"); bv != "" {
		return name + "@" + bv
	}
	return name
}

// resolveFormula returns the formula string to install.
// tap overrides auto-resolution; version triggers brew extract for pinning.
func (b *Backend) resolveFormula(ctx context.Context, pkg backend.Package) (string, error) {
	base := formulaBase(pkg)
	version := pkg.Get("version")
	tap := pkg.Get("tap")

	if tap != "" {
		return tap + "/" + base, nil
	}
	if version == "" {
		return base, nil
	}

	// Pinned version: use a versioned formula if it exists, else brew extract.
	candidate := base + "@" + version
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
	if err := b.runner.Extract(ctx, base, version, b.localTap); err != nil {
		return "", fmt.Errorf("extracting %s@%s: %w", base, version, err)
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
