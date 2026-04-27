package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"pyrorhythm.dev/moonshine/internal/config"
	"pyrorhythm.dev/moonshine/internal/config/mode"
)

const validConfigYAML = `
mode: standalone
local_tap: moonshine-local
`

const validPackagesYAML = `
brew:
  - name: git
    version: "2.41.0"
  - ripgrep
`

func TestLoadMoonfile_valid(t *testing.T) {
	dir := t.TempDir()
	configPath := writeTmpFile(t, dir, "config.yml", validConfigYAML)
	writeTmpFile(t, dir, "packages.yml", validPackagesYAML)

	mf, err := config.LoadBundle(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mf.Mode != mode.Standalone {
		t.Errorf("mode = %q, want standalone", mf.Mode)
	}
	if len(mf.Packages) != 2 {
		t.Errorf("packages = %d, want 2", len(mf.Packages))
	}
	git := mf.Packages[0]
	if !git.Pinned() {
		t.Error("git should be pinned")
	}
	if git.Get("version") != "2.41.0" {
		t.Errorf("git version = %q, want 2.41.0", git.Get("version"))
	}
}

func TestLoadMoonfile_legacyFilenames(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yml")
	writeTmpFile(t, dir, "moonconfig.yml", validConfigYAML)
	writeTmpFile(t, dir, "moonpackages.yml", validPackagesYAML)

	mf, err := config.LoadBundle(configPath)
	if err != nil {
		t.Fatalf("unexpected error loading legacy filenames: %v", err)
	}
	if len(mf.Packages) != 2 {
		t.Errorf("packages = %d, want 2", len(mf.Packages))
	}
}

func TestLoadMoonfile_defaults(t *testing.T) {
	dir := t.TempDir()
	configPath := writeTmpFile(t, dir, "config.yml", "")

	mf, err := config.LoadBundle(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mf.Mode != mode.Standalone {
		t.Errorf("default mode = %q, want standalone", mf.Mode)
	}
	if mf.LocalTap != "moonshine-local" {
		t.Errorf("default local_tap = %q", mf.LocalTap)
	}
	if len(mf.Backends) == 0 || mf.Backends[0] != "brew" {
		t.Errorf("default backends = %v, want [brew]", mf.Backends)
	}
}

func TestLoadMoonfile_invalidMode(t *testing.T) {
	dir := t.TempDir()
	configPath := writeTmpFile(t, dir, "config.yml", "mode: magic\n")
	_, err := config.LoadBundle(configPath)
	if err == nil {
		t.Fatal("expected error for invalid mode")
	}
}

func TestSaveMoonfileRoundtrip(t *testing.T) {
	dir := t.TempDir()
	configPath := writeTmpFile(t, dir, "config.yml", validConfigYAML)
	writeTmpFile(t, dir, "packages.yml", validPackagesYAML)

	mf, _ := config.LoadBundle(configPath)
	if err := config.SaveConfig(configPath, mf); err != nil {
		t.Fatalf("save error: %v", err)
	}
	mf2, err := config.LoadBundle(configPath)
	if err != nil {
		t.Fatalf("reload error: %v", err)
	}
	if len(mf2.Packages) != len(mf.Packages) {
		t.Error("package count changed after roundtrip")
	}
}

func TestNewMoonfile(t *testing.T) {
	mf := config.NewMoonfile("companion")
	if mf.Mode != mode.Companion {
		t.Errorf("mode = %q, want companion", mf.Mode)
	}
	if mf.Packages == nil {
		t.Error("packages should be initialized")
	}
}

func writeTmpFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
