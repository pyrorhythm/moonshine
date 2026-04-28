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
			pkgs[i] = fromLeavesName(name)
		}
		return pkgs, nil
	}

	byName := make(map[string]FormulaInfo, len(infos))
	for _, info := range infos {
		byName[info.FullName] = info
		byName[info.Name] = info
	}

	pkgs := make([]backend.InstalledPackage, 0, len(names))
	for _, name := range names {
		pkg := fromLeavesName(name)
		if info, ok := byName[name]; ok {
			pkg.Name = info.Name
			pkg.Tap = info.Tap
			pkg.Description = info.Desc
			if len(info.Installed) > 0 {
				pkg.Version = info.Installed[0].Version
			}
		}
		pkgs = append(pkgs, pkg)
	}
	return pkgs, nil
}

func (b *Backend) Install(ctx context.Context, pkg backend.Package) error {
	formula, err := b.resolveFormula(ctx, fromBackend(pkg))
	if err != nil {
		return err
	}
	return b.runner.Install(ctx, formula)
}

func (b *Backend) Uninstall(ctx context.Context, pkg backend.Package) error {
	return b.runner.Uninstall(ctx, fromBackend(pkg).FormulaRef())
}

func (b *Backend) Upgrade(ctx context.Context, pkg backend.Package) error {
	return b.runner.Upgrade(ctx, fromBackend(pkg).FormulaRef())
}

func (b *Backend) Search(ctx context.Context, query string) ([]backend.SearchResult, error) {
	return b.api.Search(ctx, query)
}

// resolveFormula determines the exact formula string to pass to brew install.
// For tapped formulae it ensures the tap is registered. For version-pinned core
// formulae it checks for a versioned formula and falls back to brew extract.
func (b *Backend) resolveFormula(ctx context.Context, pkg Package) (string, error) {
	if pkg.Tap != "" {
		return b.resolveTap(ctx, pkg)
	}
	if pkg.Version == "" {
		return pkg.FormulaRef(), nil
	}
	candidate := pkg.FormulaRef() + "@" + pkg.Version
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
	if err := b.runner.Extract(ctx, pkg.Name, pkg.Version, b.localTap); err != nil {
		return "", fmt.Errorf("extracting %s@%s: %w", pkg.Name, pkg.Version, err)
	}
	return b.localTap + "/" + candidate, nil
}

// resolveTap ensures the tap is registered then returns the formula ref.
func (b *Backend) resolveTap(ctx context.Context, pkg Package) (string, error) {
	exists, err := b.runner.TapExists(ctx, pkg.Tap)
	if err != nil {
		return "", fmt.Errorf("checking tap %q: %w", pkg.Tap, err)
	}
	if !exists {
		if err := b.runner.TapAdd(ctx, pkg.Tap); err != nil {
			return "", fmt.Errorf("adding tap %q: %w", pkg.Tap, err)
		}
	}
	ref := pkg.FormulaRef()
	if pkg.Version != "" {
		ref += "@" + pkg.Version
	}
	return ref, nil
}
