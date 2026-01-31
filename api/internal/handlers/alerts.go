package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

// ListAlerts returns all alerts for the organization
func (h *Handler) ListAlerts(c fiber.Ctx) error {
	orgID := c.Locals("organization_id").(uuid.UUID)

	status := c.Query("status", "")
	severity := c.Query("severity", "")
	limit := queryInt(c, "limit", 50)
	offset := queryInt(c, "offset", 0)

	alerts, err := h.alerts.List(c.Context(), orgID, status, severity, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch alerts",
		})
	}

	return c.JSON(fiber.Map{
		"alerts": alerts,
	})
}

// GetAlert returns a specific alert
func (h *Handler) GetAlert(c fiber.Ctx) error {
	orgID := c.Locals("organization_id").(uuid.UUID)
	alertID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid alert ID",
		})
	}

	alert, err := h.alerts.Get(c.Context(), orgID, alertID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Alert not found",
		})
	}

	return c.JSON(alert)
}

// AcknowledgeAlert marks an alert as acknowledged
func (h *Handler) AcknowledgeAlert(c fiber.Ctx) error {
	orgID := c.Locals("organization_id").(uuid.UUID)
	userID := c.Locals("user_id").(uuid.UUID)
	alertID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid alert ID",
		})
	}

	if err := h.alerts.Acknowledge(c.Context(), orgID, alertID, userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to acknowledge alert",
		})
	}

	return c.JSON(fiber.Map{
		"status": "acknowledged",
	})
}

// ResolveAlert marks an alert as resolved
func (h *Handler) ResolveAlert(c fiber.Ctx) error {
	orgID := c.Locals("organization_id").(uuid.UUID)
	userID := c.Locals("user_id").(uuid.UUID)
	alertID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid alert ID",
		})
	}

	if err := h.alerts.Resolve(c.Context(), orgID, alertID, userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to resolve alert",
		})
	}

	return c.JSON(fiber.Map{
		"status": "resolved",
	})
}

// DashboardSummary returns summary statistics for the dashboard
func (h *Handler) DashboardSummary(c fiber.Ctx) error {
	orgID := c.Locals("organization_id").(uuid.UUID)

	summary, err := h.alerts.GetDashboardSummary(c.Context(), orgID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch dashboard summary",
		})
	}

	return c.JSON(summary)
}

// ListNotificationChannels returns all notification channels
func (h *Handler) ListNotificationChannels(c fiber.Ctx) error {
	orgID := c.Locals("organization_id").(uuid.UUID)

	channels, err := h.alerts.ListNotificationChannels(c.Context(), orgID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch notification channels",
		})
	}

	return c.JSON(fiber.Map{
		"channels": channels,
	})
}

// CreateNotificationChannel creates a new notification channel
func (h *Handler) CreateNotificationChannel(c fiber.Ctx) error {
	orgID := c.Locals("organization_id").(uuid.UUID)

	var req struct {
		Name   string                 `json:"name"`
		Type   string                 `json:"type"`
		Config map[string]interface{} `json:"config"`
	}
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Name == "" || req.Type == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Name and type are required",
		})
	}

	channel, err := h.alerts.CreateNotificationChannel(c.Context(), orgID, req.Name, req.Type, req.Config)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create notification channel",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(channel)
}

// DeleteNotificationChannel deletes a notification channel
func (h *Handler) DeleteNotificationChannel(c fiber.Ctx) error {
	orgID := c.Locals("organization_id").(uuid.UUID)
	channelID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid channel ID",
		})
	}

	if err := h.alerts.DeleteNotificationChannel(c.Context(), orgID, channelID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete notification channel",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// queryInt helper to parse int query params
func queryInt(c fiber.Ctx, key string, defaultValue int) int {
	val := c.Query(key)
	if val == "" {
		return defaultValue
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return defaultValue
	}
	return i
}
