//go:build windows

package updater

import (
	"fmt"
	"os"
	"os/exec"
)

// syscallExec on Windows starts a new process and exits the current one
func syscallExec(argv0 string, argv []string, envv []string) error {
	cmd := exec.Command(argv0, argv[1:]...)
	cmd.Env = envv
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start new process: %w", err)
	}
	os.Exit(0)
	return nil
}
