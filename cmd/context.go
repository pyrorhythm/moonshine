package main

import (
	"fmt"
	"strings"

	"github.com/pyrorhythm/moonshine/internal/config"
	"github.com/pyrorhythm/moonshine/internal/config/mode"
	"github.com/pyrorhythm/moonshine/internal/lockfile"
	"github.com/pyrorhythm/moonshine/internal/registry"
	brewbackend "github.com/pyrorhythm/moonshine/pkg/backend/brew"
	"github.com/pyrorhythm/moonshine/pkg/backend/cargo"
	"github.com/pyrorhythm/moonshine/pkg/backend/goutil"
	"github.com/pyrorhythm/moonshine/pkg/backend/npm"
	"github.com/pyrorhythm/moonshine/pkg/backend/shell"
	"github.com/urfave/cli/v2"
)

// appContext holds loaded config, lockfile, and backend registry for a command.
type appContext struct {
	moonfile   *config.Moonfile
	lock       *lockfile.LockFile
	registry   *registry.Registry
	configPath string
	lockPath   string
	verbose    bool
	dryRun     bool
}

// loadContext loads the moonfile + lockfile and registers all backends.
func loadContext(c *cli.Context) (*appContext, error) {
	configPath := c.String(configFlag)
	mf, err := config.LoadMoonfile(configPath)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	if override := c.String(modeFlag); override != "" {
		mf.Moonshine.Mode = mode.OperatingMode(override)
	}

	lockPath := strings.TrimSuffix(configPath, ".yml") + ".lock"
	lf, err := lockfile.Load(lockPath)
	if err != nil {
		return nil, fmt.Errorf("loading lockfile: %w", err)
	}
	lf.Mode = string(mf.Mode)

	verbose := c.Bool(verboseFlag)
	reg := registry.NewRegistry()

	brewB, err := brewbackend.New(mf.LocalTap, verbose)
	if err == nil {
		reg.Register(brewB)
	}

	cargoB, _ := cargo.New()
	reg.Register(cargoB)

	goB, _ := goutil.New()
	reg.Register(goB)

	npmB, _ := npm.New()
	reg.Register(npmB)

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

// buildDefaultRegistry creates a registry without loading a moonfile (for snapshot).
func buildDefaultRegistry(verbose bool) (*registry.Registry, error) {
	reg := registry.NewRegistry()
	brewB, err := brewbackend.New("moonshine-local", verbose)
	if err == nil {
		reg.Register(brewB)
	}
	cargoB, _ := cargo.New()
	reg.Register(cargoB)
	goB, _ := goutil.New()
	reg.Register(goB)
	npmB, _ := npm.New()
	reg.Register(npmB)
	return reg, nil
}
