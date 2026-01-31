package files

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"sync"
	"time"
)

// Event represents a security event
type Event struct {
	Type      string                 `json:"type"`
	Severity  string                 `json:"severity"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// FileInfo stores file metadata for comparison
type FileInfo struct {
	Path     string
	Hash     string
	ModTime  time.Time
	Size     int64
	Mode     os.FileMode
	Checksum string
}

// IntegrityCollector monitors file changes
type IntegrityCollector struct {
	paths      []string
	baseline   map[string]FileInfo
	mu         sync.Mutex
	initialized bool
}

// NewIntegrityCollector creates a new file integrity collector
func NewIntegrityCollector(paths []string) *IntegrityCollector {
	return &IntegrityCollector{
		paths:    paths,
		baseline: make(map[string]FileInfo),
	}
}

// Name returns the collector name
func (c *IntegrityCollector) Name() string {
	return "file_integrity"
}

// Collect checks for file changes
func (c *IntegrityCollector) Collect(ctx context.Context) ([]Event, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var events []Event
	currentState := make(map[string]FileInfo)

	for _, path := range c.paths {
		info, err := c.getFileInfo(path)
		if err != nil {
			if os.IsNotExist(err) {
				// Check if file was deleted
				if _, existed := c.baseline[path]; existed {
					events = append(events, Event{
						Type:      "file_deleted",
						Severity:  "critical",
						Timestamp: time.Now(),
						Data: map[string]interface{}{
							"path": path,
						},
					})
				}
				continue
			}
			continue
		}

		currentState[path] = info

		// First run - establish baseline
		if !c.initialized {
			continue
		}

		// Compare with baseline
		if baseline, exists := c.baseline[path]; exists {
			changeEvents := c.detectChanges(baseline, info)
			events = append(events, changeEvents...)
		} else {
			// New file detected
			events = append(events, Event{
				Type:      "file_created",
				Severity:  "high",
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"path": path,
					"size": info.Size,
					"mode": info.Mode.String(),
				},
			})
		}
	}

	// Update baseline
	c.baseline = currentState
	c.initialized = true

	return events, nil
}

func (c *IntegrityCollector) getFileInfo(path string) (FileInfo, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return FileInfo{}, err
	}

	checksum, err := c.calculateChecksum(path)
	if err != nil {
		checksum = ""
	}

	return FileInfo{
		Path:     path,
		ModTime:  stat.ModTime(),
		Size:     stat.Size(),
		Mode:     stat.Mode(),
		Checksum: checksum,
	}, nil
}

func (c *IntegrityCollector) calculateChecksum(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func (c *IntegrityCollector) detectChanges(old, new FileInfo) []Event {
	var events []Event
	now := time.Now()

	// Check content change
	if old.Checksum != new.Checksum {
		events = append(events, Event{
			Type:      "file_modified",
			Severity:  "high",
			Timestamp: now,
			Data: map[string]interface{}{
				"path":         new.Path,
				"old_checksum": old.Checksum,
				"new_checksum": new.Checksum,
				"old_size":     old.Size,
				"new_size":     new.Size,
			},
		})
	}

	// Check permission change
	if old.Mode != new.Mode {
		events = append(events, Event{
			Type:      "file_permissions_changed",
			Severity:  "high",
			Timestamp: now,
			Data: map[string]interface{}{
				"path":     new.Path,
				"old_mode": old.Mode.String(),
				"new_mode": new.Mode.String(),
			},
		})
	}

	return events
}
