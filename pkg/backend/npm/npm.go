package npm

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/pyrorhythm/moonshine/pkg/backend"
	"github.com/pyrorhythm/moonshine/pkg/runenv"
)

var _ backend.Backend = (*Backend)(nil)

// Backend implements backend.Backend for global npm/bun installs.
// bun is preferred when available; falls back to npm.
type Backend struct {
	tool    string // "bun" or "npm"
	toolBin string
}

// New detects bun or npm and returns a Backend.
func New() (*Backend, error) {
	if path, err := exec.LookPath("bun"); err == nil {
		return &Backend{tool: "bun", toolBin: path}, nil
	}
	if path, err := exec.LookPath("npm"); err == nil {
		return &Backend{tool: "npm", toolBin: path}, nil
	}
	return &Backend{}, nil
}

func (b *Backend) Name() string    { return "npm" }
func (b *Backend) Available() bool { return b.toolBin != "" }

func (b *Backend) ListInstalled(ctx context.Context) ([]backend.InstalledPackage, error) {
	var args []string
	if b.tool == "bun" {
		args = []string{"pm", "ls", "-g"}
	} else {
		args = []string{"list", "-g", "--depth=0"}
	}
	out, err := b.run(ctx, args, true)
	if err != nil {
		return nil, nil
	}
	if b.tool == "bun" {
		return parseBunList(out), nil
	}
	return parseNpmList(out), nil
}

func (b *Backend) Install(ctx context.Context, pkg backend.Package) error {
	name := pkg.Get("name")
	if pkg.IsPinned() {
		name = name + "@" + pkg.Get("version")
	}
	var args []string
	if b.tool == "bun" {
		args = []string{"add", "-g", name}
	} else {
		args = []string{"install", "-g", name}
	}
	_, err := b.run(ctx, args, false)
	return err
}

func (b *Backend) Uninstall(ctx context.Context, pkg backend.Package) error {
	var args []string
	if b.tool == "bun" {
		args = []string{"remove", "-g", pkg.Get("name")}
	} else {
		args = []string{"uninstall", "-g", pkg.Get("name")}
	}
	_, err := b.run(ctx, args, false)
	return err
}

func (b *Backend) Upgrade(ctx context.Context, pkg backend.Package) error {
	if pkg.IsPinned() {
		return nil
	}
	return b.Install(ctx, pkg)
}

func parseBunList(data []byte) []backend.InstalledPackage {
	var pkgs []backend.InstalledPackage
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.Contains(line, "@") {
			continue
		}
		lastAt := strings.LastIndex(line, "@")
		if lastAt <= 0 {
			continue
		}
		pkgs = append(pkgs, backend.InstalledPackage{
			Name:    line[:lastAt],
			Version: line[lastAt+1:],
			Source:  "npm",
		})
	}
	return pkgs
}

func parseNpmList(data []byte) []backend.InstalledPackage {
	var pkgs []backend.InstalledPackage
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		for _, prefix := range []string{"├── ", "└── ", "│   ", "    "} {
			line = strings.TrimPrefix(line, prefix)
		}
		line = strings.TrimSpace(line)
		if line == "" || strings.HasSuffix(line, "/") {
			continue
		}
		lastAt := strings.LastIndex(line, "@")
		if lastAt <= 0 {
			continue
		}
		pkgs = append(pkgs, backend.InstalledPackage{
			Name:    line[:lastAt],
			Version: line[lastAt+1:],
			Source:  "npm",
		})
	}
	return pkgs
}

func (b *Backend) run(ctx context.Context, args []string, capture bool) ([]byte, error) {
	if b.toolBin == "" {
		return nil, fmt.Errorf("neither bun nor npm found on PATH")
	}
	var buf bytes.Buffer
	cmd := exec.CommandContext(ctx, b.toolBin, args...)
	cmd.Env = runenv.Get()
	if capture {
		cmd.Stdout = &buf
	} else {
		cmd.Stdout = os.Stdout
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%s %v: %w", b.tool, args, err)
	}
	return buf.Bytes(), nil
}
