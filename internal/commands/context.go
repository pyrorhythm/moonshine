package commands

import (
	"fmt"
	"path/filepath"
	"slices"

	"github.com/urfave/cli/v3"
	"pyrorhythm.dev/moonshine/internal/config"
	"pyrorhythm.dev/moonshine/internal/config/mode"
	"pyrorhythm.dev/moonshine/internal/lockfile"
	"pyrorhythm.dev/moonshine/internal/registry"
	brewbackend "pyrorhythm.dev/moonshine/pkg/backend/brew"
	"pyrorhythm.dev/moonshine/pkg/backend/cargo"
	"pyrorhythm.dev/moonshine/pkg/backend/goutil"
	"pyrorhythm.dev/moonshine/pkg/backend/npm"
	"pyrorhythm.dev/moonshine/pkg/backend/shell"
)

type appContext struct {
	moonfile   *config.Moonfile
	lock       *lockfile.LockFile
	registry   *registry.Registry
	configPath string
	lockPath   string
	verbose    bool
	dryRun     bool
}

func loadContext(c *cli.Command) (*appContext, error) {
	configPath := c.String(configFlag)
	mf, err := config.LoadBundle(configPath)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	if override := c.String(modeFlag); override != "" {
		mf.Moonshine.Mode = mode.OperatingMode(override)
	}

	lockPath := filepath.Join(filepath.Dir(configPath), "moonshine.lock")
	lf, err := lockfile.Load(lockPath)
	if err != nil {
		return nil, fmt.Errorf("loading lockfile: %w", err)
	}
	lf.Mode = string(mf.Mode)

	verbose := c.Bool(verboseFlag)
	reg := buildRegistry(mf, verbose)

	return &appContext{
		moonfile:   mf,
		lock:       lf,
		registry:   reg,
		configPath: configPath,
		lockPath:   lockPath,
		verbose:    verbose,
		dryRun:     c.Bool(dryRunFlag),
	}, nil
}

func buildRegistry(mf *config.Moonfile, verbose bool) *registry.Registry {
	reg := registry.NewRegistry()

	enabled := func(name string) bool {
		return slices.Contains(mf.Backends, name)
	}

	if enabled("brew") {
		if b, err := brewbackend.New(mf.LocalTap, verbose); err == nil {
			reg.Register(b)
		}
	}
	if enabled("cargo") {
		if b, _ := cargo.New(); b != nil {
			reg.Register(b)
		}
	}
	if enabled("go") {
		if b, _ := goutil.New(); b != nil {
			reg.Register(b)
		}
	}
	if enabled("npm") {
		if b, _ := npm.New(); b != nil {
			reg.Register(b)
		}
	}

	for _, cfg := range mf.Shell {
		reg.Register(shell.New(shell.BackendConfig{
			Name:          cfg.Name,
			List:          cfg.List,
			Install:       cfg.Install,
			InstallLatest: cfg.InstallLatest,
			Uninstall:     cfg.Uninstall,
			Upgrade:       cfg.Upgrade,
		}))
	}

	return reg
}

func buildDefaultRegistry(verbose bool) *registry.Registry {
	reg := registry.NewRegistry()
	if b, err := brewbackend.New("moonshine-local", verbose); err == nil {
		reg.Register(b)
	}
	if b, _ := cargo.New(); b != nil {
		reg.Register(b)
	}
	if b, _ := goutil.New(); b != nil {
		reg.Register(b)
	}
	if b, _ := npm.New(); b != nil {
		reg.Register(b)
	}
	return reg
}
