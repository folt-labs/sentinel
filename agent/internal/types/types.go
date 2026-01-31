package types

import "time"

// Event represents a security event to be sent to the API
type Event struct {
	Type      string                 `json:"type"`
	Severity  string                 `json:"severity"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// Collector interface for all data collectors
type Collector interface {
	Name() string
	Collect() ([]Event, error)
}
