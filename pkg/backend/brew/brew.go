package brew

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// ErrBrewNotFound is returned when the brew binary cannot be located.
var ErrBrewNotFound = errors.New("brew not found: install Homebrew from https://brew.sh")

// IRunner abstracts all brew subprocess calls for testability.
type IRunner interface {
	ListInstalled(ctx context.Context) ([]ListEntry, error)
	Info(ctx context.Context, names ...string) ([]FormulaInfo, error)
	Install(ctx context.Context, formula string, args ...string) error
	Uninstall(ctx context.Context, formula string) error
	Extract(ctx context.Context, pkg, version, tap string) error
	TapCreate(ctx context.Context, name string) error
	TapExists(ctx context.Context, name string) (bool, error)
	FormulaExists(ctx context.Context, formula string) (bool, error)
	Upgrade(ctx context.Context, formula string) error
}

// Error wraps a failed brew subprocess call.
type Error struct {
	Args     []string
	ExitCode int
	Stderr   string
}

func (e *Error) Error() string {
	return fmt.Sprintf("brew %v exited %d: %s", e.Args, e.ExitCode, e.Stderr)
}

// Runner is the production IRunner implementation.
type Runner struct {
	brewPath string
	verbose  bool
	stdout   io.Writer
	stderr   io.Writer
}

// NewRunner locates the brew binary and returns a Runner.
func NewRunner(verbose bool) (*Runner, error) {
	path, err := exec.LookPath("brew")
	if err != nil {
		return nil, ErrBrewNotFound
	}
	return &Runner{
		brewPath: path,
		verbose:  verbose,
		stdout:   os.Stdout,
		stderr:   os.Stderr,
	}, nil
}

// run executes brew with args. When captureStdout is true the output is
// returned as bytes; otherwise it is streamed to the terminal.
func (r *Runner) run(ctx context.Context, args []string, captureStdout bool) ([]byte, error) {
	cmd := exec.CommandContext(ctx, r.brewPath, args...)
	var outBuf bytes.Buffer
	if captureStdout {
		cmd.Stdout = &outBuf
	} else {
		cmd.Stdout = r.stdout
	}
	cmd.Stderr = r.stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
			return nil, &Error{
				Args:     args,
				ExitCode: exitErr.ExitCode(),
				Stderr:   string(exitErr.Stderr),
			}
		}
		return nil, err
	}
	return outBuf.Bytes(), nil
}

func (r *Runner) Extract(ctx context.Context, pkg, version, tap string) error {
	if pkg == "" || version == "" || tap == "" {
		return fmt.Errorf("extract: pkg, version, and tap must be non-empty")
	}
	_, err := r.run(ctx, []string{"extract", "--version=" + version, pkg, tap}, false)
	return err
}

func (r *Runner) Install(ctx context.Context, formula string, args ...string) error {
	cmdArgs := append([]string{"install", formula}, args...)
	_, err := r.run(ctx, cmdArgs, false)
	return err
}

// Upgrade upgrades an already-installed formula to the latest available version.
func (r *Runner) Upgrade(ctx context.Context, formula string) error {
	_, err := r.run(ctx, []string{"upgrade", formula}, false)
	return err
}

func (r *Runner) ListInstalled(ctx context.Context) ([]ListEntry, error) {
	out, err := r.run(ctx, []string{"list", "--versions"}, true)
	if err != nil {
		return nil, err
	}
	return parseListOutput(out), nil
}

// Uninstall removes the named formula.
func (r *Runner) Uninstall(ctx context.Context, formula string) error {
	_, err := r.run(ctx, []string{"uninstall", formula}, false)
	return err
}

// Info returns detailed formula information for the given names.
func (r *Runner) Info(ctx context.Context, names ...string) ([]FormulaInfo, error) {
	if len(names) == 0 {
		return nil, nil
	}
	args := append([]string{"info", "--json=v2"}, names...)
	out, err := r.run(ctx, args, true)
	if err != nil {
		return nil, err
	}
	var resp infoV2Response
	if err = json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("parsing brew info JSON: %w", err)
	}
	return resp.Formulae, nil
}

// FormulaExists reports whether a formula name resolves in any tap.
func (r *Runner) FormulaExists(ctx context.Context, formula string) (bool, error) {
	_, err := r.run(ctx, []string{"info", formula}, true)
	if err != nil {
		if be, ok := err.(*Error); ok && be.ExitCode != 0 {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
