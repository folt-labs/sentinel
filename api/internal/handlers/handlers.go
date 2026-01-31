package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/folt-labs/sentinel/api/internal/config"
	"github.com/folt-labs/sentinel/api/internal/database"
	"github.com/folt-labs/sentinel/api/internal/middleware"
	"github.com/folt-labs/sentinel/api/internal/services"
)

// Handler contains all HTTP handlers
type Handler struct {
	db       *database.DB
	cfg      *config.Config
	auth     *services.AuthService
	servers  *services.ServerService
	events   *services.EventService
	alerts   *services.AlertService
}

// New creates a new handler instance
func New(db *database.DB, cfg *config.Config) *Handler {
	return &Handler{
		db:      db,
		cfg:     cfg,
		auth:    services.NewAuthService(db, cfg),
		servers: services.NewServerService(db),
		events:  services.NewEventService(db),
		alerts:  services.NewAlertService(db),
	}
}

// Setup configures all routes
func Setup(app *fiber.App, db *database.DB, cfg *config.Config) {
	h := New(db, cfg)

	// Health check
	app.Get("/health", h.HealthCheck)

	// API v1 routes
	api := app.Group("/api/v1")

	// Public routes
	api.Post("/auth/register", h.Register)
	api.Post("/auth/login", h.Login)

	// Agent routes (API key auth)
	agent := api.Group("/agent", middleware.APIKeyAuth())
	agent.Post("/register", h.RegisterAgent)
	agent.Post("/events", h.IngestEvents)
	agent.Post("/heartbeat", h.AgentHeartbeat)

	// Protected routes (JWT auth)
	protected := api.Group("", middleware.JWTAuth(cfg.JWT.Secret))

	// User routes
	protected.Get("/me", h.GetCurrentUser)
	protected.Put("/me", h.UpdateCurrentUser)

	// Server routes
	protected.Get("/servers", h.ListServers)
	protected.Get("/servers/:id", h.GetServer)
	protected.Post("/servers", h.CreateServer)
	protected.Delete("/servers/:id", h.DeleteServer)
	protected.Get("/servers/:id/events", h.GetServerEvents)

	// Alert routes
	protected.Get("/alerts", h.ListAlerts)
	protected.Get("/alerts/:id", h.GetAlert)
	protected.Post("/alerts/:id/acknowledge", h.AcknowledgeAlert)
	protected.Post("/alerts/:id/resolve", h.ResolveAlert)

	// Dashboard routes
	protected.Get("/dashboard/summary", h.DashboardSummary)

	// Settings routes (admin only)
	settings := protected.Group("/settings", middleware.RequireRole("owner", "admin"))
	settings.Get("/notification-channels", h.ListNotificationChannels)
	settings.Post("/notification-channels", h.CreateNotificationChannel)
	settings.Delete("/notification-channels/:id", h.DeleteNotificationChannel)
}

// HealthCheck returns server health status
func (h *Handler) HealthCheck(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status": "ok",
		"time":   "now",
	})
}
