//go:build windows

package main

import (
	"context"
	"os/exec"
	"strings"
)

func execCommandHelper(ctx context.Context, name string, args ...string) *exec.Cmd {
	// On Windows, use cmd.exe for shell commands
	if name == "sh" && len(args) >= 2 && args[0] == "-c" {
		return exec.CommandContext(ctx, "cmd", "/c", args[1])
	}
	return exec.CommandContext(ctx, name, args...)
}

func trimOutput(output []byte) string {
	return strings.TrimSpace(string(output))
}
