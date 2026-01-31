package system

import (
	"context"
	"time"

	"github.com/folt-labs/sentinel/agent/internal/types"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
)

// ResourcesCollector monitors system resources
type ResourcesCollector struct {
	thresholds Thresholds
}

// Thresholds for alerting
type Thresholds struct {
	CPUPercent    float64
	MemoryPercent float64
	DiskPercent   float64
}

// DefaultThresholds returns default alert thresholds
func DefaultThresholds() Thresholds {
	return Thresholds{
		CPUPercent:    90.0,
		MemoryPercent: 90.0,
		DiskPercent:   90.0,
	}
}

// NewResourcesCollector creates a new resources collector
func NewResourcesCollector() *ResourcesCollector {
	return &ResourcesCollector{
		thresholds: DefaultThresholds(),
	}
}

// Name returns the collector name
func (c *ResourcesCollector) Name() string {
	return "resources"
}

// Collect gathers system resource metrics
func (c *ResourcesCollector) Collect(ctx context.Context) ([]types.Event, error) {
	var events []types.Event
	now := time.Now()

	// CPU usage
	cpuPercent, err := cpu.Percent(time.Second, false)
	if err == nil && len(cpuPercent) > 0 {
		usage := cpuPercent[0]
		event := types.Event{
			Type:      "system_resources",
			Severity:  "info",
			Timestamp: now,
			Data: map[string]interface{}{
				"metric": "cpu_percent",
				"value":  usage,
			},
		}

		if usage > c.thresholds.CPUPercent {
			event.Severity = "warning"
			event.Type = "high_cpu_usage"
		}

		events = append(events, event)
	}

	// Memory usage
	memInfo, err := mem.VirtualMemory()
	if err == nil {
		event := types.Event{
			Type:      "system_resources",
			Severity:  "info",
			Timestamp: now,
			Data: map[string]interface{}{
				"metric":    "memory_percent",
				"value":     memInfo.UsedPercent,
				"total":     memInfo.Total,
				"used":      memInfo.Used,
				"available": memInfo.Available,
			},
		}

		if memInfo.UsedPercent > c.thresholds.MemoryPercent {
			event.Severity = "warning"
			event.Type = "high_memory_usage"
		}

		events = append(events, event)
	}

	// Disk usage
	partitions, err := disk.Partitions(false)
	if err == nil {
		for _, partition := range partitions {
			usage, err := disk.Usage(partition.Mountpoint)
			if err != nil {
				continue
			}

			event := types.Event{
				Type:      "system_resources",
				Severity:  "info",
				Timestamp: now,
				Data: map[string]interface{}{
					"metric":     "disk_percent",
					"mountpoint": partition.Mountpoint,
					"device":     partition.Device,
					"value":      usage.UsedPercent,
					"total":      usage.Total,
					"used":       usage.Used,
					"free":       usage.Free,
				},
			}

			if usage.UsedPercent > c.thresholds.DiskPercent {
				event.Severity = "warning"
				event.Type = "high_disk_usage"
			}

			events = append(events, event)
		}
	}

	return events, nil
}
