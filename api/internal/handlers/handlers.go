package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/folt-labs/sentinel/api/internal/config"
	"github.com/folt-labs/sentinel/api/internal/database"
	"github.com/folt-labs/sentinel/api/internal/middleware"
	"github.com/folt-labs/sentinel/api/internal/services"
	"github.com/folt-labs/sentinel/api/internal/websocket"
)

// Handler contains all HTTP handlers
type Handler struct {
	db            *database.DB
	cfg           *config.Config
	auth          *services.AuthService
	servers       *services.ServerService
	events        *services.EventService
	alerts        *services.AlertService
	notifications *services.NotificationService
	wsHub         *websocket.Hub
}

// New creates a new handler instance
func New(db *database.DB, cfg *config.Config, wsHub *websocket.Hub) *Handler {
	return &Handler{
		db:            db,
		cfg:           cfg,
		auth:          services.NewAuthService(db, cfg),
		servers:       services.NewServerService(db),
		events:        services.NewEventService(db),
		alerts:        services.NewAlertService(db),
		notifications: services.NewNotificationService(db, cfg),
		wsHub:         wsHub,
	}
}

// Setup configures all routes
func Setup(app *fiber.App, db *database.DB, cfg *config.Config, wsHub *websocket.Hub) {
	h := New(db, cfg, wsHub)

	// Health check
	app.Get("/health", h.HealthCheck)

	// WebSocket endpoint
	wsHandler := websocket.NewHandler(wsHub, websocket.Config{
		JWTSecret: cfg.JWT.Secret,
	})
	app.Get("/ws", wsHandler.HandleConnection)

	// API v1 routes
	api := app.Group("/api/v1")

	// Public routes with auth rate limiting
	auth := api.Group("/auth", middleware.AuthRateLimit())
	auth.Post("/register", h.Register)
	auth.Post("/login", h.Login)

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
	protected.Get("/servers/:id/metrics", h.GetServerMetrics)

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
	settings.Get("/notification-channels/:id", h.GetNotificationChannel)
	settings.Put("/notification-channels/:id", h.UpdateNotificationChannel)
	settings.Delete("/notification-channels/:id", h.DeleteNotificationChannel)
	settings.Post("/notification-channels/:id/test", h.TestNotificationChannel)
}

// HealthCheck returns server health status
func (h *Handler) HealthCheck(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status": "ok",
		"time":   "now",
	})
}
