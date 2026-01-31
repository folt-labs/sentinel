package system

import (
	"context"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/net"
)

// Event represents a security event
type Event struct {
	Type      string                 `json:"type"`
	Severity  string                 `json:"severity"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// PortInfo represents an open port
type PortInfo struct {
	Port     uint32
	Protocol string
	Address  string
	PID      int32
	Process  string
	Status   string
}

// PortsCollector monitors open ports
type PortsCollector struct {
	baseline    map[string]PortInfo
	mu          sync.Mutex
	initialized bool
}

// NewPortsCollector creates a new ports collector
func NewPortsCollector() *PortsCollector {
	return &PortsCollector{
		baseline: make(map[string]PortInfo),
	}
}

// Name returns the collector name
func (c *PortsCollector) Name() string {
	return "ports"
}

// Collect gathers open port information
func (c *PortsCollector) Collect(ctx context.Context) ([]Event, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	connections, err := net.Connections("all")
	if err != nil {
		return nil, err
	}

	var events []Event
	currentPorts := make(map[string]PortInfo)

	for _, conn := range connections {
		// Only track listening ports
		if conn.Status != "LISTEN" {
			continue
		}

		port := PortInfo{
			Port:     conn.Laddr.Port,
			Protocol: connType(conn.Type),
			Address:  conn.Laddr.IP,
			PID:      conn.Pid,
			Status:   conn.Status,
		}

		key := portKey(port)
		currentPorts[key] = port

		// First run - establish baseline
		if !c.initialized {
			continue
		}

		// New port opened
		if _, exists := c.baseline[key]; !exists {
			severity := "info"
			// High-risk ports
			if isHighRiskPort(port.Port) {
				severity = "high"
			}

			events = append(events, Event{
				Type:      "port_opened",
				Severity:  severity,
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"port":     port.Port,
					"protocol": port.Protocol,
					"address":  port.Address,
					"pid":      port.PID,
				},
			})
		}
	}

	// Check for closed ports
	if c.initialized {
		for key, port := range c.baseline {
			if _, exists := currentPorts[key]; !exists {
				events = append(events, Event{
					Type:      "port_closed",
					Severity:  "info",
					Timestamp: time.Now(),
					Data: map[string]interface{}{
						"port":     port.Port,
						"protocol": port.Protocol,
						"address":  port.Address,
					},
				})
			}
		}
	}

	c.baseline = currentPorts
	c.initialized = true

	return events, nil
}

func connType(t uint32) string {
	switch t {
	case 1:
		return "tcp"
	case 2:
		return "udp"
	default:
		return "unknown"
	}
}

func portKey(p PortInfo) string {
	return p.Protocol + ":" + p.Address + ":" + string(rune(p.Port))
}

func isHighRiskPort(port uint32) bool {
	highRiskPorts := map[uint32]bool{
		21:    true, // FTP
		23:    true, // Telnet
		3306:  true, // MySQL
		5432:  true, // PostgreSQL
		6379:  true, // Redis
		27017: true, // MongoDB
		11211: true, // Memcached
	}
	return highRiskPorts[port]
}
