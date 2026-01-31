//go:build !windows

package updater

import "syscall"

// syscallExec replaces the current process with a new one
func syscallExec(argv0 string, argv []string, envv []string) error {
	return syscall.Exec(argv0, argv, envv)
}
