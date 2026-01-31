package transport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/folt-labs/sentinel/agent/internal/config"
)

// Client handles HTTP communication with the API server
type Client struct {
	cfg        *config.Config
	httpClient *http.Client
	queueDir   string
	mu         sync.Mutex
}

// NewClient creates a new transport client
func NewClient(cfg *config.Config) (*Client, error) {
	// Create queue directory if needed
	if err := os.MkdirAll(cfg.Agent.QueueDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create queue directory: %w", err)
	}

	return &Client{
		cfg:      cfg,
		queueDir: cfg.Agent.QueueDir,
		httpClient: &http.Client{
			Timeout: cfg.Transport.Timeout,
		},
	}, nil
}

// SendEvents sends a batch of events to the API
func (c *Client) SendEvents(events interface{}) error {
	data, err := json.Marshal(events)
	if err != nil {
		return fmt.Errorf("failed to marshal events: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/agent/events", c.cfg.Server.URL)

	var lastErr error
	for attempt := 0; attempt < c.cfg.Transport.RetryAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(c.cfg.Transport.RetryDelay)
		}

		req, err := http.NewRequest("POST", url, bytes.NewReader(data))
		if err != nil {
			lastErr = err
			continue
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", c.cfg.Server.APIKey)
		req.Header.Set("X-Agent-ID", c.cfg.Agent.ID)
		req.Header.Set("X-Hostname", c.cfg.Agent.Hostname)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}

		body, _ := io.ReadAll(resp.Body)
		lastErr = fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	// All retries failed, queue for later
	if err := c.queueEvents(data); err != nil {
		return fmt.Errorf("failed to send and queue events: send error: %v, queue error: %w", lastErr, err)
	}

	return fmt.Errorf("events queued for later delivery: %w", lastErr)
}

// queueEvents saves events to disk for later retry
func (c *Client) queueEvents(data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	filename := filepath.Join(c.queueDir, fmt.Sprintf("events_%d.json", time.Now().UnixNano()))
	return os.WriteFile(filename, data, 0600)
}

// ProcessQueue attempts to send queued events
func (c *Client) ProcessQueue() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	files, err := filepath.Glob(filepath.Join(c.queueDir, "events_*.json"))
	if err != nil {
		return err
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		url := fmt.Sprintf("%s/api/v1/agent/events", c.cfg.Server.URL)
		req, err := http.NewRequest("POST", url, bytes.NewReader(data))
		if err != nil {
			continue
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", c.cfg.Server.APIKey)
		req.Header.Set("X-Agent-ID", c.cfg.Agent.ID)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			continue
		}
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			os.Remove(file)
		}
	}

	return nil
}

// Close cleans up the client
func (c *Client) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}
