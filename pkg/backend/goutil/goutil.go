package goutil

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pyrorhythm/moonshine/pkg/backend"
)

var _ backend.Backend = (*Backend)(nil)

type Backend struct {
	goPath string
	binDir string
}

// New returns a go Backend.
func New() (*Backend, error) {
	goExec, _ := exec.LookPath("go")
	return &Backend{
		goPath: goExec,
		binDir: goBinDir(),
	}, nil
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
		pkgs = append(pkgs, backend.InstalledPackage{
			Name:    e.Name(),
			Version: "", // version unknown without go version <binary>
			Source:  "go",
		})
	}
	return pkgs, nil
}

func (b *Backend) Install(ctx context.Context, pkg backend.Package) error {
	var target string
	if pkg.IsPinned() {
		target = pkg.Name + "@" + pkg.Version
	} else {
		target = pkg.Name + "@latest"
	}
	_, err := b.run(ctx, []string{"install", target}, false)
	return err
}

func (b *Backend) Uninstall(ctx context.Context, pkg backend.Package) error {
	// `go` has no uninstall; remove the binary from GOPATH/bin.
	if b.binDir == "" {
		return fmt.Errorf("GOPATH bin directory unknown")
	}
	// binary name is the last path component
	parts := strings.Split(pkg.Name, "/")
	binName := parts[len(parts)-1]
	binPath := filepath.Join(b.binDir, binName)
	if runtime.GOOS == "windows" {
		binPath += ".exe"
	}
	return os.Remove(binPath)
}

func (b *Backend) Upgrade(ctx context.Context, pkg backend.Package) error {
	if pkg.IsPinned() {
		return nil
	}
	_, err := b.run(ctx, []string{"install", pkg.Name + "@latest"}, false)
	return err
}

func (b *Backend) run(ctx context.Context, args []string, capture bool) ([]byte, error) {
	if b.goPath == "" {
		return nil, fmt.Errorf("go not found on PATH")
	}
	var buf bytes.Buffer
	cmd := exec.CommandContext(ctx, b.goPath, args...)
	if capture {
		cmd.Stdout = &buf
	} else {
		cmd.Stdout = os.Stdout
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("go %v: %w", args, err)
	}
	return buf.Bytes(), nil
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
		_ = bufio.NewScanner(nil)
		gopath = gopath[:idx]
	}
	return filepath.Join(gopath, "bin")
}
