package handlers

import (
	"context"
	"log"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/folt-labs/sentinel/api/internal/services"
	"github.com/folt-labs/sentinel/api/internal/websocket"
)

// AgentRegisterRequest represents an agent registration request
type AgentRegisterRequest struct {
	Hostname     string `json:"hostname"`
	IPAddress    string `json:"ip_address"`
	AgentVersion string `json:"agent_version"`
}

// EventBatch represents a batch of events from an agent
type EventBatch struct {
	Events []services.AgentEvent `json:"events"`
}

// RegisterAgent handles agent registration with API key
func (h *Handler) RegisterAgent(c fiber.Ctx) error {
	apiKey := c.Locals("api_key").(string)

	var req AgentRegisterRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate API key and get server
	server, err := h.servers.ValidateAPIKey(c.Context(), apiKey)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid API key",
		})
	}

	// Update server info
	if err := h.servers.UpdateAgentInfo(c.Context(), server.ID, req.Hostname, req.IPAddress, req.AgentVersion); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update server info",
		})
	}

	return c.JSON(fiber.Map{
		"status":    "registered",
		"server_id": server.ID,
	})
}

// IngestEvents receives events from an agent
func (h *Handler) IngestEvents(c fiber.Ctx) error {
	apiKey := c.Locals("api_key").(string)

	// Validate API key
	server, err := h.servers.ValidateAPIKey(c.Context(), apiKey)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid API key",
		})
	}

	var batch EventBatch
	if err := c.Bind().JSON(&batch); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Store events
	count, err := h.events.IngestBatch(c.Context(), server.ID, batch.Events)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to store events",
		})
	}

	// Update server last seen
	h.servers.UpdateLastSeen(c.Context(), server.ID)

	// Process alerts for high-severity events and send notifications
	go h.processEventsWithNotifications(server.OrganizationID, server.ID, server.Hostname, batch.Events)

	return c.JSON(fiber.Map{
		"status":   "ok",
		"received": count,
	})
}

// processEventsWithNotifications creates alerts and sends notifications
func (h *Handler) processEventsWithNotifications(orgID, serverID uuid.UUID, hostname string, events []services.AgentEvent) {
	ctx := context.Background()

	// Broadcast new events via WebSocket with FULL event data
	if h.wsHub != nil && len(events) > 0 {
		// Convert events to format matching SecurityEvent for frontend
		wsEvents := make([]map[string]interface{}, len(events))
		for i, e := range events {
			wsEvents[i] = map[string]interface{}{
				"id":         uuid.New().String(), // Generate ID for display
				"server_id":  serverID.String(),
				"event_type": e.Type,
				"severity":   e.Severity,
				"timestamp":  e.Timestamp,
				"data":       e.Data,
			}
		}
		h.wsHub.Broadcast(orgID, websocket.MessageTypeEvent, map[string]interface{}{
			"server_id": serverID.String(),
			"hostname":  hostname,
			"events":    wsEvents,
		})
	}

	for _, event := range events {
		if event.Severity == "high" || event.Severity == "critical" {
			alert, isNew, err := h.alerts.FindOrCreateWithNotification(ctx, orgID, serverID, event.Severity, event.Type, event.Data)
			if err != nil {
				log.Printf("Failed to create/update alert: %v", err)
				continue
			}

			// Only broadcast and notify for NEW alerts (not deduplicated ones)
			if isNew {
				// Broadcast alert via WebSocket
				if h.wsHub != nil {
					h.wsHub.Broadcast(orgID, websocket.MessageTypeAlert, map[string]interface{}{
						"id":          alert.ID.String(),
						"server_id":   serverID.String(),
						"hostname":    hostname,
						"severity":    alert.Severity,
						"title":       alert.Title,
						"description": alert.Description,
					})
				}

				// Send notifications (email/webhook)
				if err := h.notifications.NotifyAlert(ctx, alert, hostname); err != nil {
					log.Printf("Failed to send notification: %v", err)
				}
			}
		}
	}
}

// AgentHeartbeat handles agent heartbeat
func (h *Handler) AgentHeartbeat(c fiber.Ctx) error {
	apiKey := c.Locals("api_key").(string)

	server, err := h.servers.ValidateAPIKey(c.Context(), apiKey)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid API key",
		})
	}

	h.servers.UpdateLastSeen(c.Context(), server.ID)

	return c.JSON(fiber.Map{
		"status": "ok",
		"time":   time.Now().UTC(),
	})
}
