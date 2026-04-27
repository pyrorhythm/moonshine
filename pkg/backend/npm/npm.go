package npm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"pyrorhythm.dev/moonshine/pkg/backend"
	"pyrorhythm.dev/moonshine/pkg/runenv"
)

const (
	toolBun = "bun"
	toolNpm = "npm"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripANSI(s string) string { return ansiRe.ReplaceAllString(s, "") }

func stripTreeChars(s string) string {
	return strings.TrimLeftFunc(s, func(r rune) bool {
		return (r >= 0x2500 && r <= 0x257F) || r == ' ' || r == '\t'
	})
}

var _ backend.Backend = (*Backend)(nil)

// InstalledPackage is the npm-specific installed package record.
type InstalledPackage struct {
	Name    string
	Version string
	Manager string // "npm" or "bun"
}

func (p InstalledPackage) GetName() string    { return p.Name }
func (p InstalledPackage) GetVersion() string { return p.Version }
func (p InstalledPackage) GetSource() string  { return p.Manager }

var _ backend.InstalledPackage = InstalledPackage{}

// Backend implements backend.Backend for global npm/bun installs.
type Backend struct {
	installTool string
	bunPath     string
	npmPath     string
}

// New returns a Backend. bun is preferred for management when available.
func New() (*Backend, error) {
	b := &Backend{}
	b.bunPath, _ = exec.LookPath(toolBun)
	b.npmPath, _ = exec.LookPath(toolNpm)
	switch {
	case b.bunPath != "":
		b.installTool = toolBun
	case b.npmPath != "":
		b.installTool = toolNpm
	}
	return b, nil
}

// NewWithTool returns a Backend that uses the specified tool for management.
func NewWithTool(tool string) (*Backend, error) {
	b, err := New()
	if err != nil {
		return nil, err
	}
	switch tool {
	case toolBun:
		if b.bunPath == "" {
			return nil, errors.New("bun not found on PATH")
		}
		b.installTool = toolBun
	case toolNpm:
		if b.npmPath == "" {
			return nil, errors.New("npm not found on PATH")
		}
		b.installTool = toolNpm
	default:
		return nil, fmt.Errorf("unknown js tool %q: must be bun or npm", tool)
	}
	return b, nil
}

func (b *Backend) Name() string    { return "npm" }
func (b *Backend) Available() bool { return b.bunPath != "" || b.npmPath != "" }

func (b *Backend) ListInstalled(ctx context.Context) ([]backend.InstalledPackage, error) {
	seen := make(map[string]backend.InstalledPackage)

	if b.npmPath != "" {
		if pkgs, err := b.listNpm(ctx); err == nil {
			for _, p := range pkgs {
				seen[p.GetName()] = p
			}
		}
	}
	if b.bunPath != "" {
		if pkgs, err := b.listBun(ctx); err == nil {
			for _, p := range pkgs {
				if _, exists := seen[p.GetName()]; !exists {
					seen[p.GetName()] = p
				}
			}
		}
	}

	pkgs := make([]backend.InstalledPackage, 0, len(seen))
	for _, p := range seen {
		pkgs = append(pkgs, p)
	}
	return pkgs, nil
}

func (b *Backend) Install(ctx context.Context, pkg backend.Package) error {
	name := pkg.Get("name")
	if pkg.IsPinned() {
		name = name + "@" + pkg.Get("version")
	}
	var args []string
	if b.installTool == toolBun {
		args = []string{"add", "-g", name}
	} else {
		args = []string{"install", "-g", name}
	}
	_, err := b.runTool(ctx, b.activePath(), args, false)
	return err
}

func (b *Backend) Uninstall(ctx context.Context, pkg backend.Package) error {
	var args []string
	if b.installTool == toolBun {
		args = []string{"remove", "-g", pkg.Get("name")}
	} else {
		args = []string{"uninstall", "-g", pkg.Get("name")}
	}
	_, err := b.runTool(ctx, b.activePath(), args, false)
	return err
}

func (b *Backend) Upgrade(ctx context.Context, pkg backend.Package) error {
	if pkg.IsPinned() {
		return nil
	}
	return b.Install(ctx, pkg)
}

func (b *Backend) activePath() string {
	if b.installTool == toolBun {
		return b.bunPath
	}
	return b.npmPath
}

func (b *Backend) listNpm(ctx context.Context) ([]InstalledPackage, error) {
	out, err := b.runTool(ctx, b.npmPath, []string{"list", "-g", "--json", "--depth=0"}, true)
	if err != nil {
		return nil, err
	}
	return parseNpmJSON(out), nil
}

func (b *Backend) listBun(ctx context.Context) ([]InstalledPackage, error) {
	out, err := b.runTool(ctx, b.bunPath, []string{"pm", "ls", "-g"}, true)
	if err != nil {
		return nil, err
	}
	return parseBunList(out), nil
}

type npmListJSON struct {
	Dependencies map[string]struct {
		Version string `json:"version"`
	} `json:"dependencies"`
}

func parseNpmJSON(data []byte) []InstalledPackage {
	var result npmListJSON
	if err := json.Unmarshal(data, &result); err != nil {
		return nil
	}
	pkgs := make([]InstalledPackage, 0, len(result.Dependencies))
	for name, dep := range result.Dependencies {
		pkgs = append(pkgs, InstalledPackage{Name: name, Version: dep.Version, Manager: toolNpm})
	}
	return pkgs
}

func parseBunList(data []byte) []InstalledPackage {
	var pkgs []InstalledPackage
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := stripTreeChars(stripANSI(scanner.Text()))
		if line == "" {
			continue
		}
		lastAt := strings.LastIndex(line, "@")
		if lastAt <= 0 {
			continue
		}
		name := strings.TrimSpace(line[:lastAt])
		version := strings.TrimSpace(line[lastAt+1:])
		if name == "" || version == "" {
			continue
		}
		pkgs = append(pkgs, InstalledPackage{Name: name, Version: version, Manager: toolBun})
	}
	return pkgs
}

// Search queries npm for packages matching query.
func (b *Backend) Search(ctx context.Context, query string) ([]backend.SearchResult, error) {
	if b.npmPath == "" {
		return nil, nil
	}
	out, err := b.runTool(ctx, b.npmPath, []string{"search", "--json", query}, true)
	if err != nil {
		return nil, err
	}
	return parseNpmSearch(out), nil
}

type npmSearchResult struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

func parseNpmSearch(data []byte) []backend.SearchResult {
	var raw []npmSearchResult
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}
	results := make([]backend.SearchResult, len(raw))
	for i, r := range raw {
		results[i] = backend.SearchResult{
			Name:        r.Name,
			Version:     r.Version,
			Description: r.Description,
			Backend:     "npm",
		}
	}
	return results
}

func (b *Backend) runTool(
	ctx context.Context,
	toolPath string,
	args []string,
	capture bool,
) ([]byte, error) { //nolint:gosec
	if toolPath == "" {
		return nil, errors.New("tool not available")
	}
	var buf bytes.Buffer
	cmd := exec.CommandContext(ctx, toolPath, args...)
	cmd.Env = runenv.Get()
	if capture {
		cmd.Stdout = &buf
	} else {
		cmd.Stdout = os.Stdout
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%s %v: %w", toolPath, args, err)
	}
	return buf.Bytes(), nil
}
