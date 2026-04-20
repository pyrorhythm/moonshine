package npm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"pyrorhythm.dev/moonshine/pkg/backend"
	"pyrorhythm.dev/moonshine/pkg/runenv"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripANSI(s string) string { return ansiRe.ReplaceAllString(s, "") }

// stripTreeChars removes Unicode box-drawing prefix characters that bun and npm
// use in their tree output (├──, └──, │, ─, etc.) and any surrounding whitespace.
// Box-drawing block: U+2500–U+257F.
func stripTreeChars(s string) string {
	return strings.TrimLeftFunc(s, func(r rune) bool {
		return (r >= 0x2500 && r <= 0x257F) || r == ' ' || r == '\t'
	})
}

var _ backend.Backend = (*Backend)(nil)

// Backend implements backend.Backend for global npm/bun installs.
// installTool is the active tool used for install/uninstall/upgrade.
// Both bun and npm are queried during ListInstalled (snapshot).
type Backend struct {
	installTool string // "bun" or "npm"
	bunPath     string
	npmPath     string
}

// New returns a Backend. bun is preferred for management when available.
func New() (*Backend, error) {
	b := &Backend{}
	b.bunPath, _ = exec.LookPath("bun")
	b.npmPath, _ = exec.LookPath("npm")
	switch {
	case b.bunPath != "":
		b.installTool = "bun"
	case b.npmPath != "":
		b.installTool = "npm"
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
	case "bun":
		if b.bunPath == "" {
			return nil, fmt.Errorf("bun not found on PATH")
		}
		b.installTool = "bun"
	case "npm":
		if b.npmPath == "" {
			return nil, fmt.Errorf("npm not found on PATH")
		}
		b.installTool = "npm"
	default:
		return nil, fmt.Errorf("unknown js tool %q: must be bun or npm", tool)
	}
	return b, nil
}

// Name always returns "npm" — bun is an implementation detail, not a separate backend.
func (b *Backend) Name() string    { return "npm" }
func (b *Backend) Available() bool { return b.bunPath != "" || b.npmPath != "" }

// ListInstalled queries both bun and npm and returns the merged set.
// This ensures snapshot captures packages managed by either tool.
func (b *Backend) ListInstalled(ctx context.Context) ([]backend.InstalledPackage, error) {
	seen := make(map[string]backend.InstalledPackage)

	if b.npmPath != "" {
		if pkgs, err := b.listNpm(ctx); err == nil {
			for _, p := range pkgs {
				seen[p.Name] = p
			}
		}
	}
	if b.bunPath != "" {
		if pkgs, err := b.listBun(ctx); err == nil {
			for _, p := range pkgs {
				if _, exists := seen[p.Name]; !exists {
					seen[p.Name] = p
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
	if b.installTool == "bun" {
		args = []string{"add", "-g", name}
	} else {
		args = []string{"install", "-g", name}
	}
	_, err := b.runTool(ctx, b.activePath(), args, false)
	return err
}

func (b *Backend) Uninstall(ctx context.Context, pkg backend.Package) error {
	var args []string
	if b.installTool == "bun" {
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
	if b.installTool == "bun" {
		return b.bunPath
	}
	return b.npmPath
}

// listNpm runs `npm list -g --json --depth=0` and parses the JSON response.
func (b *Backend) listNpm(ctx context.Context) ([]backend.InstalledPackage, error) {
	out, err := b.runTool(ctx, b.npmPath, []string{"list", "-g", "--json", "--depth=0"}, true)
	if err != nil {
		return nil, err
	}
	return parseNpmJSON(out), nil
}

// listBun runs `bun pm ls -g` and parses the tree output.
func (b *Backend) listBun(ctx context.Context) ([]backend.InstalledPackage, error) {
	out, err := b.runTool(ctx, b.bunPath, []string{"pm", "ls", "-g"}, true)
	if err != nil {
		return nil, err
	}
	return parseBunList(out), nil
}

// npmListJSON is the shape of `npm list -g --json --depth=0`.
type npmListJSON struct {
	Dependencies map[string]struct {
		Version string `json:"version"`
	} `json:"dependencies"`
}

func parseNpmJSON(data []byte) []backend.InstalledPackage {
	var result npmListJSON
	if err := json.Unmarshal(data, &result); err != nil {
		return nil
	}
	pkgs := make([]backend.InstalledPackage, 0, len(result.Dependencies))
	for name, dep := range result.Dependencies {
		pkgs = append(pkgs, backend.InstalledPackage{
			Name:    name,
			Version: dep.Version,
			Source:  "npm",
		})
	}
	return pkgs
}

func parseBunList(data []byte) []backend.InstalledPackage {
	var pkgs []backend.InstalledPackage
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := stripTreeChars(stripANSI(scanner.Text()))
		if line == "" {
			continue
		}
		// bun format: "name@version" or "@scope/name@version"
		lastAt := strings.LastIndex(line, "@")
		if lastAt <= 0 {
			continue
		}
		name := strings.TrimSpace(line[:lastAt])
		version := strings.TrimSpace(line[lastAt+1:])
		if name == "" || version == "" {
			continue
		}
		pkgs = append(pkgs, backend.InstalledPackage{
			Name:    name,
			Version: version,
			Source:  "npm",
		})
	}
	return pkgs
}

// Search queries npm for packages matching query using `npm search --json`.
func (b *Backend) Search(ctx context.Context, query string) ([]backend.SearchResult, error) {
	toolPath := b.npmPath
	if toolPath == "" {
		toolPath = b.bunPath
	}
	if toolPath == "" {
		return nil, nil
	}
	var args []string
	if b.npmPath != "" {
		args = []string{"search", "--json", query}
	} else {
		// bun doesn't have a search command; fall back to npm info for exact match
		out, err := b.runTool(ctx, b.bunPath, []string{"pm", "ls", "-g"}, true)
		if err != nil {
			return nil, nil
		}
		_ = out
		return nil, nil
	}
	out, err := b.runTool(ctx, b.npmPath, args, true)
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
) ([]byte, error) {
	if toolPath == "" {
		return nil, fmt.Errorf("tool not available")
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
