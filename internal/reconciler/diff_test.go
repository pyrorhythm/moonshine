package reconciler_test

import (
	"testing"

	"github.com/pyrorhythm/moonshine/internal/config"
	"github.com/pyrorhythm/moonshine/internal/config/mode"
	"github.com/pyrorhythm/moonshine/internal/lockfile"
	"github.com/pyrorhythm/moonshine/internal/packages"
	"github.com/pyrorhythm/moonshine/internal/reconciler"
	"github.com/pyrorhythm/moonshine/internal/state"
	"github.com/pyrorhythm/moonshine/pkg/backend"
)

func brewPkg(name string, extra ...string) packages.Package {
	meta := map[string]string{"name": name}
	for i := 0; i+1 < len(extra); i += 2 {
		meta[extra[i]] = extra[i+1]
	}
	return packages.Package{PackageManager: "brew", Meta: meta}
}

func makeMoonfile(opMode mode.OperatingMode, pkgs packages.List) *config.Moonfile {
	return &config.Moonfile{
		Moonshine: config.Moonshine{Mode: opMode, LocalTap: "moonshine-local"},
		Packages:  pkgs,
	}
}

func makeState(backendName string, pkgs ...backend.InstalledPackage) state.SystemState {
	pm := make(state.PackageMap)
	for _, p := range pkgs {
		pm[p.Name] = p
	}
	return state.SystemState{backendName: pm}
}

func TestDiff_install(t *testing.T) {
	mf := makeMoonfile(mode.Standalone, packages.List{brewPkg("git")})
	ss := state.SystemState{}
	lf := lockfile.New(string(mode.Standalone))

	result := reconciler.Diff(mf, ss, lf)
	installs := result.ByKind(reconciler.ActionInstall)
	if len(installs) != 1 {
		t.Fatalf("expected 1 install, got %d", len(installs))
	}
	if installs[0].Package.Name() != "git" {
		t.Errorf("expected git, got %q", installs[0].Package.Name())
	}
}

func TestDiff_alreadyInstalled(t *testing.T) {
	mf := makeMoonfile(mode.Standalone, packages.List{brewPkg("ripgrep")})
	ss := makeState("brew", backend.InstalledPackage{Name: "ripgrep", Version: "14.0.0"})
	lf := lockfile.New(string(mode.Standalone))

	result := reconciler.Diff(mf, ss, lf)
	if result.HasChanges() {
		t.Error("expected no changes for already-installed package")
	}
}

func TestDiff_versionMismatch(t *testing.T) {
	mf := makeMoonfile(mode.Standalone, packages.List{brewPkg("git", "version", "2.41.0")})
	ss := makeState("brew", backend.InstalledPackage{Name: "git", Version: "2.39.0"})
	lf := lockfile.New(string(mode.Standalone))

	result := reconciler.Diff(mf, ss, lf)
	upgrades := result.ByKind(reconciler.ActionUpgrade)
	if len(upgrades) != 1 {
		t.Fatalf("expected 1 upgrade, got %d", len(upgrades))
	}
	if upgrades[0].Package.Get("version") != "2.41.0" {
		t.Errorf("upgrade version = %q, want 2.41.0", upgrades[0].Package.Get("version"))
	}
}

func TestDiff_versionMatch_bottleRevision(t *testing.T) {
	mf := makeMoonfile(mode.Standalone, packages.List{brewPkg("git", "version", "2.41.0")})
	ss := makeState("brew", backend.InstalledPackage{Name: "git", Version: "2.41.0_1"})
	lf := lockfile.New(string(mode.Standalone))

	result := reconciler.Diff(mf, ss, lf)
	if result.HasChanges() {
		t.Error("expected no changes when versions match after normalization")
	}
}

func TestDiff_uninstallStandalone(t *testing.T) {
	mf := makeMoonfile(mode.Standalone, packages.List{brewPkg("ripgrep")})
	ss := makeState("brew",
		backend.InstalledPackage{Name: "ripgrep", Version: "14.0.0"},
		backend.InstalledPackage{Name: "bat", Version: "0.24.0"},
	)
	lf := lockfile.New(string(mode.Standalone))
	lf.Upsert("brew", lockfile.LockedPackage{Name: "bat", Version: "0.24.0"})

	result := reconciler.Diff(mf, ss, lf)
	uninstalls := result.ByKind(reconciler.ActionUninstall)
	if len(uninstalls) != 1 {
		t.Fatalf("expected 1 uninstall, got %d", len(uninstalls))
	}
	if uninstalls[0].Current.Name != "bat" {
		t.Errorf("expected bat, got %q", uninstalls[0].Current.Name)
	}
}

func TestDiff_noUninstallCompanion(t *testing.T) {
	mf := makeMoonfile(mode.Companion, packages.List{brewPkg("ripgrep")})
	ss := makeState("brew",
		backend.InstalledPackage{Name: "ripgrep", Version: "14.0.0"},
		backend.InstalledPackage{Name: "bat", Version: "0.24.0"},
	)
	lf := lockfile.New(string(mode.Companion))
	lf.Upsert("brew", lockfile.LockedPackage{Name: "bat"})

	result := reconciler.Diff(mf, ss, lf)
	uninstalls := result.ByKind(reconciler.ActionUninstall)
	if len(uninstalls) != 0 {
		t.Errorf("companion mode must not uninstall packages, got %d uninstalls", len(uninstalls))
	}
}

func TestDiff_noUninstallNotOurs(t *testing.T) {
	mf := makeMoonfile(mode.Standalone, packages.List{brewPkg("ripgrep")})
	ss := makeState("brew",
		backend.InstalledPackage{Name: "ripgrep", Version: "14.0.0"},
		backend.InstalledPackage{Name: "bat", Version: "0.24.0"},
	)
	lf := lockfile.New(string(mode.Standalone))

	result := reconciler.Diff(mf, ss, lf)
	uninstalls := result.ByKind(reconciler.ActionUninstall)
	if len(uninstalls) != 0 {
		t.Errorf("should not uninstall packages moonshine didn't install, got %d", len(uninstalls))
	}
}
