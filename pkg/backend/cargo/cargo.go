package cargo

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/pyrorhythm/moonshine/pkg/backend"
)

var _ backend.Backend = (*Backend)(nil)

// Backend implements backend.Backend for Cargo global installs.
type Backend struct {
	cargoPath string
}

// New returns a cargo Backend.
func New() (*Backend, error) {
	path, _ := exec.LookPath("cargo")
	return &Backend{cargoPath: path}, nil
}

func (b *Backend) Name() string    { return "cargo" }
func (b *Backend) Available() bool { return b.cargoPath != "" }

func (b *Backend) ListInstalled(ctx context.Context) ([]backend.InstalledPackage, error) {
	out, err := b.run(ctx, []string{"install", "--list"}, true)
	if err != nil {
		return nil, err
	}
	return parseCargoList(out), nil
}

func (b *Backend) Install(ctx context.Context, pkg backend.Package) error {
	args := []string{"install", pkg.Get("name")}
	if pkg.IsPinned() {
		args = append(args, "--version", pkg.Get("version"))
	}
	_, err := b.run(ctx, args, false)
	return err
}

func (b *Backend) Uninstall(ctx context.Context, pkg backend.Package) error {
	_, err := b.run(ctx, []string{"uninstall", pkg.Get("name")}, false)
	return err
}

func (b *Backend) Upgrade(ctx context.Context, pkg backend.Package) error {
	if pkg.IsPinned() {
		return nil
	}
	_, err := b.run(ctx, []string{"install", pkg.Get("name")}, false)
	return err
}

// parseCargoList parses `cargo install --list` output.
func parseCargoList(data []byte) []backend.InstalledPackage {
	var pkgs []backend.InstalledPackage
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, " ") || line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := fields[0]
		ver := strings.TrimPrefix(fields[1], "v")
		pkgs = append(pkgs, backend.InstalledPackage{
			Name:    name,
			Version: ver,
			Source:  "crates.io",
		})
	}
	return pkgs
}

func (b *Backend) run(ctx context.Context, args []string, capture bool) ([]byte, error) {
	if b.cargoPath == "" {
		return nil, fmt.Errorf("cargo not found on PATH")
	}
	var buf bytes.Buffer
	cmd := exec.CommandContext(ctx, b.cargoPath, args...)
	if capture {
		cmd.Stdout = &buf
	} else {
		cmd.Stdout = os.Stdout
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("cargo %v: %w", args, err)
	}
	return buf.Bytes(), nil
}
