package shell

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"pyrorhythm.dev/moonshine/pkg/backend"
)

var _ backend.Backend = (*Backend)(nil)

// BackendConfig is the user-supplied configuration for a custom backend.
type BackendConfig struct {
	Name          string `yaml:"name"`
	List          string `yaml:"list"`
	Install       string `yaml:"install"`
	InstallLatest string `yaml:"install_latest"`
	Uninstall     string `yaml:"uninstall"`
	Upgrade       string `yaml:"upgrade"`
}

// InstalledPackage is the shell-backend-specific installed package record.
type InstalledPackage struct {
	Name    string
	Version string
	Backend string
}

func (p InstalledPackage) GetName() string    { return p.Name }
func (p InstalledPackage) GetVersion() string { return p.Version }
func (p InstalledPackage) GetSource() string  { return p.Backend }

var _ backend.InstalledPackage = InstalledPackage{}

// Backend implements backend.Backend via user-defined shell commands.
type Backend struct {
	config BackendConfig
}

// New creates a shell.Backend from user config.
func New(cfg BackendConfig) *Backend {
	return &Backend{config: cfg}
}

func (s *Backend) Name() string    { return s.config.Name }
func (s *Backend) Available() bool { return true }

func (s *Backend) ListInstalled(ctx context.Context) ([]backend.InstalledPackage, error) {
	out, err := s.plainRun(ctx, s.config.List)
	if err != nil {
		return nil, err
	}
	var pkgs []backend.InstalledPackage
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		pkg := InstalledPackage{Name: fields[0], Backend: s.config.Name}
		if len(fields) >= 2 {
			pkg.Version = fields[1]
		}
		pkgs = append(pkgs, pkg)
	}
	return pkgs, scanner.Err()
}

func (s *Backend) Install(ctx context.Context, pkg backend.Package) error {
	tmpl := s.config.Install
	if !pkg.IsPinned() && s.config.InstallLatest != "" {
		tmpl = s.config.InstallLatest
	}
	return s.run(ctx, tmpl, pkg)
}

func (s *Backend) Uninstall(ctx context.Context, pkg backend.Package) error {
	return s.run(ctx, s.config.Uninstall, pkg)
}

func (s *Backend) Upgrade(ctx context.Context, pkg backend.Package) error {
	tmpl := s.config.Upgrade
	if tmpl == "" {
		tmpl = s.config.InstallLatest
	}
	return s.run(ctx, tmpl, pkg)
}

type templateData struct {
	Name    string
	Version string
	Meta    map[string]string
}

func (s *Backend) run(ctx context.Context, toRun string, pkg backend.Package) error { //nolint:gosec
	if toRun == "" {
		return fmt.Errorf("backend %q: command template is empty", s.config.Name)
	}
	tmpl, err := template.New(s.config.Name).Option("missingkey=error").Parse(toRun)
	if err != nil {
		return fmt.Errorf(
			"backend %q: failed to parse cmd tmpl (%s): %w",
			s.config.Name,
			toRun,
			err,
		)
	}
	data := templateData{Name: pkg.Get("name"), Version: pkg.Get("version"), Meta: pkg.Meta}
	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to execute cmd tmpl: %w", err)
	}
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "sh"
	}
	execCmd := exec.CommandContext(ctx, shell, "-c", buf.String()) //nolint:gosec
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr
	if err = execCmd.Run(); err != nil {
		return fmt.Errorf("backend %q: %w", s.config.Name, err)
	}
	return nil
}

func (s *Backend) plainRun(ctx context.Context, cmd string) ([]byte, error) { //nolint:gosec
	if cmd == "" {
		return nil, errors.New("backend " + s.config.Name + ": command is empty")
	}
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "sh"
	}
	var buf bytes.Buffer
	execCmd := exec.CommandContext(ctx, shell, "-c", cmd) //nolint:gosec
	execCmd.Stdout = &buf
	execCmd.Stderr = os.Stderr
	if err := execCmd.Run(); err != nil {
		return nil, fmt.Errorf("backend %q: %w", s.config.Name, err)
	}
	return buf.Bytes(), nil
}
