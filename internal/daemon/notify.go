package daemon

import (
	"os/exec"
	"runtime"
)

// Notify sends a desktop notification. Silently no-ops if the platform
// doesn't support it or the required tool isn't available.
func Notify(title, body string) {
	switch runtime.GOOS {
	case "darwin":
		script := `display notification "` + body + `" with title "` + title + `"`
		_ = exec.Command("osascript", "-e", script).Run()
	case "linux":
		_ = exec.Command("notify-send", title, body).Run()
	}
}
