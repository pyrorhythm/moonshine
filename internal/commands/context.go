package commands

import (
	"fmt"
	"strings"

	"github.com/urfave/cli/v2"
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
