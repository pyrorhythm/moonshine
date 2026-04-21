package brew

import (
	"context"
	"testing"

	"pyrorhythm.dev/moonshine/pkg/backend"
)

type recordingRunner struct {
	installedFormula   string
	uninstalledFormula string
	upgradedFormula    string
}

func (r *recordingRunner) ListInstalled(context.Context) ([]ListEntry, error) {
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

func (r *recordingRunner) Extract(context.Context, string, string, string) error {
	return nil
}

func (r *recordingRunner) TapCreate(context.Context, string) error {
	return nil
}

func (r *recordingRunner) TapExists(context.Context, string) (bool, error) {
	return false, nil
}

func (r *recordingRunner) Upgrade(_ context.Context, formula string) error {
	r.upgradedFormula = formula
	return nil
}

func TestInstallUsesTapFormula(t *testing.T) {
	runner := &recordingRunner{}
	b := &Backend{runner: runner}

	pkg := backend.Package{
		PackageManager: "brew",
		Meta: map[string]string{
			"name": "foo",
			"tap":  "acme/tools",
		},
	}

	if err := b.Install(context.Background(), pkg); err != nil {
		t.Fatalf("Install returned error: %v", err)
	}
	if runner.installedFormula != "acme/tools/foo" {
		t.Fatalf("installed formula = %q, want %q", runner.installedFormula, "acme/tools/foo")
	}
}

func TestInstallUsesTapVersionedFormulaForPinnedVersion(t *testing.T) {
	runner := &recordingRunner{}
	b := &Backend{runner: runner}

	pkg := backend.Package{
		PackageManager: "brew",
		Meta: map[string]string{
			"name":    "foo",
			"tap":     "acme/tools",
			"version": "1.2.3",
		},
	}

	if err := b.Install(context.Background(), pkg); err != nil {
		t.Fatalf("Install returned error: %v", err)
	}
	if runner.installedFormula != "acme/tools/foo@1.2.3" {
		t.Fatalf("installed formula = %q, want %q", runner.installedFormula, "acme/tools/foo@1.2.3")
	}
}

func TestUninstallAndUpgradeUseTapFormula(t *testing.T) {
	runner := &recordingRunner{}
	b := &Backend{runner: runner}

	pkg := backend.Package{
		PackageManager: "brew",
		Meta: map[string]string{
			"name": "foo",
			"tap":  "acme/tools",
		},
	}

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
