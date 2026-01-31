package auth

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/folt-labs/sentinel/agent/internal/types"
)

// SSHCollector monitors SSH authentication events
type SSHCollector struct {
	logFiles      []string
	lastPosition  map[string]int64
	useJournald   bool
	lastJournalTS time.Time
	mu            sync.Mutex
}

// Common SSH log patterns
var (
	sshAcceptedPattern = regexp.MustCompile(`Accepted\s+(\w+)\s+for\s+(\w+)\s+from\s+([\d\.]+)\s+port\s+(\d+)`)
	sshFailedPattern   = regexp.MustCompile(`Failed\s+(\w+)\s+for\s+(?:invalid user\s+)?(\w+)\s+from\s+([\d\.]+)\s+port\s+(\d+)`)
	sshInvalidUser     = regexp.MustCompile(`Invalid user\s+(\w+)\s+from\s+([\d\.]+)`)
)

// NewSSHCollector creates a new SSH log collector
func NewSSHCollector(logFiles []string) *SSHCollector {
	c := &SSHCollector{
		logFiles:      logFiles,
		lastPosition:  make(map[string]int64),
		lastJournalTS: time.Now(),
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
		// Check if journalctl is available
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

// Collect gathers SSH authentication events
func (c *SSHCollector) Collect(ctx context.Context) ([]types.Event, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var events []types.Event

	if c.useJournald {
		journalEvents, err := c.collectFromJournald()
		if err == nil {
			events = append(events, journalEvents...)
		}
	} else {
		for _, logFile := range c.logFiles {
			fileEvents, err := c.processLogFile(logFile)
			if err != nil {
				// Log file might not exist on all systems
				continue
			}
			events = append(events, fileEvents...)
		}
	}

	return events, nil
}

// collectFromJournald reads SSH events from systemd journal
func (c *SSHCollector) collectFromJournald() ([]types.Event, error) {
	// Get logs since last check
	since := c.lastJournalTS.Format("2006-01-02 15:04:05")
	c.lastJournalTS = time.Now()

	// Query journald for sshd and sudo logs
	cmd := exec.Command("journalctl", "-u", "ssh", "-u", "sshd", "-t", "sudo",
		"--since", since, "--no-pager", "-q")
	output, err := cmd.Output()
	if err != nil {
		// Try alternative: query by syslog identifier
		cmd = exec.Command("journalctl", "-t", "sshd", "-t", "sudo",
			"--since", since, "--no-pager", "-q")
		output, err = cmd.Output()
		if err != nil {
			return nil, err
		}
	}

	var events []types.Event
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if event := c.parseLine(line); event != nil {
			events = append(events, *event)
		}
	}

	return events, nil
}

func (c *SSHCollector) processLogFile(path string) ([]types.Event, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Seek to last known position
	lastPos := c.lastPosition[path]
	if lastPos > 0 {
		info, err := file.Stat()
		if err != nil {
			return nil, err
		}
		// Reset if file was truncated (log rotation)
		if info.Size() < lastPos {
			lastPos = 0
		}
		file.Seek(lastPos, 0)
	}

	var events []types.Event
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		event := c.parseLine(line)
		if event != nil {
			events = append(events, *event)
		}
	}

	// Save current position
	pos, _ := file.Seek(0, 1)
	c.lastPosition[path] = pos

	return events, scanner.Err()
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
		// Check for root login attempts
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

	// Check for sudo events
	if strings.Contains(line, "sudo:") {
		return c.parseSudoLine(line)
	}

	return nil
}

func (c *SSHCollector) parseSudoLine(line string) *types.Event {
	now := time.Now()

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

	if strings.Contains(line, "COMMAND=") {
		return &types.Event{
			Type:      "sudo_command",
			Severity:  "info",
			Timestamp: now,
			Data: map[string]interface{}{
				"raw_log": line,
			},
		}
	}

	return nil
}
