package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/folt-labs/sentinel/agent/internal/config"
	"github.com/folt-labs/sentinel/agent/internal/daemon"
	"github.com/folt-labs/sentinel/agent/internal/updater"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	if len(os.Args) < 2 {
		runDaemon()
		return
	}

	switch os.Args[1] {
	case "version", "--version", "-v":
		printVersion()
	case "help", "--help", "-h":
		printHelp()
	case "status":
		runStatus()
	case "update":
		runUpdate()
	case "run":
		// Allow "sentinel-agent run" as explicit daemon start
		runDaemon()
	default:
		// Check if it's a flag (for backwards compatibility)
		if os.Args[1] == "--config" || os.Args[1] == "-c" {
			runDaemon()
			return
		}
		fmt.Printf("Unknown command: %s\n\n", os.Args[1])
		printHelp()
		os.Exit(1)
	}
}

func printVersion() {
	fmt.Printf("Sentinel Agent %s (built %s)\n", version, buildTime)
}

func printHelp() {
	fmt.Printf(`Sentinel Agent %s

Security monitoring agent for Linux servers.

USAGE:
    sentinel-agent [COMMAND] [OPTIONS]

COMMANDS:
    run             Start the agent daemon (default if no command given)
    status          Show agent status and version info
    update          Check for updates and apply if available
    update --check  Only check for updates, don't apply
    version         Show version information
    help            Show this help message

OPTIONS:
    --config, -c    Path to configuration file (default: /etc/sentinel/agent.yaml)

EXAMPLES:
    sentinel-agent                     # Start daemon with default config
    sentinel-agent --config /path/to/config.yaml
    sentinel-agent status              # Check if agent is running
    sentinel-agent update              # Update to latest version
    sentinel-agent update --check      # Check for updates only

CONFIGURATION:
    Default config location: /etc/sentinel/agent.yaml

    Auto-update is enabled by default. To disable, add to config:
        auto_update:
          enabled: false

For more information, visit: https://github.com/folt-labs/sentinel
`, version)
}

func runStatus() {
	fmt.Printf("Sentinel Agent Status\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("Version:     %s\n", version)
	fmt.Printf("Build Time:  %s\n", buildTime)

	// Check if service is running via systemctl
	if isServiceRunning() {
		fmt.Printf("Service:     running\n")
	} else {
		fmt.Printf("Service:     stopped\n")
	}

	// Load config and show key settings
	configPath := getConfigPath()
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Printf("Config:      error loading (%v)\n", err)
		return
	}

	fmt.Printf("Config:      %s\n", configPath)
	fmt.Printf("Server URL:  %s\n", cfg.Server.URL)
	fmt.Printf("Hostname:    %s\n", cfg.Agent.Hostname)
	fmt.Printf("Auto-Update: %v\n", cfg.AutoUpdate.Enabled)

	// Check for updates
	fmt.Printf("\nChecking for updates...\n")
	u := updater.New(updater.Config{
		Enabled:       cfg.AutoUpdate.Enabled,
		CheckInterval: cfg.AutoUpdate.CheckInterval,
		Channel:       cfg.AutoUpdate.Channel,
	}, version, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	info, err := u.CheckForUpdate(ctx)
	if err != nil {
		fmt.Printf("Update Check: failed (%v)\n", err)
		return
	}

	if info.Available {
		fmt.Printf("Update:      available (%s -> %s)\n", info.CurrentVersion, info.LatestVersion)
		fmt.Printf("             Run 'sentinel-agent update' to install\n")
	} else {
		fmt.Printf("Update:      up to date\n")
	}
}

func runUpdate() {
	checkOnly := false
	for _, arg := range os.Args[2:] {
		if arg == "--check" || arg == "-c" {
			checkOnly = true
		}
	}

	configPath := getConfigPath()
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	u := updater.New(updater.Config{
		Enabled:       true, // Always enabled for manual update command
		CheckInterval: cfg.AutoUpdate.CheckInterval,
		Channel:       cfg.AutoUpdate.Channel,
	}, version, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	fmt.Println("Checking for updates...")

	info, err := u.CheckForUpdate(ctx)
	if err != nil {
		log.Fatalf("Update check failed: %v", err)
	}

	if !info.Available {
		fmt.Printf("Already running the latest version (%s)\n", info.CurrentVersion)
		return
	}

	fmt.Printf("Update available: %s -> %s\n", info.CurrentVersion, info.LatestVersion)

	if info.ReleaseNotes != "" {
		fmt.Printf("\nRelease Notes:\n%s\n", info.ReleaseNotes)
	}

	if checkOnly {
		fmt.Println("\nRun 'sentinel-agent update' (without --check) to install")
		return
	}

	fmt.Println("\nDownloading and installing update...")

	if err := u.Update(ctx, info); err != nil {
		log.Fatalf("Update failed: %v", err)
	}

	fmt.Println("Update installed successfully!")
	fmt.Println("")
	fmt.Println("To apply the update, restart the service:")
	fmt.Println("  sudo systemctl restart sentinel-agent")
}

func runDaemon() {
	configPath := getConfigPath()

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create and start daemon
	d, err := daemon.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create daemon: %v", err)
	}

	// Channel for update restart signal
	restartChan := make(chan struct{})

	// Start auto-updater in background
	u := updater.New(updater.Config{
		Enabled:       cfg.AutoUpdate.Enabled,
		CheckInterval: cfg.AutoUpdate.CheckInterval,
		Channel:       cfg.AutoUpdate.Channel,
	}, version, restartChan)
	go u.Run(ctx)

	// Start daemon in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- d.Run(ctx)
	}()

	log.Printf("Sentinel Agent %s started", version)

	// Wait for shutdown signal, error, or update restart
	select {
	case sig := <-sigChan:
		log.Printf("Received signal %v, shutting down...", sig)
		cancel()
	case <-restartChan:
		log.Println("Restarting for update...")
		cancel()
	case err := <-errChan:
		if err != nil {
			log.Fatalf("Daemon error: %v", err)
		}
	}

	// Wait for graceful shutdown
	if err := d.Shutdown(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	log.Println("Sentinel Agent stopped")
}

func getConfigPath() string {
	// Check command line args for --config or -c
	for i, arg := range os.Args {
		if (arg == "--config" || arg == "-c") && i+1 < len(os.Args) {
			return os.Args[i+1]
		}
	}
	return "/etc/sentinel/agent.yaml"
}

func isServiceRunning() bool {
	// Try to check via systemctl
	cmd := fmt.Sprintf("systemctl is-active sentinel-agent 2>/dev/null")
	output, err := runCommand(cmd)
	if err != nil {
		return false
	}
	// Trim whitespace/newlines from output
	return strings.TrimSpace(output) == "active"
}

func runCommand(cmd string) (string, error) {
	out, err := execCommand("sh", "-c", cmd)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func execCommand(name string, args ...string) ([]byte, error) {
	return execCommandContext(context.Background(), name, args...)
}

func execCommandContext(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := execCommandHelper(ctx, name, args...)
	return cmd.Output()
}
