package goutil

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"pyrorhythm.dev/moonshine/pkg/backend"
	"pyrorhythm.dev/moonshine/pkg/runenv"
)

var _ backend.Backend = (*Backend)(nil)

// InstalledPackage is the go-specific installed package record.
type InstalledPackage struct {
	Name string
}

func (p InstalledPackage) GetName() string    { return p.Name }
func (p InstalledPackage) GetVersion() string { return "" }
func (p InstalledPackage) GetSource() string  { return "go" }

var _ backend.InstalledPackage = InstalledPackage{}

// Backend implements backend.Backend for go install.
type Backend struct {
	goPath string
	binDir string
}

// New returns a go Backend.
func New() (*Backend, error) {
	goExec, _ := exec.LookPath("go")
	return &Backend{goPath: goExec, binDir: goBinDir()}, nil
}

func (b *Backend) Name() string    { return "go" }
func (b *Backend) Available() bool { return b.goPath != "" }

func (b *Backend) ListInstalled(context.Context) ([]backend.InstalledPackage, error) {
	if b.binDir == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(b.binDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var pkgs []backend.InstalledPackage
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		pkgs = append(pkgs, InstalledPackage{Name: e.Name()})
	}
	return pkgs, nil
}

func (b *Backend) Install(ctx context.Context, pkg backend.Package) error {
	return b.run(ctx, []string{"install", installTarget(pkg)})
}

func (b *Backend) Uninstall(_ context.Context, pkg backend.Package) error {
	if b.binDir == "" {
		return errors.New("GOPATH bin directory unknown")
	}
	binPath := filepath.Join(b.binDir, pkg.Name())
	if runtime.GOOS == "windows" {
		binPath += ".exe"
	}
	return os.Remove(binPath)
}

func (b *Backend) Upgrade(ctx context.Context, pkg backend.Package) error {
	if pkg.IsPinned() {
		return nil
	}
	return b.run(ctx, []string{"install", installTarget(pkg)})
}

func installTarget(pkg backend.Package) string {
	link := pkg.Get("link")
	ver := pkg.Get("version")
	if ver == "" {
		ver = "latest"
	}
	return link + "@" + ver
}

func (b *Backend) run(ctx context.Context, args []string) error { //nolint:gosec
	if b.goPath == "" {
		return errors.New("go not found on PATH")
	}
	cmd := exec.CommandContext(ctx, b.goPath, args...)
	cmd.Env = runenv.Get()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go %v: %w", args, err)
	}
	return nil
}

func goBinDir() string {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		gopath = filepath.Join(home, "go")
	}
	if idx := strings.IndexByte(gopath, ':'); idx != -1 {
		gopath = gopath[:idx]
	}
	return filepath.Join(gopath, "bin")
}
