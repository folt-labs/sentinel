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

// GetByServer returns events for a server
func (s *EventService) GetByServer(ctx context.Context, serverID uuid.UUID, limit, offset int) ([]models.SecurityEvent, error) {
	rows, err := s.db.Pool.Query(ctx,
		`SELECT id, server_id, event_type, severity, timestamp, data
		 FROM security_events
		 WHERE server_id = $1
		 ORDER BY timestamp DESC
		 LIMIT $2 OFFSET $3`, serverID, limit, offset)
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
