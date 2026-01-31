package handlers

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/folt-labs/sentinel/api/internal/services"
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

	// Process alerts for high-severity events
	go h.alerts.ProcessEvents(c.Context(), server.OrganizationID, server.ID, batch.Events)

	return c.JSON(fiber.Map{
		"status":   "ok",
		"received": count,
	})
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
