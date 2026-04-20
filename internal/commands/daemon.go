package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/pyrorhythm/moonshine/internal/daemon"
	"github.com/pyrorhythm/moonshine/internal/ui"
	"github.com/urfave/cli/v2"
)

func daemonCommand() *cli.Command {
	return &cli.Command{
		Name:  "daemon",
		Usage: "manage the background drift-watching daemon",
		Subcommands: []*cli.Command{
			{
				Name:  "start",
				Usage: "start the background daemon",
				Action: func(c *cli.Context) error {
					ac, err := loadContext(c)
					if err != nil {
						return err
					}
					if !ac.moonfile.Daemon.Enabled {
						ui.Warn("daemon.enabled is false in moonconfig; starting anyway")
					}
					interval := ac.moonfile.Daemon.CheckInterval
					if interval == 0 {
						interval = 6 * time.Hour
					}
					d := daemon.New(
						ac.configPath,
						ac.lockPath,
						ac.registry,
						interval,
						ac.moonfile.Daemon.AutoApply,
						ac.moonfile.Daemon.Notify,
					)
					if err = writePID(daemon.PIDPath()); err != nil {
						return err
					}
					ctx, stop := signal.NotifyContext(
						context.Background(),
						os.Interrupt,
						syscall.SIGTERM,
					)
					defer stop()
					ui.Success("daemon started (pid " + strconv.Itoa(os.Getpid()) + ")")
					return d.Run(ctx)
				},
			},
			{
				Name:  "stop",
				Usage: "stop the running daemon",
				Action: func(_ *cli.Context) error {
					pidFile := daemon.PIDPath()
					data, err := os.ReadFile(pidFile)
					if err != nil {
						return fmt.Errorf("daemon not running (no pid file)")
					}
					pid, err := strconv.Atoi(string(data))
					if err != nil {
						return fmt.Errorf("invalid pid file: %w", err)
					}
					proc, err := os.FindProcess(pid)
					if err != nil {
						return fmt.Errorf("process %d not found: %w", pid, err)
					}
					if err := proc.Signal(syscall.SIGTERM); err != nil {
						return fmt.Errorf("sending signal to pid %d: %w", pid, err)
					}
					os.Remove(pidFile)
					ui.Success(fmt.Sprintf("daemon (pid %d) stopped", pid))
					return nil
				},
			},
			{
				Name:  "status",
				Usage: "show daemon health and last check time",
				Action: func(_ *cli.Context) error {
					report, err := daemon.ReadStatus(daemon.StatusPath())
					if err != nil {
						ui.Warn("daemon is not running or has never checked")
						return nil
					}
					fmt.Printf("last check:  %s\n", report.LastCheck.Local().Format(time.RFC822))
					fmt.Printf("next check:  %s\n", report.NextCheck.Local().Format(time.RFC822))
					if report.HasDrift {
						ui.Warn(fmt.Sprintf(
							"drift:       %d package(s) need attention",
							report.DriftCount,
						))
					} else {
						ui.Success("drift:       none")
					}
					return nil
				},
			},
		},
	}
}

func writePID(path string) error {
	return os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())), 0o644)
}
