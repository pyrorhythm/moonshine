package goutil

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"pyrorhythm.dev/moonshine/pkg/backend"
	"pyrorhythm.dev/moonshine/pkg/runenv"
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
			Name:   e.Name(),
			Source: "go",
		})
	}
	return pkgs, nil
}

func (b *Backend) Install(ctx context.Context, pkg backend.Package) error {
	target := installTarget(pkg)
	_, err := b.run(ctx, []string{"install", target}, false)
	return err
}

func (b *Backend) Uninstall(ctx context.Context, pkg backend.Package) error {
	if b.binDir == "" {
		return fmt.Errorf("GOPATH bin directory unknown")
	}
	binName := pkg.Name() // last path segment
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
	_, err := b.run(ctx, []string{"install", installTarget(pkg)}, false)
	return err
}

// installTarget builds the go install argument: link@version.
func installTarget(pkg backend.Package) string {
	link := pkg.Get("link")
	ver := pkg.Get("version")
	if ver == "" {
		ver = "latest"
	}
	return link + "@" + ver
}

func (b *Backend) run(ctx context.Context, args []string, capture bool) ([]byte, error) {
	if b.goPath == "" {
		return nil, fmt.Errorf("go not found on PATH")
	}
	var buf bytes.Buffer
	cmd := exec.CommandContext(ctx, b.goPath, args...)
	cmd.Env = runenv.Get()
	log.Printf("env=%+v", cmd.Env)
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
