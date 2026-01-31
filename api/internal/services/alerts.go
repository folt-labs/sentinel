package services

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/folt-labs/sentinel/api/internal/database"
	"github.com/folt-labs/sentinel/api/internal/models"
)

// AlertService handles alert operations
type AlertService struct {
	db *database.DB
}

// NewAlertService creates a new alert service
func NewAlertService(db *database.DB) *AlertService {
	return &AlertService{db: db}
}

// DashboardSummary represents dashboard statistics
type DashboardSummary struct {
	TotalServers   int            `json:"total_servers"`
	OnlineServers  int            `json:"online_servers"`
	OpenAlerts     int            `json:"open_alerts"`
	CriticalAlerts int            `json:"critical_alerts"`
	EventsToday    int            `json:"events_today"`
	ServersByStatus map[string]int `json:"servers_by_status"`
}

// List returns alerts for an organization
func (s *AlertService) List(ctx context.Context, orgID uuid.UUID, status, severity string, limit, offset int) ([]models.Alert, error) {
	query := `SELECT id, organization_id, server_id, severity, status, title, description, triggered_at, acknowledged_at, resolved_at, occurrence_count, last_occurrence
		 FROM alerts WHERE organization_id = $1`
	args := []interface{}{orgID}
	argIdx := 2

	if status != "" {
		query += ` AND status = $` + string(rune('0'+argIdx))
		args = append(args, status)
		argIdx++
	}

	if severity != "" {
		query += ` AND severity = $` + string(rune('0'+argIdx))
		args = append(args, severity)
		argIdx++
	}

	query += ` ORDER BY triggered_at DESC LIMIT $` + string(rune('0'+argIdx)) + ` OFFSET $` + string(rune('0'+argIdx+1))
	args = append(args, limit, offset)

	rows, err := s.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []models.Alert
	for rows.Next() {
		var alert models.Alert
		err := rows.Scan(
			&alert.ID, &alert.OrganizationID, &alert.ServerID, &alert.Severity,
			&alert.Status, &alert.Title, &alert.Description, &alert.TriggeredAt,
			&alert.AcknowledgedAt, &alert.ResolvedAt, &alert.OccurrenceCount, &alert.LastOccurrence)
		if err != nil {
			return nil, err
		}
		alerts = append(alerts, alert)
	}

	return alerts, nil
}

// Get returns a specific alert
func (s *AlertService) Get(ctx context.Context, orgID, alertID uuid.UUID) (*models.Alert, error) {
	var alert models.Alert
	err := s.db.Pool.QueryRow(ctx,
		`SELECT id, organization_id, server_id, severity, status, title, description, triggered_at, acknowledged_at, resolved_at, occurrence_count, last_occurrence
		 FROM alerts WHERE id = $1 AND organization_id = $2`, alertID, orgID).Scan(
		&alert.ID, &alert.OrganizationID, &alert.ServerID, &alert.Severity,
		&alert.Status, &alert.Title, &alert.Description, &alert.TriggeredAt,
		&alert.AcknowledgedAt, &alert.ResolvedAt, &alert.OccurrenceCount, &alert.LastOccurrence)
	if err != nil {
		return nil, err
	}
	return &alert, nil
}

// Create creates a new alert
func (s *AlertService) Create(ctx context.Context, orgID, serverID uuid.UUID, severity, title, description string) (*models.Alert, error) {
	alertID := uuid.New()
	now := time.Now()

	_, err := s.db.Pool.Exec(ctx,
		`INSERT INTO alerts (id, organization_id, server_id, severity, status, title, description, triggered_at, occurrence_count)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 1)`,
		alertID, orgID, serverID, severity, models.AlertStatusOpen, title, description, now)
	if err != nil {
		return nil, err
	}

	return &models.Alert{
		ID:              alertID,
		OrganizationID:  orgID,
		ServerID:        serverID,
		Severity:        severity,
		Status:          models.AlertStatusOpen,
		Title:           title,
		Description:     description,
		TriggeredAt:     now,
		OccurrenceCount: 1,
	}, nil
}

// Acknowledge marks an alert as acknowledged
func (s *AlertService) Acknowledge(ctx context.Context, orgID, alertID, userID uuid.UUID) error {
	now := time.Now()
	_, err := s.db.Pool.Exec(ctx,
		`UPDATE alerts SET status = $1, acknowledged_at = $2, acknowledged_by = $3
		 WHERE id = $4 AND organization_id = $5`,
		models.AlertStatusAcknowledged, now, userID, alertID, orgID)
	return err
}

// Resolve marks an alert as resolved
func (s *AlertService) Resolve(ctx context.Context, orgID, alertID, userID uuid.UUID) error {
	now := time.Now()
	_, err := s.db.Pool.Exec(ctx,
		`UPDATE alerts SET status = $1, resolved_at = $2, resolved_by = $3
		 WHERE id = $4 AND organization_id = $5`,
		models.AlertStatusResolved, now, userID, alertID, orgID)
	return err
}

// ProcessEvents checks events and creates alerts as needed (deprecated, use CreateWithNotification)
func (s *AlertService) ProcessEvents(ctx context.Context, orgID, serverID uuid.UUID, events []AgentEvent) {
	for _, event := range events {
		// Create alerts for high-severity events
		if event.Severity == "high" || event.Severity == "critical" {
			title := getAlertTitle(event.Type)
			description := getAlertDescription(event)

			_, err := s.Create(ctx, orgID, serverID, event.Severity, title, description)
			if err != nil {
				log.Printf("Failed to create alert: %v", err)
			}
		}
	}
}

// CreateWithNotification creates an alert from an event and returns it for notification
func (s *AlertService) CreateWithNotification(ctx context.Context, orgID, serverID uuid.UUID, severity, eventType string, data map[string]interface{}) (*models.Alert, error) {
	title := getAlertTitle(eventType)
	description := getAlertDescription(AgentEvent{
		Type:     eventType,
		Severity: severity,
		Data:     data,
	})

	return s.Create(ctx, orgID, serverID, severity, title, description)
}

// FindOrCreateWithNotification finds an existing open alert within the deduplication window,
// or creates a new one. Returns (alert, isNew, error).
func (s *AlertService) FindOrCreateWithNotification(ctx context.Context, orgID, serverID uuid.UUID, severity, eventType string, data map[string]interface{}) (*models.Alert, bool, error) {
	title := getAlertTitle(eventType)
	description := getAlertDescription(AgentEvent{
		Type:     eventType,
		Severity: severity,
		Data:     data,
	})

	// Check for existing open alert within 5-minute window
	var existingAlert models.Alert
	err := s.db.Pool.QueryRow(ctx,
		`SELECT id, organization_id, server_id, severity, status, title, description, triggered_at, acknowledged_at, resolved_at, occurrence_count, last_occurrence
		 FROM alerts
		 WHERE server_id = $1 AND title = $2 AND status = 'open' AND triggered_at > (NOW() - INTERVAL '5 minutes')
		 ORDER BY triggered_at DESC
		 LIMIT 1`, serverID, title).Scan(
		&existingAlert.ID, &existingAlert.OrganizationID, &existingAlert.ServerID, &existingAlert.Severity,
		&existingAlert.Status, &existingAlert.Title, &existingAlert.Description, &existingAlert.TriggeredAt,
		&existingAlert.AcknowledgedAt, &existingAlert.ResolvedAt, &existingAlert.OccurrenceCount, &existingAlert.LastOccurrence)

	// Found existing alert - update occurrence count
	if err == nil {
		now := time.Now()
		_, updateErr := s.db.Pool.Exec(ctx,
			`UPDATE alerts SET occurrence_count = occurrence_count + 1, last_occurrence = $1 WHERE id = $2`,
			now, existingAlert.ID)
		if updateErr != nil {
			return nil, false, updateErr
		}
		existingAlert.OccurrenceCount++
		existingAlert.LastOccurrence = &now
		return &existingAlert, false, nil
	}

	// Check if it's a real error vs just no rows found
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, false, err
	}

	// No existing alert found - create new one
	alert, err := s.Create(ctx, orgID, serverID, severity, title, description)
	if err != nil {
		return nil, false, err
	}
	return alert, true, nil
}

// GetDashboardSummary returns dashboard statistics
func (s *AlertService) GetDashboardSummary(ctx context.Context, orgID uuid.UUID) (*DashboardSummary, error) {
	summary := &DashboardSummary{
		ServersByStatus: make(map[string]int),
	}

	// Count servers
	err := s.db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM servers WHERE organization_id = $1`, orgID).Scan(&summary.TotalServers)
	if err != nil {
		return nil, err
	}

	err = s.db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM servers WHERE organization_id = $1 AND status = 'online'`, orgID).Scan(&summary.OnlineServers)
	if err != nil {
		return nil, err
	}

	// Count alerts
	err = s.db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM alerts WHERE organization_id = $1 AND status = 'open'`, orgID).Scan(&summary.OpenAlerts)
	if err != nil {
		return nil, err
	}

	err = s.db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM alerts WHERE organization_id = $1 AND status = 'open' AND severity = 'critical'`, orgID).Scan(&summary.CriticalAlerts)
	if err != nil {
		return nil, err
	}

	// Count events today
	today := time.Now().Truncate(24 * time.Hour)
	err = s.db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM security_events e
		 JOIN servers s ON e.server_id = s.id
		 WHERE s.organization_id = $1 AND e.timestamp > $2`, orgID, today).Scan(&summary.EventsToday)
	if err != nil {
		return nil, err
	}

	// Servers by status
	rows, err := s.db.Pool.Query(ctx,
		`SELECT status, COUNT(*) FROM servers WHERE organization_id = $1 GROUP BY status`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		summary.ServersByStatus[status] = count
	}

	return summary, nil
}

// ListNotificationChannels returns all notification channels
func (s *AlertService) ListNotificationChannels(ctx context.Context, orgID uuid.UUID) ([]models.NotificationChannel, error) {
	rows, err := s.db.Pool.Query(ctx,
		`SELECT id, organization_id, name, type, config, enabled, created_at, updated_at
		 FROM notification_channels WHERE organization_id = $1`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []models.NotificationChannel
	for rows.Next() {
		var channel models.NotificationChannel
		err := rows.Scan(
			&channel.ID, &channel.OrganizationID, &channel.Name, &channel.Type,
			&channel.Config, &channel.Enabled, &channel.CreatedAt, &channel.UpdatedAt)
		if err != nil {
			return nil, err
		}
		channels = append(channels, channel)
	}

	return channels, nil
}

// CreateNotificationChannel creates a new notification channel
func (s *AlertService) CreateNotificationChannel(ctx context.Context, orgID uuid.UUID, name, channelType string, config map[string]interface{}) (*models.NotificationChannel, error) {
	channelID := uuid.New()
	now := time.Now()

	_, err := s.db.Pool.Exec(ctx,
		`INSERT INTO notification_channels (id, organization_id, name, type, config, enabled, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, true, $6, $6)`,
		channelID, orgID, name, channelType, config, now)
	if err != nil {
		return nil, err
	}

	return &models.NotificationChannel{
		ID:             channelID,
		OrganizationID: orgID,
		Name:           name,
		Type:           channelType,
		Config:         config,
		Enabled:        true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

// DeleteNotificationChannel deletes a notification channel
func (s *AlertService) DeleteNotificationChannel(ctx context.Context, orgID, channelID uuid.UUID) error {
	_, err := s.db.Pool.Exec(ctx,
		`DELETE FROM notification_channels WHERE id = $1 AND organization_id = $2`, channelID, orgID)
	return err
}

// GetNotificationChannel returns a specific notification channel
func (s *AlertService) GetNotificationChannel(ctx context.Context, orgID, channelID uuid.UUID) (*models.NotificationChannel, error) {
	var channel models.NotificationChannel
	err := s.db.Pool.QueryRow(ctx,
		`SELECT id, organization_id, name, type, config, enabled, created_at, updated_at
		 FROM notification_channels WHERE id = $1 AND organization_id = $2`,
		channelID, orgID).Scan(
		&channel.ID, &channel.OrganizationID, &channel.Name, &channel.Type,
		&channel.Config, &channel.Enabled, &channel.CreatedAt, &channel.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &channel, nil
}

// UpdateNotificationChannel updates a notification channel
func (s *AlertService) UpdateNotificationChannel(ctx context.Context, orgID, channelID uuid.UUID, name string, config map[string]interface{}, enabled bool) error {
	_, err := s.db.Pool.Exec(ctx,
		`UPDATE notification_channels SET name = $1, config = $2, enabled = $3, updated_at = NOW()
		 WHERE id = $4 AND organization_id = $5`,
		name, config, enabled, channelID, orgID)
	return err
}

func getAlertTitle(eventType string) string {
	titles := map[string]string{
		"ssh_login_failed":          "Failed SSH Login Attempt",
		"ssh_invalid_user":          "SSH Login with Invalid User",
		"file_modified":             "Critical File Modified",
		"file_deleted":              "Critical File Deleted",
		"file_permissions_changed":  "File Permissions Changed",
		"port_opened":               "New Port Opened",
		"high_cpu_usage":            "High CPU Usage",
		"high_memory_usage":         "High Memory Usage",
		"high_disk_usage":           "High Disk Usage",
		"sudo_auth_failed":          "Sudo Authentication Failed",
	}

	if title, ok := titles[eventType]; ok {
		return title
	}
	return "Security Event: " + eventType
}

func getAlertDescription(event AgentEvent) string {
	// Generate description from event data
	if rawLog, ok := event.Data["raw_log"].(string); ok {
		return rawLog
	}
	return "A security event was detected that requires attention."
}
