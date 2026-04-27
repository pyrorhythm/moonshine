package brew

import (
	"context"
	"errors"
	"fmt"

	"pyrorhythm.dev/moonshine/pkg/backend"
)

// Backend implements backend.Backend for Homebrew.
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
func NewWithRunner(r IRunner, localTap string) *Backend {
	return &Backend{runner: r, api: newAPIClient(), localTap: localTap}
}

func (b *Backend) Name() string { return "brew" }

func (b *Backend) Available() bool {
	_, err := NewRunner(false)
	return err == nil
}

func (b *Backend) ListInstalled(ctx context.Context) ([]backend.InstalledPackage, error) {
	names, err := b.runner.Leaves(ctx)
	if err != nil {
		return nil, err
	}
	if len(names) == 0 {
		return nil, nil
	}
	infos, err := b.runner.InfoJSON(ctx, names)
	if err != nil {
		pkgs := make([]backend.InstalledPackage, len(names))
		for i, name := range names {
			pkgs[i] = InstalledPackage{Name: name}
		}
		return pkgs, nil
	}
	byName := make(map[string]InfoEntry, len(infos))
	for _, info := range infos {
		byName[info.FullName] = info
		byName[info.Name] = info // fallback
	}
	pkgs := make([]backend.InstalledPackage, 0, len(names))
	for _, name := range names {
		pkg := InstalledPackage{Name: name}
		if info, ok := byName[name]; ok {
			pkg.Description = info.Desc
			pkg.Tap = info.Tap
			if len(info.Installed) > 0 {
				pkg.Version = info.Installed[0].Version
			}
		}
		pkgs = append(pkgs, pkg)
	}
	return pkgs, nil
}

func (b *Backend) Install(ctx context.Context, pkg backend.Package) error {
	formula, err := b.resolveFormula(ctx, pkg)
	if err != nil {
		return err
	}
	return b.runner.Install(ctx, formula)
}

func (b *Backend) Uninstall(ctx context.Context, pkg backend.Package) error {
	return b.runner.Uninstall(ctx, formulaRef(pkg))
}

func (b *Backend) Upgrade(ctx context.Context, pkg backend.Package) error {
	return b.runner.Upgrade(ctx, formulaRef(pkg))
}

// Search queries formulae.brew.sh for packages matching query.
func (b *Backend) Search(ctx context.Context, query string) ([]backend.SearchResult, error) {
	return b.api.Search(ctx, query)
}

func formulaBase(pkg backend.Package) string {
	name := pkg.Get("name")
	if bv := pkg.Get("brew_version"); bv != "" {
		return name + "@" + bv
	}
	return name
}

func formulaRef(pkg backend.Package) string {
	base := formulaBase(pkg)
	if tap := pkg.Get("tap"); tap != "" {
		return tap + "/" + base
	}
	return base
}

func (b *Backend) resolveFormula(ctx context.Context, pkg backend.Package) (string, error) {
	base := formulaBase(pkg)
	version := pkg.Get("version")
	tap := pkg.Get("tap")

	if tap != "" {
		return b.resolveTap(ctx, tap, version, base)
	}

	if version == "" {
		return base, nil
	}

	candidate := base + "@" + version
	exists, err := b.api.PackageExists(ctx, candidate)
	if err != nil && !errors.Is(err, errNotFound) {
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

func (b *Backend) resolveTap(
	ctx context.Context,
	tap string,
	version string,
	base string,
) (string, error) {
	exists, err := b.runner.TapExists(ctx, tap)
	if err != nil {
		return "", fmt.Errorf("checking tap %q: %w", tap, err)
	}
	if !exists {
		if err := b.runner.TapAdd(ctx, tap); err != nil {
			return "", fmt.Errorf("adding tap %q: %w", tap, err)
		}
	}

	if version != "" {
		return tap + "/" + base + "@" + version, nil
	}
	return tap + "/" + base, nil
}
