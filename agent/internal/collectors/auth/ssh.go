package auth

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/folt-labs/sentinel/agent/internal/types"
)

// SSHCollector monitors SSH authentication events in real-time using journalctl --follow
type SSHCollector struct {
	logFiles    []string
	useJournald bool
	stateFile   string
	eventChan   chan types.Event
	mu          sync.Mutex
	cmd         *exec.Cmd
	cancel      context.CancelFunc
}

// State persisted to disk for crash recovery
type collectorState struct {
	JournalCursor string    `json:"journal_cursor"`
	LastTimestamp time.Time `json:"last_timestamp"`
}

// Log patterns - match message CONTENT, works across all distros
var (
	sshAcceptedPattern  = regexp.MustCompile(`Accepted\s+(\w+)\s+for\s+(\w+)\s+from\s+([\d\.:a-fA-F]+)\s+port\s+(\d+)`)
	sshFailedPattern    = regexp.MustCompile(`Failed\s+(\w+)\s+for\s+(?:invalid user\s+)?(\w+)\s+from\s+([\d\.:a-fA-F]+)\s+port\s+(\d+)`)
	sshInvalidUser      = regexp.MustCompile(`Invalid user\s+(\w+)\s+from\s+([\d\.:a-fA-F]+)`)
	sshTooManyAuth      = regexp.MustCompile(`Disconnecting.*:\s+Too many authentication failures`)
	sshConnectionClosed = regexp.MustCompile(`Connection closed by.*\s+([\d\.:a-fA-F]+)\s+port\s+(\d+)\s+\[preauth\]`)
	sudoCommandPattern  = regexp.MustCompile(`sudo.*COMMAND=(.+)`)
	sudoFailPattern     = regexp.MustCompile(`sudo.*authentication failure|sudo.*3 incorrect password attempts`)
)

// NewSSHCollector creates a new real-time SSH log collector
func NewSSHCollector(logFiles []string) *SSHCollector {
	c := &SSHCollector{
		logFiles:  logFiles,
		stateFile: "/var/lib/serverguard/ssh_collector_state.json",
		eventChan: make(chan types.Event, 1000),
	}

	// Check if any log file exists, otherwise use journald
	logFileExists := false
	for _, f := range logFiles {
		if _, err := os.Stat(f); err == nil {
			logFileExists = true
			break
		}
	}

	if !logFileExists {
		if _, err := exec.LookPath("journalctl"); err == nil {
			c.useJournald = true
		}
	}

	return c
}

// Name returns the collector name
func (c *SSHCollector) Name() string {
	return "ssh"
}

// Collect is called periodically but for streaming we return buffered events
func (c *SSHCollector) Collect(ctx context.Context) ([]types.Event, error) {
	var events []types.Event

	// Drain all available events from the channel (non-blocking)
	for {
		select {
		case event := <-c.eventChan:
			events = append(events, event)
		default:
			// No more events available
			return events, nil
		}
	}
}

// StartStreaming starts the real-time log streaming (call this once on startup)
func (c *SSHCollector) StartStreaming(ctx context.Context) error {
	if c.useJournald {
		return c.streamFromJournald(ctx)
	}
	return c.streamFromFiles(ctx)
}

// streamFromJournald uses journalctl --follow for real-time streaming
func (c *SSHCollector) streamFromJournald(ctx context.Context) error {
	ctx, c.cancel = context.WithCancel(ctx)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				c.runJournalStream(ctx)
				// If we get here, journalctl died - wait and restart
				select {
				case <-ctx.Done():
					return
				case <-time.After(5 * time.Second):
					// Reconnect
				}
			}
		}
	}()

	return nil
}

func (c *SSHCollector) runJournalStream(ctx context.Context) {
	// Build journalctl command with --follow for real-time streaming
	args := []string{
		"--follow",          // Stream new entries
		"--no-pager",        // Don't page output
		"-o", "short-precise", // Precise timestamps
		"-n", "100",         // Start with last 100 entries (catch recent events)
	}

	// Load saved cursor for crash recovery
	state := c.loadState()
	if state.JournalCursor != "" {
		args = append(args, "--after-cursor="+state.JournalCursor)
	} else {
		// First run: start from 5 minutes ago
		args = append(args, "--since=5 minutes ago")
	}

	// Pattern matching - get all potentially relevant logs
	args = append(args, "--grep=sshd|sudo|Failed|Accepted|Invalid user|authentication")

	c.cmd = exec.CommandContext(ctx, "journalctl", args...)

	stdout, err := c.cmd.StdoutPipe()
	if err != nil {
		return
	}

	if err := c.cmd.Start(); err != nil {
		return
	}

	scanner := bufio.NewScanner(stdout)
	// Increase buffer size for long log lines
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		// Extract cursor from journal output if available
		if strings.HasPrefix(line, "__CURSOR=") {
			cursor := strings.TrimPrefix(line, "__CURSOR=")
			c.saveState(collectorState{JournalCursor: cursor, LastTimestamp: time.Now()})
			continue
		}

		// Parse and emit event
		if event := c.parseLine(line); event != nil {
			select {
			case c.eventChan <- *event:
			default:
				// Channel full, drop oldest
				select {
				case <-c.eventChan:
				default:
				}
				c.eventChan <- *event
			}
		}
	}

	c.cmd.Wait()
}

// streamFromFiles uses tail -F for real-time file streaming
func (c *SSHCollector) streamFromFiles(ctx context.Context) error {
	ctx, c.cancel = context.WithCancel(ctx)

	for _, logFile := range c.logFiles {
		if _, err := os.Stat(logFile); err == nil {
			go c.tailFile(ctx, logFile)
		}
	}

	return nil
}

func (c *SSHCollector) tailFile(ctx context.Context, path string) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			c.runTail(ctx, path)
			// If tail dies, wait and restart
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
				// Reconnect
			}
		}
	}
}

func (c *SSHCollector) runTail(ctx context.Context, path string) {
	// Use tail -F (capital F follows through log rotation)
	cmd := exec.CommandContext(ctx, "tail", "-F", "-n", "100", path)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return
	}

	if err := cmd.Start(); err != nil {
		return
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		if event := c.parseLine(line); event != nil {
			select {
			case c.eventChan <- *event:
			default:
				// Channel full, drop oldest
				select {
				case <-c.eventChan:
				default:
				}
				c.eventChan <- *event
			}
		}
	}

	cmd.Wait()
}

// Stop stops the streaming
func (c *SSHCollector) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
	if c.cmd != nil && c.cmd.Process != nil {
		c.cmd.Process.Kill()
	}
}

func (c *SSHCollector) loadState() collectorState {
	var state collectorState
	data, err := os.ReadFile(c.stateFile)
	if err != nil {
		return state
	}
	json.Unmarshal(data, &state)
	return state
}

func (c *SSHCollector) saveState(state collectorState) {
	c.mu.Lock()
	defer c.mu.Unlock()

	dir := filepath.Dir(c.stateFile)
	os.MkdirAll(dir, 0755)

	data, _ := json.Marshal(state)
	os.WriteFile(c.stateFile, data, 0644)
}

func (c *SSHCollector) parseLine(line string) *types.Event {
	now := time.Now()

	// Check for accepted authentication
	if matches := sshAcceptedPattern.FindStringSubmatch(line); matches != nil {
		return &types.Event{
			Type:      "ssh_login_success",
			Severity:  "info",
			Timestamp: now,
			Data: map[string]interface{}{
				"auth_method": matches[1],
				"username":    matches[2],
				"source_ip":   matches[3],
				"source_port": matches[4],
				"raw_log":     line,
			},
		}
	}

	// Check for failed authentication
	if matches := sshFailedPattern.FindStringSubmatch(line); matches != nil {
		severity := "warning"
		if matches[2] == "root" {
			severity = "high"
		}
		return &types.Event{
			Type:      "ssh_login_failed",
			Severity:  severity,
			Timestamp: now,
			Data: map[string]interface{}{
				"auth_method": matches[1],
				"username":    matches[2],
				"source_ip":   matches[3],
				"source_port": matches[4],
				"raw_log":     line,
			},
		}
	}

	// Check for invalid user attempts
	if matches := sshInvalidUser.FindStringSubmatch(line); matches != nil {
		return &types.Event{
			Type:      "ssh_invalid_user",
			Severity:  "warning",
			Timestamp: now,
			Data: map[string]interface{}{
				"username":  matches[1],
				"source_ip": matches[2],
				"raw_log":   line,
			},
		}
	}

	// Check for brute force (too many auth failures)
	if sshTooManyAuth.MatchString(line) {
		return &types.Event{
			Type:      "ssh_brute_force",
			Severity:  "high",
			Timestamp: now,
			Data: map[string]interface{}{
				"raw_log": line,
			},
		}
	}

	// Check for sudo events
	if strings.Contains(line, "sudo") {
		return c.parseSudoLine(line)
	}

	return nil
}

func (c *SSHCollector) parseSudoLine(line string) *types.Event {
	now := time.Now()

	// Check for sudo authentication failure
	if sudoFailPattern.MatchString(line) {
		return &types.Event{
			Type:      "sudo_auth_failed",
			Severity:  "high",
			Timestamp: now,
			Data: map[string]interface{}{
				"raw_log": line,
			},
		}
	}

	// Check for sudo command execution
	if matches := sudoCommandPattern.FindStringSubmatch(line); matches != nil {
		severity := "info"
		dangerousCommands := []string{"rm -rf", "chmod 777", "passwd", "useradd", "userdel", "visudo", "shutdown", "reboot", "mkfs", "dd if="}
		for _, dangerous := range dangerousCommands {
			if strings.Contains(matches[1], dangerous) {
				severity = "warning"
				break
			}
		}

		return &types.Event{
			Type:      "sudo_command",
			Severity:  severity,
			Timestamp: now,
			Data: map[string]interface{}{
				"command": matches[1],
				"raw_log": line,
			},
		}
	}

	// Generic sudo auth failure
	if strings.Contains(line, "authentication failure") {
		return &types.Event{
			Type:      "sudo_auth_failed",
			Severity:  "warning",
			Timestamp: now,
			Data: map[string]interface{}{
				"raw_log": line,
			},
		}
	}

	return nil
}

// Ensure SSHCollector implements StreamingCollector interface
var _ StreamingCollector = (*SSHCollector)(nil)

// StreamingCollector interface for collectors that support real-time streaming
type StreamingCollector interface {
	StartStreaming(ctx context.Context) error
	Stop()
}

// Helper to check if a collector supports streaming
func IsStreamingCollector(c interface{}) (StreamingCollector, bool) {
	sc, ok := c.(StreamingCollector)
	return sc, ok
}
