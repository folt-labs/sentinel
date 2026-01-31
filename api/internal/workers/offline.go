package workers

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/folt-labs/sentinel/api/internal/database"
	"github.com/folt-labs/sentinel/api/internal/models"
	"github.com/folt-labs/sentinel/api/internal/websocket"
)

// OfflineDetector checks for servers that have gone offline
type OfflineDetector struct {
	db            *database.DB
	checkInterval time.Duration
	offlineAfter  time.Duration
	notifier      Notifier
	wsHub         *websocket.Hub
}

// Notifier interface for sending notifications
type Notifier interface {
	NotifyServerOffline(ctx context.Context, orgID, serverID uuid.UUID, hostname string) error
}

// NewOfflineDetector creates a new offline detector
func NewOfflineDetector(db *database.DB, notifier Notifier, wsHub *websocket.Hub) *OfflineDetector {
	return &OfflineDetector{
		db:            db,
		checkInterval: 60 * time.Second,
		offlineAfter:  5 * time.Minute,
		notifier:      notifier,
		wsHub:         wsHub,
	}
}

// Start begins the offline detection loop
func (d *OfflineDetector) Start(ctx context.Context) {
	log.Println("Starting offline detector worker")
	ticker := time.NewTicker(d.checkInterval)
	defer ticker.Stop()

	// Run immediately on start
	d.checkOfflineServers(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("Offline detector stopped")
			return
		case <-ticker.C:
			d.checkOfflineServers(ctx)
		}
	}
}

func (d *OfflineDetector) checkOfflineServers(ctx context.Context) {
	threshold := time.Now().Add(-d.offlineAfter)

	// Find servers that were online but haven't reported recently
	rows, err := d.db.Pool.Query(ctx,
		`SELECT id, organization_id, hostname FROM servers
		 WHERE status = $1 AND last_seen < $2`,
		models.StatusOnline, threshold)
	if err != nil {
		log.Printf("Error checking offline servers: %v", err)
		return
	}
	defer rows.Close()

	var offlineServers []struct {
		ID             uuid.UUID
		OrganizationID uuid.UUID
		Hostname       string
	}

	for rows.Next() {
		var s struct {
			ID             uuid.UUID
			OrganizationID uuid.UUID
			Hostname       string
		}
		if err := rows.Scan(&s.ID, &s.OrganizationID, &s.Hostname); err != nil {
			log.Printf("Error scanning server row: %v", err)
			continue
		}
		offlineServers = append(offlineServers, s)
	}

	// Mark each server as offline and create alerts
	for _, server := range offlineServers {
		// Update status
		_, err := d.db.Pool.Exec(ctx,
			`UPDATE servers SET status = $1, updated_at = NOW() WHERE id = $2`,
			models.StatusOffline, server.ID)
		if err != nil {
			log.Printf("Error updating server status: %v", err)
			continue
		}

		// Check if there's already an open offline alert for this server
		var existingCount int
		err = d.db.Pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM alerts
			 WHERE server_id = $1 AND title = 'Server Offline' AND status = 'open'`,
			server.ID).Scan(&existingCount)
		if err != nil {
			log.Printf("Error checking existing alerts: %v", err)
			continue
		}

		if existingCount > 0 {
			continue // Don't create duplicate alerts
		}

		// Create an alert
		alertID := uuid.New()
		_, err = d.db.Pool.Exec(ctx,
			`INSERT INTO alerts (id, organization_id, server_id, severity, status, title, description, triggered_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())`,
			alertID, server.OrganizationID, server.ID, models.SeverityHigh,
			models.AlertStatusOpen, "Server Offline",
			"Server '"+server.Hostname+"' has stopped reporting and is now offline.")
		if err != nil {
			log.Printf("Error creating offline alert: %v", err)
			continue
		}

		log.Printf("Server %s marked offline, alert created", server.Hostname)

		// Broadcast server status change via WebSocket
		if d.wsHub != nil {
			d.wsHub.Broadcast(server.OrganizationID, websocket.MessageTypeServerStatus, map[string]interface{}{
				"server_id": server.ID.String(),
				"hostname":  server.Hostname,
				"status":    "offline",
			})
			d.wsHub.Broadcast(server.OrganizationID, websocket.MessageTypeAlert, map[string]interface{}{
				"id":          alertID.String(),
				"server_id":   server.ID.String(),
				"hostname":    server.Hostname,
				"severity":    models.SeverityHigh,
				"title":       "Server Offline",
				"description": "Server '" + server.Hostname + "' has stopped reporting and is now offline.",
			})
		}

		// Send notification
		if d.notifier != nil {
			go func(orgID, serverID uuid.UUID, hostname string) {
				if err := d.notifier.NotifyServerOffline(ctx, orgID, serverID, hostname); err != nil {
					log.Printf("Error sending offline notification: %v", err)
				}
			}(server.OrganizationID, server.ID, server.Hostname)
		}
	}
}
