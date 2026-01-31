package websocket

import (
	"log"
	"strings"

	"github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
)

var upgrader = websocket.FastHTTPUpgrader{
	CheckOrigin: func(ctx *fasthttp.RequestCtx) bool {
		// In production, validate origin against allowed hosts
		return true
	},
}

// Config holds WebSocket handler configuration
type Config struct {
	JWTSecret string
}

// Handler handles WebSocket connections
type Handler struct {
	hub *Hub
	cfg Config
}

// NewHandler creates a new WebSocket handler
func NewHandler(hub *Hub, cfg Config) *Handler {
	return &Handler{
		hub: hub,
		cfg: cfg,
	}
}

// HandleConnection handles WebSocket upgrade and connection
func (h *Handler) HandleConnection(c fiber.Ctx) error {
	// Get token from query parameter (WebSocket can't use headers easily)
	token := c.Query("token")
	if token == "" {
		// Also check Authorization header for regular HTTP upgrade
		authHeader := c.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "missing authentication token",
		})
	}

	// Parse and validate JWT
	claims, err := h.parseToken(token)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "invalid token",
		})
	}

	orgID, err := uuid.Parse(claims.OrganizationID)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "invalid organization ID in token",
		})
	}

	// Upgrade to WebSocket
	err = upgrader.Upgrade(c.Context(), func(conn *websocket.Conn) {
		client := &Client{
			ID:             uuid.New().String(),
			OrganizationID: orgID,
			Conn:           conn,
			Send:           make(chan []byte, 256),
			Hub:            h.hub,
		}

		h.hub.register <- client

		// Start write pump in goroutine
		go client.WritePump()

		// Run read pump (blocks until connection closes)
		client.ReadPump()
	})

	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to upgrade to WebSocket",
		})
	}

	return nil
}

// JWTClaims represents the JWT payload
type JWTClaims struct {
	UserID         string `json:"user_id"`
	OrganizationID string `json:"organization_id"`
	Role           string `json:"role"`
	jwt.RegisteredClaims
}

func (h *Handler) parseToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(h.cfg.JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrSignatureInvalid
}
