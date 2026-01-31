package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
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
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create server",
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

	events, err := h.events.GetByServer(c.Context(), serverID, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch events",
		})
	}

	return c.JSON(fiber.Map{
		"events": events,
	})
}
