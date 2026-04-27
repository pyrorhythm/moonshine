package brew_test

import (
	"context"
	"testing"

	"pyrorhythm.dev/moonshine/pkg/backend"
	"pyrorhythm.dev/moonshine/pkg/backend/brew"
)

func backendPkg(pm string, meta map[string]string) backend.Package {
	return backend.Package{PackageManager: pm, Meta: meta}
}

type recordingRunner struct {
	installedFormula   string
	uninstalledFormula string
	upgradedFormula    string
}

func (r *recordingRunner) Leaves(context.Context) ([]string, error) { return nil, nil }
func (r *recordingRunner) InfoJSON(context.Context, []string) ([]brew.InfoEntry, error) {
	return nil, nil
}

func (r *recordingRunner) Install(_ context.Context, formula string, _ ...string) error {
	r.installedFormula = formula
	return nil
}

func (r *recordingRunner) Uninstall(_ context.Context, formula string) error {
	r.uninstalledFormula = formula
	return nil
}
func (r *recordingRunner) Extract(context.Context, string, string, string) error { return nil }
func (r *recordingRunner) TapAdd(context.Context, string) error                  { return nil }
func (r *recordingRunner) TapCreate(context.Context, string) error               { return nil }

func (r *recordingRunner) TapExists(
	context.Context,
	string,
) (bool, error) {
	return false, nil
}

func (r *recordingRunner) Upgrade(_ context.Context, formula string) error {
	r.upgradedFormula = formula
	return nil
}

func TestInstallUsesTapFormula(t *testing.T) {
	runner := &recordingRunner{}
	b := brew.NewWithRunner(runner, "")

	pkg := backendPkg("brew", map[string]string{"name": "foo", "tap": "acme/tools"})
	if err := b.Install(context.Background(), pkg); err != nil {
		t.Fatalf("Install returned error: %v", err)
	}
	if runner.installedFormula != "acme/tools/foo" {
		t.Fatalf("installed formula = %q, want %q", runner.installedFormula, "acme/tools/foo")
	}
}

func TestInstallUsesTapVersionedFormulaForPinnedVersion(t *testing.T) {
	runner := &recordingRunner{}
	b := brew.NewWithRunner(runner, "")

	pkg := backendPkg(
		"brew",
		map[string]string{"name": "foo", "tap": "acme/tools", "version": "1.2.3"},
	)
	if err := b.Install(context.Background(), pkg); err != nil {
		t.Fatalf("Install returned error: %v", err)
	}
	if runner.installedFormula != "acme/tools/foo@1.2.3" {
		t.Fatalf("installed formula = %q, want %q", runner.installedFormula, "acme/tools/foo@1.2.3")
	}
}

func TestUninstallAndUpgradeUseTapFormula(t *testing.T) {
	runner := &recordingRunner{}
	b := brew.NewWithRunner(runner, "")

	pkg := backendPkg("brew", map[string]string{"name": "foo", "tap": "acme/tools"})

	if err := b.Uninstall(context.Background(), pkg); err != nil {
		t.Fatalf("Uninstall returned error: %v", err)
	}
	if runner.uninstalledFormula != "acme/tools/foo" {
		t.Fatalf("uninstalled formula = %q, want %q", runner.uninstalledFormula, "acme/tools/foo")
	}

	if err := b.Upgrade(context.Background(), pkg); err != nil {
		t.Fatalf("Upgrade returned error: %v", err)
	}
	if runner.upgradedFormula != "acme/tools/foo" {
		t.Fatalf("upgraded formula = %q, want %q", runner.upgradedFormula, "acme/tools/foo")
	}
}
