package handlers

import (
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/folt-labs/sentinel/api/internal/websocket"
)

// CreateServerRequest represents the server creation payload
type CreateServerRequest struct {
	Hostname  string `json:"hostname"`
	IPAddress string `json:"ip_address"`
}

// ListServers returns all servers for the organization
func (h *Handler) ListServers(c fiber.Ctx) error {
	orgID := c.Locals("organization_id").(uuid.UUID)

	servers, err := h.servers.List(c.Context(), orgID)
	if err != nil {
		log.Printf("ListServers error: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch servers",
		})
	}

	return c.JSON(fiber.Map{
		"servers": servers,
	})
}

// GetServer returns a specific server
func (h *Handler) GetServer(c fiber.Ctx) error {
	orgID := c.Locals("organization_id").(uuid.UUID)
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid server ID",
		})
	}

	server, err := h.servers.Get(c.Context(), orgID, serverID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Server not found",
		})
	}

	return c.JSON(server)
}

// CreateServer creates a new server entry and returns an API key
func (h *Handler) CreateServer(c fiber.Ctx) error {
	orgID := c.Locals("organization_id").(uuid.UUID)

	var req CreateServerRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Hostname == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Hostname is required",
		})
	}

	result, err := h.servers.Create(c.Context(), orgID, req.Hostname, req.IPAddress)
	if err != nil {
		log.Printf("CreateServer error: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create server",
		})
	}

	// Broadcast server created via WebSocket
	if h.wsHub != nil {
		h.wsHub.Broadcast(orgID, websocket.MessageTypeServerCreated, map[string]interface{}{
			"server": result.Server,
		})
	}

	return c.Status(fiber.StatusCreated).JSON(result)
}

// DeleteServer removes a server
func (h *Handler) DeleteServer(c fiber.Ctx) error {
	orgID := c.Locals("organization_id").(uuid.UUID)
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid server ID",
		})
	}

	if err := h.servers.Delete(c.Context(), orgID, serverID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete server",
		})
	}

	// Broadcast server deleted via WebSocket
	if h.wsHub != nil {
		h.wsHub.Broadcast(orgID, websocket.MessageTypeServerDeleted, map[string]interface{}{
			"server_id": serverID.String(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// GetServerEvents returns events for a specific server
func (h *Handler) GetServerEvents(c fiber.Ctx) error {
	orgID := c.Locals("organization_id").(uuid.UUID)
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid server ID",
		})
	}

	// Verify server belongs to organization
	_, err = h.servers.Get(c.Context(), orgID, serverID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Server not found",
		})
	}

	limit := queryInt(c, "limit", 100)
	offset := queryInt(c, "offset", 0)

	// Parse optional date filters
	var startTime, endTime *time.Time
	if startDateStr := c.Query("start_date"); startDateStr != "" {
		if t, err := time.Parse(time.RFC3339, startDateStr); err == nil {
			startTime = &t
		}
	}
	if endDateStr := c.Query("end_date"); endDateStr != "" {
		if t, err := time.Parse(time.RFC3339, endDateStr); err == nil {
			endTime = &t
		}
	}

	events, total, err := h.events.GetByServer(c.Context(), serverID, limit, offset, startTime, endTime)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch events",
		})
	}

	return c.JSON(fiber.Map{
		"events": events,
		"total":  total,
	})
}

// GetServerMetrics returns time-bucketed metrics for a server
func (h *Handler) GetServerMetrics(c fiber.Ctx) error {
	orgID := c.Locals("organization_id").(uuid.UUID)
	serverID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid server ID",
		})
	}

	// Verify server belongs to organization
	_, err = h.servers.Get(c.Context(), orgID, serverID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Server not found",
		})
	}

	// Parse query params
	metricsStr := c.Query("metrics", "cpu_percent,memory_percent,disk_percent")
	metrics := strings.Split(metricsStr, ",")

	bucket := queryInt(c, "bucket", 5)

	// Default time range: last 24 hours
	endTime := time.Now()
	startTime := endTime.Add(-24 * time.Hour)

	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			startTime = t
		}
	}
	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			endTime = t
		}
	}

	metricSeries, err := h.events.GetMetrics(c.Context(), serverID, metrics, startTime, endTime, bucket)
	if err != nil {
		log.Printf("GetServerMetrics error: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch metrics",
		})
	}

	return c.JSON(fiber.Map{
		"metrics": metricSeries,
	})
}
