package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pyrorhythm/moonshine/internal/config"
	"github.com/pyrorhythm/moonshine/internal/config/mode"
	"github.com/pyrorhythm/moonshine/internal/ui"
	"github.com/urfave/cli/v2"
)

func initCommand() *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: "interactively create moonconfig.yml and moonpackages.yml",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Value:   defaultConfigPath(),
				Usage:   "output path for moonconfig.yml",
			},
		},
		Action: func(c *cli.Context) error {
			output := c.String("output")

			if _, err := os.Stat(output); err == nil {
				return fmt.Errorf(
					"%s already exists; delete it or use --output for a different path",
					output,
				)
			}
			if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
				return fmt.Errorf("creating config directory: %w", err)
			}

			ui.Banner()
			fmt.Println("  Let's create your moonconfig.")
			fmt.Println()

			opMode := prompt("Operating mode (standalone/companion)", "standalone")
			if !mode.OperatingMode(opMode).Valid() {
				ui.Warn("unknown mode, defaulting to standalone")
				opMode = string(mode.Standalone)
			}

			localTap := prompt("Local brew tap name", "moonshine-local")
			enableDaemon := promptBool("Enable background daemon?", false)

			mf := config.NewMoonfile(opMode)
			mf.LocalTap = localTap
			mf.Daemon.Enabled = enableDaemon

			if err := config.SaveMoonfile(output, mf); err != nil {
				return fmt.Errorf("writing moonconfig: %w", err)
			}
			ui.Success(fmt.Sprintf("moonconfig.yml written to %s", output))
			ui.Info("next: add packages with 'ms add brew#<package>' or edit moonpackages.yml directly")
			return nil
		},
	}
}

func prompt(label, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("  %s [%s]: ", label, defaultVal)
	} else {
		fmt.Printf("  %s: ", label)
	}
	r := bufio.NewReader(os.Stdin)
	line, _ := r.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultVal
	}
	return line
}

func promptBool(label string, defaultVal bool) bool {
	def := "n"
	if defaultVal {
		def = "y"
	}
	answer := prompt(label+" (y/n)", def)
	return strings.ToLower(answer) == "y"
}
