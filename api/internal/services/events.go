package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/folt-labs/sentinel/api/internal/database"
	"github.com/folt-labs/sentinel/api/internal/models"
)

// EventService handles security event operations
type EventService struct {
	db *database.DB
}

// NewEventService creates a new event service
func NewEventService(db *database.DB) *EventService {
	return &EventService{db: db}
}

// AgentEvent represents an event from an agent
type AgentEvent struct {
	Type      string                 `json:"type"`
	Severity  string                 `json:"severity"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// MetricPoint represents a single data point in a time series
type MetricPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// MetricSeries represents a time series for a specific metric
type MetricSeries struct {
	Metric string        `json:"metric"`
	Data   []MetricPoint `json:"data"`
}

// IngestBatch stores a batch of events
func (s *EventService) IngestBatch(ctx context.Context, serverID uuid.UUID, events []AgentEvent) (int, error) {
	if len(events) == 0 {
		return 0, nil
	}

	// Use batch insert for efficiency
	batch := &insertBatch{}
	for _, event := range events {
		batch.queue(serverID, event)
	}

	return batch.execute(ctx, s.db)
}

type insertBatch struct {
	events []struct {
		serverID  uuid.UUID
		eventType string
		severity  string
		timestamp time.Time
		data      map[string]interface{}
	}
}

func (b *insertBatch) queue(serverID uuid.UUID, event AgentEvent) {
	b.events = append(b.events, struct {
		serverID  uuid.UUID
		eventType string
		severity  string
		timestamp time.Time
		data      map[string]interface{}
	}{
		serverID:  serverID,
		eventType: event.Type,
		severity:  event.Severity,
		timestamp: event.Timestamp,
		data:      event.Data,
	})
}

func (b *insertBatch) execute(ctx context.Context, db *database.DB) (int, error) {
	for _, event := range b.events {
		id := uuid.New()
		_, err := db.Pool.Exec(ctx,
			`INSERT INTO security_events (id, server_id, event_type, severity, timestamp, data)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			id, event.serverID, event.eventType, event.severity, event.timestamp, event.data)
		if err != nil {
			return 0, err
		}
	}
	return len(b.events), nil
}

// GetByServer returns events for a server with optional date filtering
func (s *EventService) GetByServer(ctx context.Context, serverID uuid.UUID, limit, offset int, startTime, endTime *time.Time) ([]models.SecurityEvent, int, error) {
	// Build query with optional date filters
	query := `SELECT id, server_id, event_type, severity, timestamp, data
		 FROM security_events
		 WHERE server_id = $1`
	countQuery := `SELECT COUNT(*) FROM security_events WHERE server_id = $1`

	args := []interface{}{serverID}
	argIdx := 2

	if startTime != nil {
		query += ` AND timestamp >= $` + string(rune('0'+argIdx))
		countQuery += ` AND timestamp >= $` + string(rune('0'+argIdx))
		args = append(args, *startTime)
		argIdx++
	}

	if endTime != nil {
		query += ` AND timestamp <= $` + string(rune('0'+argIdx))
		countQuery += ` AND timestamp <= $` + string(rune('0'+argIdx))
		args = append(args, *endTime)
		argIdx++
	}

	// Get total count
	var total int
	err := s.db.Pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Add ordering and pagination
	query += ` ORDER BY timestamp DESC LIMIT $` + string(rune('0'+argIdx)) + ` OFFSET $` + string(rune('0'+argIdx+1))
	args = append(args, limit, offset)

	rows, err := s.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []models.SecurityEvent
	for rows.Next() {
		var event models.SecurityEvent
		err := rows.Scan(&event.ID, &event.ServerID, &event.EventType, &event.Severity, &event.Timestamp, &event.Data)
		if err != nil {
			return nil, 0, err
		}
		events = append(events, event)
	}

	return events, total, nil
}

// GetRecent returns recent events across all servers in an organization
func (s *EventService) GetRecent(ctx context.Context, orgID uuid.UUID, limit int) ([]models.SecurityEvent, error) {
	rows, err := s.db.Pool.Query(ctx,
		`SELECT e.id, e.server_id, e.event_type, e.severity, e.timestamp, e.data
		 FROM security_events e
		 JOIN servers s ON e.server_id = s.id
		 WHERE s.organization_id = $1
		 ORDER BY e.timestamp DESC
		 LIMIT $2`, orgID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.SecurityEvent
	for rows.Next() {
		var event models.SecurityEvent
		err := rows.Scan(&event.ID, &event.ServerID, &event.EventType, &event.Severity, &event.Timestamp, &event.Data)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return events, nil
}

// CountByType returns event counts grouped by type
func (s *EventService) CountByType(ctx context.Context, orgID uuid.UUID, since time.Time) (map[string]int, error) {
	rows, err := s.db.Pool.Query(ctx,
		`SELECT e.event_type, COUNT(*)
		 FROM security_events e
		 JOIN servers s ON e.server_id = s.id
		 WHERE s.organization_id = $1 AND e.timestamp > $2
		 GROUP BY e.event_type`, orgID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var eventType string
		var count int
		if err := rows.Scan(&eventType, &count); err != nil {
			return nil, err
		}
		counts[eventType] = count
	}

	return counts, nil
}

// GetMetrics returns time-bucketed metrics for a server
func (s *EventService) GetMetrics(ctx context.Context, serverID uuid.UUID, metrics []string, startTime, endTime time.Time, bucketMinutes int) ([]MetricSeries, error) {
	if len(metrics) == 0 {
		metrics = []string{"cpu_percent", "memory_percent", "disk_percent"}
	}
	if bucketMinutes <= 0 {
		bucketMinutes = 5
	}

	// Use time_bucket for TimescaleDB, fallback to date_trunc for standard PostgreSQL
	query := `
		SELECT
			date_trunc('minute', timestamp) - (EXTRACT(MINUTE FROM timestamp)::int % $4) * INTERVAL '1 minute' AS bucket,
			data->>'metric' AS metric,
			AVG((data->>'value')::float) AS avg_value
		FROM security_events
		WHERE server_id = $1
			AND event_type = 'system_resources'
			AND timestamp BETWEEN $2 AND $3
			AND data->>'metric' = ANY($5)
		GROUP BY bucket, metric
		ORDER BY bucket`

	rows, err := s.db.Pool.Query(ctx, query, serverID, startTime, endTime, bucketMinutes, metrics)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Group results by metric
	seriesMap := make(map[string][]MetricPoint)
	for rows.Next() {
		var bucket time.Time
		var metric string
		var avgValue float64
		if err := rows.Scan(&bucket, &metric, &avgValue); err != nil {
			return nil, err
		}
		seriesMap[metric] = append(seriesMap[metric], MetricPoint{
			Timestamp: bucket,
			Value:     avgValue,
		})
	}

	// Convert map to slice
	var result []MetricSeries
	for _, m := range metrics {
		if data, ok := seriesMap[m]; ok {
			result = append(result, MetricSeries{
				Metric: m,
				Data:   data,
			})
		}
	}

	return result, nil
}
