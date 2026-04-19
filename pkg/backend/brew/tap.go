package brew

import (
	"bufio"
	"bytes"
	"context"
	"strings"
)

// TapExists reports whether the given tap name is already registered.
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

// TapCreate registers a new local tap, creating the directory structure.
// For a user-scoped local tap the name should be "user/tap-name".
// If it already exists, TapCreate is a no-op.
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
