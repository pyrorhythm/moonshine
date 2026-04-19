package hooks

import (
	"context"
	"os"
	"os/exec"
)

// Hooks defines shell scripts to run at lifecycle points.
// Empty strings are silently skipped.
type Hooks struct {
	PreApply        string `yaml:"pre_apply"`
	PostApply       string `yaml:"post_apply"`
	PreInstall      string `yaml:"pre_install"`
	PostInstall     string `yaml:"post_install"`
	PreRemove       string `yaml:"pre_remove"`
	PostRemove      string `yaml:"post_remove"`
	PreBrewExtract  string `yaml:"pre_brew_extract"`
	PostBrewExtract string `yaml:"post_brew_extract"`
}

// Env carries context variables injected into every hook execution.
type Env struct {
	Package string
	Version string
	Backend string
	Action  string
	Mode    string
}

// Run executes script via $SHELL -c, streaming stdout/stderr to the terminal.
// A non-zero exit code returns an error, aborting the calling operation.
// If script is empty, Run is a no-op.
func Run(ctx context.Context, script string, env Env) error {
	if script == "" {
		return nil
	}
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "sh"
	}
	cmd := exec.CommandContext(ctx, shell, "-c", script)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(),
		"MOONSHINE_PACKAGE="+env.Package,
		"MOONSHINE_VERSION="+env.Version,
		"MOONSHINE_BACKEND="+env.Backend,
		"MOONSHINE_ACTION="+env.Action,
		"MOONSHINE_MODE="+env.Mode,
	)
	return cmd.Run()
}
