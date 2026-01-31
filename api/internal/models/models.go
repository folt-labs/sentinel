package models

import (
	"time"

	"github.com/google/uuid"
)

// Organization represents a tenant
type Organization struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Slug      string    `json:"slug" db:"slug"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// User represents an authenticated user
type User struct {
	ID             uuid.UUID `json:"id" db:"id"`
	OrganizationID uuid.UUID `json:"organization_id" db:"organization_id"`
	Email          string    `json:"email" db:"email"`
	PasswordHash   string    `json:"-" db:"password_hash"`
	Name           string    `json:"name" db:"name"`
	Role           string    `json:"role" db:"role"` // owner, admin, member, viewer
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

// UserRole constants
const (
	RoleOwner  = "owner"
	RoleAdmin  = "admin"
	RoleMember = "member"
	RoleViewer = "viewer"
)

// Server represents a monitored server
type Server struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	OrganizationID uuid.UUID  `json:"organization_id" db:"organization_id"`
	Hostname       string     `json:"hostname" db:"hostname"`
	IPAddress      string     `json:"ip_address" db:"ip_address"`
	Status         string     `json:"status" db:"status"` // online, offline, warning, critical
	LastSeen       *time.Time `json:"last_seen" db:"last_seen"`
	AgentVersion   string     `json:"agent_version" db:"agent_version"`
	APIKey         string     `json:"-" db:"api_key"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}

// ServerStatus constants
const (
	StatusOnline   = "online"
	StatusOffline  = "offline"
	StatusWarning  = "warning"
	StatusCritical = "critical"
)

// SecurityEvent represents a collected security event
type SecurityEvent struct {
	ID        uuid.UUID              `json:"id" db:"id"`
	ServerID  uuid.UUID              `json:"server_id" db:"server_id"`
	EventType string                 `json:"event_type" db:"event_type"`
	Severity  string                 `json:"severity" db:"severity"` // info, low, medium, high, critical
	Timestamp time.Time              `json:"timestamp" db:"timestamp"`
	Data      map[string]interface{} `json:"data" db:"data"`
}

// Severity constants
const (
	SeverityInfo     = "info"
	SeverityLow      = "low"
	SeverityMedium   = "medium"
	SeverityHigh     = "high"
	SeverityCritical = "critical"
)

// Alert represents a triggered alert
type Alert struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	OrganizationID uuid.UUID  `json:"organization_id" db:"organization_id"`
	ServerID       uuid.UUID  `json:"server_id" db:"server_id"`
	Severity       string     `json:"severity" db:"severity"`
	Status         string     `json:"status" db:"status"` // open, acknowledged, resolved
	Title          string     `json:"title" db:"title"`
	Description    string     `json:"description" db:"description"`
	TriggeredAt    time.Time  `json:"triggered_at" db:"triggered_at"`
	AcknowledgedAt *time.Time `json:"acknowledged_at" db:"acknowledged_at"`
	ResolvedAt     *time.Time `json:"resolved_at" db:"resolved_at"`
	AcknowledgedBy *uuid.UUID `json:"acknowledged_by" db:"acknowledged_by"`
	ResolvedBy     *uuid.UUID `json:"resolved_by" db:"resolved_by"`
}

// AlertStatus constants
const (
	AlertStatusOpen         = "open"
	AlertStatusAcknowledged = "acknowledged"
	AlertStatusResolved     = "resolved"
)

// AlertRule defines when to trigger an alert
type AlertRule struct {
	ID             uuid.UUID              `json:"id" db:"id"`
	OrganizationID uuid.UUID              `json:"organization_id" db:"organization_id"`
	Name           string                 `json:"name" db:"name"`
	Description    string                 `json:"description" db:"description"`
	EventType      string                 `json:"event_type" db:"event_type"`
	Severity       string                 `json:"severity" db:"severity"`
	Conditions     map[string]interface{} `json:"conditions" db:"conditions"`
	Enabled        bool                   `json:"enabled" db:"enabled"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at" db:"updated_at"`
}

// NotificationChannel represents a notification destination
type NotificationChannel struct {
	ID             uuid.UUID              `json:"id" db:"id"`
	OrganizationID uuid.UUID              `json:"organization_id" db:"organization_id"`
	Name           string                 `json:"name" db:"name"`
	Type           string                 `json:"type" db:"type"` // email, webhook
	Config         map[string]interface{} `json:"config" db:"config"`
	Enabled        bool                   `json:"enabled" db:"enabled"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at" db:"updated_at"`
}

// ChannelType constants
const (
	ChannelTypeEmail   = "email"
	ChannelTypeWebhook = "webhook"
)
