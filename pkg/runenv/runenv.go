// Package runenv provides the user's login shell environment for subprocesses.
// When moonshine runs as a daemon its inherited environment may be stripped of
// PATH additions, PKG_CONFIG_PATH, LIBRARY_PATH, etc. that compilers need.
// Backends that invoke compilers (cargo, go install) should use Get() so the
// spawned process sees the same environment the user's shell would.
package runenv

import (
	"os"
	"os/exec"
	"strings"
	"sync"
)

var (
	once     sync.Once
	loginEnv []string
)

// Get returns the user's login-shell environment, cached after the first call.
// Falls back to os.Environ() if the login shell cannot be queried.
func Get() []string {
	once.Do(func() { loginEnv = capture() })
	return loginEnv
}

func capture() []string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	out, err := exec.Command(shell, "-l", "-c", "env").Output()
	if err != nil {
		return os.Environ()
	}
	var env []string
	for line := range strings.SplitSeq(string(out), "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.ContainsRune(line, '=') {
			env = append(env, line)
		}
	}
	if len(env) == 0 {
		return os.Environ()
	}
	return env
}
