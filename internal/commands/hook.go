package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v3"
)

const shellBash = "bash"

func hookCommand() *cli.Command {
	return &cli.Command{
		Name:      "hook",
		Usage:     "print shell integration snippet",
		ArgsUsage: "[shell]",
		Description: `Prints a shell snippet that adds moonshine to your PATH.
Pipe or eval it into your shell config file.

Examples:
  bash/zsh:  eval "$(ms hook)"
  fish:      ms hook fish | source
  nushell:   ms hook nu  (paste output into config.nu)
  xonsh:     execx($(ms hook xonsh))
  ion:       eval $(ms hook ion)`,
		Action: func(ctx context.Context, c *cli.Command) error {
			shellName := c.Args().First()
			if shellName == "" {
				shellName = detectShell()
			}
			snippet, err := shellSnippet(shellName)
			if err != nil {
				return err
			}
			fmt.Print(snippet)
			return nil
		},
	}
}

// detectShell infers the current shell from $SHELL or $0.
func detectShell() string {
	for _, v := range []string{os.Getenv("SHELL"), os.Getenv("0")} {
		base := strings.ToLower(filepath.Base(v))
		switch {
		case strings.Contains(base, "fish"):
			return "fish"
		case strings.Contains(base, "zsh"):
			return "zsh"
		case strings.Contains(base, "nu"):
			return "nushell"
		case strings.Contains(base, "xonsh"):
			return "xonsh"
		case strings.Contains(base, "ion"):
			return "ion"
		case strings.Contains(base, "bash"):
			return shellBash
		}
	}
	return "bash"
}

func shellSnippet(shell string) (string, error) {
	switch strings.ToLower(shell) {
	case "bash", "sh":
		return `# moonshine shell integration
export MOONSHINE_HOME="${MOONSHINE_HOME:-$HOME/.moonshine}"
export PATH="$MOONSHINE_HOME/bin:$PATH"
`, nil

	case "zsh":
		return `# moonshine shell integration
export MOONSHINE_HOME="${MOONSHINE_HOME:-$HOME/.moonshine}"
export PATH="$MOONSHINE_HOME/bin:$PATH"
`, nil

	case "fish":
		return `# moonshine shell integration
set -gx MOONSHINE_HOME (test -n "$MOONSHINE_HOME" && echo $MOONSHINE_HOME || echo $HOME/.moonshine)
fish_add_path $MOONSHINE_HOME/bin
`, nil

	case "nu", "nushell":
		return `# moonshine shell integration
# Add to config.nu:
$env.MOONSHINE_HOME = ($env | get -i MOONSHINE_HOME | default ($env.HOME | path join ".moonshine"))
$env.PATH = ($env.PATH | prepend ($env.MOONSHINE_HOME | path join "bin"))
`, nil

	case "xonsh":
		return `# moonshine shell integration
import os as _ms_os
$MOONSHINE_HOME = _ms_os.environ.get('MOONSHINE_HOME', _ms_os.path.join($HOME, '.moonshine'))
$PATH.insert(0, $MOONSHINE_HOME + '/bin')
del _ms_os
`, nil

	case "ion", "ionsh":
		return `# moonshine shell integration
export MOONSHINE_HOME = "~/.moonshine"
export PATH = "$MOONSHINE_HOME/bin:$PATH"
`, nil

	default:
		return "", fmt.Errorf("unknown shell %q — supported: bash, zsh, fish, nushell, xonsh, ion", shell)
	}
}
