package brew

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// ErrBrewNotFound is returned when the brew binary cannot be located.
var ErrBrewNotFound = errors.New("brew not found: install Homebrew from https://brew.sh")

// FormulaInfo holds the fields returned by brew info --json.
type FormulaInfo struct {
	Name      string `json:"name"`
	FullName  string `json:"full_name"`
	Tap       string `json:"tap"`
	Desc      string `json:"desc"`
	Installed []struct {
		Version string `json:"version"`
	} `json:"installed"`
}

// IRunner abstracts brew CLI subprocess calls for testability.
type IRunner interface {
	Leaves(ctx context.Context) ([]string, error)
	InfoJSON(ctx context.Context, names []string) ([]FormulaInfo, error)
	Install(ctx context.Context, formula string, args ...string) error
	Uninstall(ctx context.Context, formula string) error
	Extract(ctx context.Context, pkg, version, tap string) error
	TapAdd(ctx context.Context, name string) error
	TapCreate(ctx context.Context, name string) error
	TapExists(ctx context.Context, name string) (bool, error)
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

func (r *Runner) run(
	ctx context.Context,
	args []string,
	captureStdout bool,
) ([]byte, error) { //nolint:gosec
	cmd := exec.CommandContext(ctx, r.brewPath, args...)
	var outBuf bytes.Buffer
	if captureStdout {
		cmd.Stdout = &outBuf
	} else {
		cmd.Stdout = r.stdout
	}
	cmd.Stderr = r.stderr
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
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

func (r *Runner) Leaves(ctx context.Context) ([]string, error) {
	out, err := r.run(ctx, []string{"leaves"}, true)
	if err != nil {
		return nil, err
	}
	return parseLeaves(out), nil
}

func (r *Runner) InfoJSON(ctx context.Context, names []string) ([]FormulaInfo, error) {
	if len(names) == 0 {
		return nil, nil
	}
	args := append([]string{"info", "--json"}, names...)
	out, err := r.run(ctx, args, true)
	if err != nil {
		return nil, err
	}
	var entries []FormulaInfo
	if err := json.Unmarshal(out, &entries); err != nil {
		return nil, fmt.Errorf("parsing brew info output: %w", err)
	}
	return entries, nil
}

func (r *Runner) Install(ctx context.Context, formula string, args ...string) error {
	cmdArgs := append([]string{"install", formula}, args...)
	_, err := r.run(ctx, cmdArgs, false)
	return err
}

func (r *Runner) Upgrade(ctx context.Context, formula string) error {
	_, err := r.run(ctx, []string{"upgrade", formula}, false)
	return err
}

func (r *Runner) Uninstall(ctx context.Context, formula string) error {
	_, err := r.run(ctx, []string{"uninstall", formula}, false)
	return err
}

func (r *Runner) Extract(ctx context.Context, pkg, version, tap string) error {
	if pkg == "" || version == "" || tap == "" {
		return errors.New("extract: pkg, version, and tap must be non-empty")
	}
	_, err := r.run(ctx, []string{"extract", "--version=" + version, pkg, tap}, false)
	return err
}

func (r *Runner) TapAdd(ctx context.Context, name string) error {
	_, err := r.run(ctx, []string{"tap", name}, false)
	return err
}

func (r *Runner) TapExists(ctx context.Context, name string) (bool, error) {
	out, err := r.run(ctx, []string{"tap"}, true)
	if err != nil {
		return false, err
	}
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == name {
			return true, nil
		}
	}
	return false, scanner.Err()
}

func (r *Runner) TapCreate(ctx context.Context, name string) error {
	exists, err := r.TapExists(ctx, name)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	_, err = r.run(ctx, []string{"tap-new", name}, false)
	return err
}

func parseLeaves(data []byte) []string {
	var names []string
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		if name := strings.TrimSpace(scanner.Text()); name != "" {
			names = append(names, name)
		}
	}
	return names
}
