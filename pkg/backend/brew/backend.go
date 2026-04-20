package brew

import (
	"context"
	"errors"
	"fmt"

	"pyrorhythm.dev/moonshine/pkg/backend"
)

// Backend implements backend.Backend for Homebrew.
// runner handles all CLI operations (install, uninstall, list, etc.).
// api handles metadata queries against formulae.brew.sh.
type Backend struct {
	runner   IRunner
	api      *apiClient
	localTap string
}

// New returns a brew Backend using the real Runner and the public API.
func New(localTap string, verbose bool) (*Backend, error) {
	r, err := NewRunner(verbose)
	if err != nil {
		return nil, err
	}
	return &Backend{runner: r, api: newAPIClient(), localTap: localTap}, nil
}

// NewWithRunner creates a Backend with a custom runner (for testing).
// api is set to the real client; pass a nil api to disable network calls in tests.
func NewWithRunner(r IRunner, localTap string) *Backend {
	return &Backend{runner: r, api: newAPIClient(), localTap: localTap}
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

// Search queries formulae.brew.sh for packages matching query.
func (b *Backend) Search(ctx context.Context, query string) ([]backend.SearchResult, error) {
	return b.api.Search(ctx, query)
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

// resolveFormula returns the formula string to pass to `brew install`.
// tap overrides auto-resolution; pinned version triggers a versioned-formula
// check via the API, falling back to brew-extract if not found publicly.
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

	// Pinned: prefer an existing versioned formula (e.g. openssl@3.1.0) before
	// falling back to brew-extract into a local tap.
	candidate := base + "@" + version
	exists, err := b.api.PackageExists(ctx, candidate)
	if err != nil && !errors.Is(err, errNotFound) {
		// Network error — skip the check and fall through to extract.
		exists = false
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
