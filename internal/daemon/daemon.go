package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"pyrorhythm.dev/moonshine/internal/config"
	"pyrorhythm.dev/moonshine/internal/lockfile"
	"pyrorhythm.dev/moonshine/internal/reconciler"
	"pyrorhythm.dev/moonshine/internal/registry"
	"pyrorhythm.dev/moonshine/internal/state"
)

// StatusReport is broadcast to connected clients and written to the status file.
type StatusReport struct {
	LastCheck    time.Time `json:"last_check"`
	NextCheck    time.Time `json:"next_check"`
	HasDrift     bool      `json:"has_drift"`
	DriftCount   int       `json:"drift_count"`
	MoonfilePath string    `json:"moonfile_path"`
}

// Daemon is the background service that watches for drift.
type Daemon struct {
	moonfilePath string
	lockfilePath string
	registry     *registry.Registry
	interval     time.Duration
	autoApply    bool
	notify       bool
	socketPath   string
	statusPath   string
	logger       *log.Logger
}

// New creates a Daemon.
func New(
	moonfilePath, lockfilePath string,
	reg *registry.Registry,
	interval time.Duration,
	autoApply, notify bool,
) *Daemon {
	dir := Dir()
	return &Daemon{
		moonfilePath: moonfilePath,
		lockfilePath: lockfilePath,
		registry:     reg,
		interval:     interval,
		autoApply:    autoApply,
		notify:       notify,
		socketPath:   filepath.Join(dir, "daemon.sock"),
		statusPath:   filepath.Join(dir, "status.json"),
		logger:       log.New(os.Stderr, "[moonshine daemon] ", log.LstdFlags),
	}
}

// Run starts the daemon loop. Blocks until ctx is cancelled.
func (d *Daemon) Run(ctx context.Context) error {
	if err := os.MkdirAll(Dir(), 0o755); err != nil {
		return err
	}

	go d.serveSocket(ctx)

	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()

	d.logger.Printf("starting, check interval %s", d.interval)
	d.check(ctx) // immediate first check

	for {
		select {
		case <-ctx.Done():
			d.logger.Println("stopping")
			return nil
		case <-ticker.C:
			d.check(ctx)
		}
	}
}

func (d *Daemon) check(ctx context.Context) {
	mf, err := config.LoadBundle(d.moonfilePath)
	if err != nil {
		d.logger.Printf("loading moonfile: %v", err)
		return
	}

	lf, err := lockfile.Load(d.lockfilePath)
	if err != nil {
		d.logger.Printf("loading lockfile: %v", err)
		lf = lockfile.New(string(mf.Mode))
	}

	ss, err := state.Snapshot(ctx, d.registry)
	if err != nil {
		d.logger.Printf("snapshot: %v", err)
		return
	}

	plan := reconciler.Diff(mf, ss, lf)
	driftCount := len(plan.ByKind(reconciler.ActionInstall)) +
		len(plan.ByKind(reconciler.ActionUpgrade)) +
		len(plan.ByKind(reconciler.ActionUninstall))

	report := StatusReport{
		LastCheck:    time.Now().UTC(),
		NextCheck:    time.Now().Add(d.interval).UTC(),
		HasDrift:     plan.HasChanges(),
		DriftCount:   driftCount,
		MoonfilePath: d.moonfilePath,
	}
	d.writeStatus(report)

	if plan.HasChanges() {
		msg := fmt.Sprintf("%d package(s) need attention; run 'ms apply'", driftCount)
		d.logger.Println("drift detected:", msg)
		if d.notify {
			Notify("moonshine", msg)
		}
		if d.autoApply {
			d.logger.Println("auto-apply enabled, running apply…")
			opts := reconciler.ApplyOptions{
				Mode:  string(mf.Mode),
				Hooks: mf.Hooks,
			}
			if err := reconciler.Apply(ctx, plan, d.registry, lf, opts); err != nil {
				d.logger.Printf("auto-apply failed: %v", err)
				return
			}
			if err := lockfile.Save(d.lockfilePath, lf); err != nil {
				d.logger.Printf("saving lockfile: %v", err)
			}
		}
	}
}

func (d *Daemon) writeStatus(r StatusReport) {
	data, _ := json.MarshalIndent(r, "", "  ")
	_ = os.WriteFile(d.statusPath, data, 0o644)
}

// ReadStatus reads the last status report written by the daemon.
func ReadStatus(statusPath string) (*StatusReport, error) {
	data, err := os.ReadFile(statusPath)
	if err != nil {
		return nil, err
	}
	var r StatusReport
	return &r, json.Unmarshal(data, &r)
}

func (d *Daemon) serveSocket(ctx context.Context) {
	_ = os.Remove(d.socketPath)
	ln, err := net.Listen("unix", d.socketPath)
	if err != nil {
		d.logger.Printf("socket listen: %v", err)
		return
	}
	defer ln.Close()

	go func() {
		<-ctx.Done()
		ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		go d.handleConn(conn)
	}
}

func (d *Daemon) handleConn(conn net.Conn) {
	defer conn.Close()
	data, err := os.ReadFile(d.statusPath)
	if err != nil {
		return
	}
	_, _ = conn.Write(data)
}

func Dir() string {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".moonshine")
	os.MkdirAll(path, 0o644)
	return path
}

// PIDPath returns the daemon PID file path.
func PIDPath() string { return filepath.Join(Dir(), "daemon.pid") }

// SocketPath returns the daemon socket path.
func SocketPath() string { return filepath.Join(Dir(), "daemon.sock") }

// StatusPath returns the daemon status file path.
func StatusPath() string { return filepath.Join(Dir(), "status.json") }
