package state

import (
	"context"
	"fmt"
	"log/slog"

	"pyrorhythm.dev/moonshine/internal/registry"
	"pyrorhythm.dev/moonshine/pkg/backend"
)

// PackageMap is a set of installed packages keyed by package name.
type PackageMap map[string]backend.InstalledPackage

// SystemState is a snapshot of all installed packages across every backend.
type SystemState map[string]PackageMap

// Snapshot queries every available backend and returns the combined state.
// Unavailable backends are skipped silently.
func Snapshot(ctx context.Context, reg *registry.Registry) (SystemState, error) {
	ss := make(SystemState)
	for _, b := range reg.All() {
		if !b.Available() {
			continue
		}
		pkgs, err := b.ListInstalled(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing %s packages: %w", b.Name(), err)
		}
		pm := make(PackageMap, len(pkgs))
		for _, p := range pkgs {
			pm[p.GetName()] = p
		}
		ss[b.Name()] = pm
	}
	return ss, nil
}

// Get returns the installed package for name in the given backend, if present.
func (ss SystemState) Get(backendName, name string) (backend.InstalledPackage, bool) {
	pm, ok := ss[backendName]
	if !ok {
		return nil, false
	}
	pkg, ok := pm[name]
	if !ok {
		slog.Info("not found", "name", name, "pm", pm)
	}
	return pkg, ok
}
