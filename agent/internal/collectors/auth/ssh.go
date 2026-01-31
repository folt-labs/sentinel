package auth

import (
	"bufio"
	"context"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/folt-labs/sentinel/agent/internal/types"
)

// SSHCollector monitors SSH authentication events
type SSHCollector struct {
	logFiles     []string
	lastPosition map[string]int64
	mu           sync.Mutex
}

// Common SSH log patterns
var (
	sshAcceptedPattern = regexp.MustCompile(`Accepted\s+(\w+)\s+for\s+(\w+)\s+from\s+([\d\.]+)\s+port\s+(\d+)`)
	sshFailedPattern   = regexp.MustCompile(`Failed\s+(\w+)\s+for\s+(?:invalid user\s+)?(\w+)\s+from\s+([\d\.]+)\s+port\s+(\d+)`)
	sshInvalidUser     = regexp.MustCompile(`Invalid user\s+(\w+)\s+from\s+([\d\.]+)`)
)

// NewSSHCollector creates a new SSH log collector
func NewSSHCollector(logFiles []string) *SSHCollector {
	return &SSHCollector{
		logFiles:     logFiles,
		lastPosition: make(map[string]int64),
	}
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

	for _, logFile := range c.logFiles {
		fileEvents, err := c.processLogFile(logFile)
		if err != nil {
			// Log file might not exist on all systems
			continue
		}
		events = append(events, fileEvents...)
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
