package middleware

import (
	"log"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/folt-labs/sentinel/api/internal/config"
)

// RateLimiter tracks request counts per IP
type RateLimiter struct {
	requests map[string]*requestCount
	mu       sync.RWMutex
	limit    int
	window   time.Duration
}

type requestCount struct {
	count    int
	resetAt  time.Time
	lockout  bool
	lockoutUntil time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string]*requestCount),
		limit:    limit,
		window:   window,
	}
	// Start cleanup goroutine
	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, rc := range rl.requests {
			if now.After(rc.resetAt) && !rc.lockout {
				delete(rl.requests, ip)
			}
			if rc.lockout && now.After(rc.lockoutUntil) {
				delete(rl.requests, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// Allow checks if a request is allowed
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	rc, exists := rl.requests[ip]

	if !exists {
		rl.requests[ip] = &requestCount{
			count:   1,
			resetAt: now.Add(rl.window),
		}
		return true
	}

	// Check lockout
	if rc.lockout {
		if now.Before(rc.lockoutUntil) {
			return false
		}
		// Lockout expired, reset
		rc.lockout = false
		rc.count = 1
		rc.resetAt = now.Add(rl.window)
		return true
	}

	// Check if window expired
	if now.After(rc.resetAt) {
		rc.count = 1
		rc.resetAt = now.Add(rl.window)
		return true
	}

	rc.count++
	if rc.count > rl.limit {
		// Apply lockout after too many attempts
		rc.lockout = true
		rc.lockoutUntil = now.Add(15 * time.Minute) // 15 minute lockout
		return false
	}

	return true
}

// GetRemainingAttempts returns remaining attempts for an IP
func (rl *RateLimiter) GetRemainingAttempts(ip string) int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	rc, exists := rl.requests[ip]
	if !exists {
		return rl.limit
	}
	if rc.lockout {
		return 0
	}
	remaining := rl.limit - rc.count
	if remaining < 0 {
		return 0
	}
	return remaining
}

// Global rate limiters
var (
	authRateLimiter *RateLimiter
	apiRateLimiter  *RateLimiter
)

func init() {
	// 5 auth attempts per minute, lockout for 15 minutes after
	authRateLimiter = NewRateLimiter(5, time.Minute)
	// 100 API requests per minute
	apiRateLimiter = NewRateLimiter(100, time.Minute)
}

// Setup configures all middleware for the Fiber app
func Setup(app *fiber.App, cfg *config.Config) {
	// Request logging
	app.Use(Logger())

	// Security headers
	app.Use(SecurityHeaders())

	// CORS
	app.Use(CORS(cfg.Server.AllowOrigins))

	// Recovery from panics
	app.Use(Recovery())

	// API rate limiting (applied to all routes)
	app.Use(APIRateLimit())
}

// SecurityHeaders adds security-related HTTP headers
func SecurityHeaders() fiber.Handler {
	return func(c fiber.Ctx) error {
		// Prevent MIME type sniffing
		c.Set("X-Content-Type-Options", "nosniff")
		// Enable XSS protection
		c.Set("X-XSS-Protection", "1; mode=block")
		// Prevent clickjacking
		c.Set("X-Frame-Options", "DENY")
		// Referrer policy
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		// Content Security Policy for API
		c.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		// Strict Transport Security (only in production)
		c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		return c.Next()
	}
}

// AuthRateLimit rate limits authentication endpoints
func AuthRateLimit() fiber.Handler {
	return func(c fiber.Ctx) error {
		ip := c.IP()
		if !authRateLimiter.Allow(ip) {
			remaining := authRateLimiter.GetRemainingAttempts(ip)
			c.Set("X-RateLimit-Remaining", "0")
			c.Set("Retry-After", "900") // 15 minutes
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":     "Too many authentication attempts. Please try again later.",
				"remaining": remaining,
			})
		}
		c.Set("X-RateLimit-Remaining", string(rune('0'+authRateLimiter.GetRemainingAttempts(ip))))
		return c.Next()
	}
}

// APIRateLimit rate limits general API access
func APIRateLimit() fiber.Handler {
	return func(c fiber.Ctx) error {
		ip := c.IP()
		if !apiRateLimiter.Allow(ip) {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Too many requests. Please slow down.",
			})
		}
		return c.Next()
	}
}

// Logger logs all requests
func Logger() fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()

		err := c.Next()

		log.Printf(
			"%s %s %d %s",
			c.Method(),
			c.Path(),
			c.Response().StatusCode(),
			time.Since(start),
		)

		return err
	}
}

// CORS handles Cross-Origin Resource Sharing
func CORS(allowOrigins string) fiber.Handler {
	return func(c fiber.Ctx) error {
		c.Set("Access-Control-Allow-Origin", allowOrigins)
		c.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		c.Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-API-Key")
		c.Set("Access-Control-Allow-Credentials", "true")

		if c.Method() == "OPTIONS" {
			return c.SendStatus(fiber.StatusNoContent)
		}

		return c.Next()
	}
}

// Recovery recovers from panics
func Recovery() fiber.Handler {
	return func(c fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Panic recovered: %v", r)
				c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Internal server error",
				})
			}
		}()
		return c.Next()
	}
}

// JWTClaims represents the JWT token claims
type JWTClaims struct {
	UserID         string `json:"user_id"`
	OrganizationID string `json:"organization_id"`
	Email          string `json:"email"`
	Role           string `json:"role"`
	jwt.RegisteredClaims
}

// JWTAuth validates JWT tokens for protected routes
func JWTAuth(secret string) fiber.Handler {
	return func(c fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing authorization header",
			})
		}

		// Extract token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid authorization header format",
			})
		}

		tokenString := parts[1]

		// Parse and validate token
		token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid or expired token",
			})
		}

		claims, ok := token.Claims.(*JWTClaims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid token claims",
			})
		}

		// Parse UUIDs
		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid user ID in token",
			})
		}

		orgID, err := uuid.Parse(claims.OrganizationID)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid organization ID in token",
			})
		}

		// Store user info in context
		c.Locals("user_id", userID)
		c.Locals("organization_id", orgID)
		c.Locals("email", claims.Email)
		c.Locals("role", claims.Role)

		return c.Next()
	}
}

// APIKeyAuth validates API keys for agent authentication
func APIKeyAuth() fiber.Handler {
	return func(c fiber.Ctx) error {
		apiKey := c.Get("X-API-Key")
		if apiKey == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing API key",
			})
		}

		// API key validation is done in the handler
		// Store it in context for handler to validate
		c.Locals("api_key", apiKey)
		c.Locals("agent_id", c.Get("X-Agent-ID"))
		c.Locals("hostname", c.Get("X-Hostname"))

		return c.Next()
	}
}

// RequireRole ensures the user has the required role
func RequireRole(roles ...string) fiber.Handler {
	return func(c fiber.Ctx) error {
		userRole := c.Locals("role").(string)

		for _, role := range roles {
			if userRole == role {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Insufficient permissions",
		})
	}
}
