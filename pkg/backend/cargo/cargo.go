package cargo

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"pyrorhythm.dev/moonshine/pkg/backend"
	"pyrorhythm.dev/moonshine/pkg/runenv"
)

var _ backend.Backend = (*Backend)(nil)

// InstalledPackage is the cargo-specific installed package record.
type InstalledPackage struct {
	Name    string
	Version string
}

func (p InstalledPackage) GetName() string    { return p.Name }
func (p InstalledPackage) GetVersion() string { return p.Version }
func (p InstalledPackage) GetSource() string  { return "crates.io" }

var _ backend.InstalledPackage = InstalledPackage{}

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

// Search runs `cargo search <query>` and returns up to 10 results.
func (b *Backend) Search(ctx context.Context, query string) ([]backend.SearchResult, error) {
	out, err := b.run(ctx, []string{"search", "--limit", "10", query}, true)
	if err != nil {
		return nil, err
	}
	return parseCargoSearch(out), nil
}

func parseCargoSearch(data []byte) []backend.SearchResult {
	var results []backend.SearchResult
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "... ") || line == "" {
			continue
		}
		name, rest, ok := strings.Cut(line, " = ")
		if !ok {
			continue
		}
		name = strings.TrimSpace(name)
		var version, desc string
		if len(rest) > 2 && rest[0] == '"' {
			if end := strings.IndexByte(rest[1:], '"'); end >= 0 {
				version = rest[1 : end+1]
				rest = rest[end+2:]
			}
		}
		if _, d, found := strings.Cut(rest, "# "); found {
			desc = strings.TrimSpace(d)
		}
		results = append(results, backend.SearchResult{
			Name:        name,
			Version:     version,
			Description: desc,
			Backend:     "cargo",
		})
	}
	return results
}

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
		pkgs = append(pkgs, InstalledPackage{
			Name:    fields[0],
			Version: strings.TrimPrefix(fields[1], "v"),
		})
	}
	return pkgs
}

func (b *Backend) run(
	ctx context.Context,
	args []string,
	capture bool,
) ([]byte, error) { //nolint:gosec
	if b.cargoPath == "" {
		return nil, errors.New("cargo not found on PATH")
	}
	var buf bytes.Buffer
	cmd := exec.CommandContext(ctx, b.cargoPath, args...)
	cmd.Env = runenv.Get()
	if capture {
		cmd.Stdout = &buf
	} else if out, ok := backend.OutputFrom(ctx); ok {
		cmd.Stdout = out
		cmd.Stderr = out
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("cargo %v: %w", args, err)
	}
	return buf.Bytes(), nil
}
