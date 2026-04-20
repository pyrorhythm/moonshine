package lockfile_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pyrorhythm/moonshine/internal/lockfile"
)

func TestLoadMissing(t *testing.T) {
	lf, err := lockfile.Load("/nonexistent/path/moonfile.lock")
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if lf.Packages == nil {
		t.Error("packages map should be initialized")
	}
}

func TestUpsertAndContains(t *testing.T) {
	lf := lockfile.New("standalone")
	pkg := lockfile.LockedPackage{Name: "git", Version: "2.41.0", Source: "homebrew/core", InstalledAt: time.Now()}
	lf.Upsert("brew", pkg)

	if !lf.Contains("brew", "git") {
		t.Error("expected git to be in lockfile")
	}
	if lf.Contains("brew", "unknown") {
		t.Error("expected unknown to not be in lockfile")
	}

	// Update existing
	pkg.Version = "2.42.0"
	lf.Upsert("brew", pkg)
	got, ok := lf.Get("brew", "git")
	if !ok {
		t.Fatal("git not found after upsert")
	}
	if got.Version != "2.42.0" {
		t.Errorf("version = %q, want 2.42.0", got.Version)
	}
}

func TestRemove(t *testing.T) {
	lf := lockfile.New("standalone")
	lf.Upsert("brew", lockfile.LockedPackage{Name: "git"})
	lf.Upsert("brew", lockfile.LockedPackage{Name: "ripgrep"})
	lf.Remove("brew", "git")

	if lf.Contains("brew", "git") {
		t.Error("git should have been removed")
	}
	if !lf.Contains("brew", "ripgrep") {
		t.Error("ripgrep should still be present")
	}
}

func TestSaveRoundtrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "moonfile.lock")
	lf := lockfile.New("companion")
	lf.Upsert("brew", lockfile.LockedPackage{Name: "bat", Version: "0.24.0", Source: "homebrew/core"})

	if err := lockfile.Save(path, lf); err != nil {
		t.Fatalf("save: %v", err)
	}

	lf2, err := lockfile.Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if lf2.Mode != "companion" {
		t.Errorf("mode = %q, want companion", lf2.Mode)
	}
	got, ok := lf2.Get("brew", "bat")
	if !ok {
		t.Fatal("bat not found after roundtrip")
	}
	if got.Version != "0.24.0" {
		t.Errorf("version = %q, want 0.24.0", got.Version)
	}
}

func TestSaveAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "moonfile.lock")
	// Write initial content
	if err := os.WriteFile(path, []byte("initial"), 0644); err != nil {
		t.Fatal(err)
	}
	lf := lockfile.New("standalone")
	if err := lockfile.Save(path, lf); err != nil {
		t.Fatalf("save: %v", err)
	}
	// Original file should be replaced cleanly
	data, _ := os.ReadFile(path)
	if string(data) == "initial" {
		t.Error("file was not overwritten")
	}
}
