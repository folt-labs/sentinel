//go:build !windows

package main

import (
	"context"
	"os/exec"
	"strings"
)

func execCommandHelper(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}

func trimOutput(output []byte) string {
	return strings.TrimSpace(string(output))
}
