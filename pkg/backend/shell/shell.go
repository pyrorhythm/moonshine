package shell

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"github.com/pyrorhythm/moonshine/pkg/backend"
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

type Backend struct {
	config BackendConfig
}

// New creates a shell.Backend from user config.
func New(cfg BackendConfig) *Backend {
	return &Backend{config: cfg}
}

func (s *Backend) Name() string    { return s.config.Name }
func (s *Backend) Available() bool { return true }

// ListInstalled runs the list command and parses "name version" lines.
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
		pkg := backend.InstalledPackage{Name: fields[0], Source: s.config.Name}
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
	_, err := s.run(ctx, tmpl, pkg)
	return err
}

func (s *Backend) Uninstall(ctx context.Context, pkg backend.Package) error {
	_, err := s.run(ctx, s.config.Uninstall, pkg)
	return err
}

func (s *Backend) Upgrade(ctx context.Context, pkg backend.Package) error {
	tmpl := s.config.Upgrade
	if tmpl == "" {
		tmpl = s.config.InstallLatest
	}
	_, err := s.run(ctx, tmpl, pkg)
	return err
}

// templateData exposes package fields to shell templates.
type templateData struct {
	Name    string
	Version string
	Meta    map[string]string
}

func (s *Backend) run(ctx context.Context, toRun string, pkg backend.Package) ([]byte, error) {
	if toRun == "" {
		return nil, fmt.Errorf("backend %q: command template is empty", s.config.Name)
	}

	tmpl, err := template.New(s.config.Name).Option("missingkey=error").Parse(toRun)
	if err != nil {
		return nil, fmt.Errorf("backend %q: failed to parse cmd tmpl (%s): %w", s.config.Name, toRun, err)
	}

	data := templateData{
		Name:    pkg.Get("name"),
		Version: pkg.Get("version"),
		Meta:    pkg.Meta,
	}

	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute cmd tmpl: %w", err)
	}

	script := buf.String()
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "sh"
	}
	buf.Reset()
	execCmd := exec.CommandContext(ctx, shell, "-c", script)
	execCmd.Stdout = &buf
	execCmd.Stderr = os.Stderr
	if err = execCmd.Run(); err != nil {
		return nil, fmt.Errorf("backend %q: %w", s.config.Name, err)
	}
	return buf.Bytes(), nil
}

func (s *Backend) plainRun(ctx context.Context, cmd string) ([]byte, error) {
	if cmd == "" {
		return nil, fmt.Errorf("backend %q: command is empty", s.config.Name)
	}
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "sh"
	}
	var buf bytes.Buffer
	execCmd := exec.CommandContext(ctx, shell, "-c", cmd)
	execCmd.Stdout = &buf
	execCmd.Stderr = os.Stderr
	if err := execCmd.Run(); err != nil {
		return nil, fmt.Errorf("backend %q: %w", s.config.Name, err)
	}
	return buf.Bytes(), nil
}
