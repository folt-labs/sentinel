package services

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/folt-labs/sentinel/api/internal/config"
	"github.com/folt-labs/sentinel/api/internal/database"
	"github.com/folt-labs/sentinel/api/internal/models"
)

// NotificationService handles sending notifications
type NotificationService struct {
	db         *database.DB
	smtpConfig config.SMTPConfig
	httpClient *http.Client
}

// NewNotificationService creates a new notification service
func NewNotificationService(db *database.DB, cfg *config.Config) *NotificationService {
	return &NotificationService{
		db:         db,
		smtpConfig: cfg.SMTP,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// AlertPayload is the structure sent to webhooks
type AlertPayload struct {
	AlertID     string                 `json:"alert_id"`
	ServerID    string                 `json:"server_id"`
	Hostname    string                 `json:"hostname"`
	Severity    string                 `json:"severity"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	TriggeredAt time.Time              `json:"triggered_at"`
	Data        map[string]interface{} `json:"data,omitempty"`
}

// NotifyAlert sends notifications for an alert to all enabled channels
func (s *NotificationService) NotifyAlert(ctx context.Context, alert *models.Alert, hostname string) error {
	channels, err := s.getEnabledChannels(ctx, alert.OrganizationID)
	if err != nil {
		return fmt.Errorf("failed to get notification channels: %w", err)
	}

	payload := AlertPayload{
		AlertID:     alert.ID.String(),
		ServerID:    alert.ServerID.String(),
		Hostname:    hostname,
		Severity:    alert.Severity,
		Title:       alert.Title,
		Description: alert.Description,
		TriggeredAt: alert.TriggeredAt,
	}

	var errs []string
	for _, channel := range channels {
		var err error
		switch channel.Type {
		case models.ChannelTypeEmail:
			err = s.sendEmail(channel, payload)
		case models.ChannelTypeWebhook:
			err = s.sendWebhook(channel, payload)
		}
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", channel.Name, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("notification errors: %s", strings.Join(errs, "; "))
	}

	return nil
}

// NotifyServerOffline sends notifications when a server goes offline
func (s *NotificationService) NotifyServerOffline(ctx context.Context, orgID, serverID uuid.UUID, hostname string) error {
	alert := &models.Alert{
		ID:          uuid.New(),
		ServerID:    serverID,
		Severity:    models.SeverityHigh,
		Title:       "Server Offline",
		Description: fmt.Sprintf("Server '%s' has stopped reporting and is now offline.", hostname),
		TriggeredAt: time.Now(),
	}
	alert.OrganizationID = orgID

	return s.NotifyAlert(ctx, alert, hostname)
}

func (s *NotificationService) getEnabledChannels(ctx context.Context, orgID uuid.UUID) ([]models.NotificationChannel, error) {
	rows, err := s.db.Pool.Query(ctx,
		`SELECT id, organization_id, name, type, config, enabled, created_at, updated_at
		 FROM notification_channels
		 WHERE organization_id = $1 AND enabled = true`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []models.NotificationChannel
	for rows.Next() {
		var channel models.NotificationChannel
		if err := rows.Scan(
			&channel.ID, &channel.OrganizationID, &channel.Name, &channel.Type,
			&channel.Config, &channel.Enabled, &channel.CreatedAt, &channel.UpdatedAt,
		); err != nil {
			return nil, err
		}
		channels = append(channels, channel)
	}

	return channels, nil
}

func (s *NotificationService) sendEmail(channel models.NotificationChannel, payload AlertPayload) error {
	// Extract email addresses from config
	recipients, ok := channel.Config["recipients"].([]interface{})
	if !ok {
		// Try single email
		email, ok := channel.Config["email"].(string)
		if !ok {
			return fmt.Errorf("no recipients configured")
		}
		recipients = []interface{}{email}
	}

	var to []string
	for _, r := range recipients {
		if email, ok := r.(string); ok {
			to = append(to, email)
		}
	}

	if len(to) == 0 {
		return fmt.Errorf("no valid recipients")
	}

	// Build email content
	subject := fmt.Sprintf("[Sentinel] %s Alert: %s", strings.ToUpper(payload.Severity), payload.Title)
	body := fmt.Sprintf(`
Sentinel Security Alert

Severity: %s
Server: %s
Alert: %s

%s

Triggered at: %s

---
View details at your Sentinel dashboard.
`,
		strings.ToUpper(payload.Severity),
		payload.Hostname,
		payload.Title,
		payload.Description,
		payload.TriggeredAt.Format(time.RFC1123),
	)

	// Build message
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		s.smtpConfig.From,
		strings.Join(to, ", "),
		subject,
		body,
	)

	// Send email
	addr := fmt.Sprintf("%s:%d", s.smtpConfig.Host, s.smtpConfig.Port)
	var auth smtp.Auth
	if s.smtpConfig.User != "" {
		auth = smtp.PlainAuth("", s.smtpConfig.User, s.smtpConfig.Password, s.smtpConfig.Host)
	}

	err := smtp.SendMail(addr, auth, s.smtpConfig.From, to, []byte(msg))
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	log.Printf("Email sent to %v for alert %s", to, payload.Title)
	return nil
}

func (s *NotificationService) sendWebhook(channel models.NotificationChannel, payload AlertPayload) error {
	url, ok := channel.Config["url"].(string)
	if !ok {
		return fmt.Errorf("no webhook URL configured")
	}

	// Get optional secret for signing
	secret, _ := channel.Config["secret"].(string)

	// Marshal payload
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create request
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Sentinel-Webhook/1.0")

	// Add signature if secret is configured
	if secret != "" {
		signature := signPayload(body, secret)
		req.Header.Set("X-Sentinel-Signature", signature)
	}

	// Send request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	log.Printf("Webhook sent to %s for alert %s", url, payload.Title)
	return nil
}

// signPayload creates an HMAC-SHA256 signature
func signPayload(payload []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return "sha256=" + hex.EncodeToString(h.Sum(nil))
}

// TestChannel sends a test notification to a channel
func (s *NotificationService) TestChannel(ctx context.Context, channel models.NotificationChannel) error {
	payload := AlertPayload{
		AlertID:     "test-alert-id",
		ServerID:    "test-server-id",
		Hostname:    "test-server",
		Severity:    "info",
		Title:       "Test Notification",
		Description: "This is a test notification from Sentinel to verify your notification channel is working correctly.",
		TriggeredAt: time.Now(),
	}

	switch channel.Type {
	case models.ChannelTypeEmail:
		return s.sendEmail(channel, payload)
	case models.ChannelTypeWebhook:
		return s.sendWebhook(channel, payload)
	default:
		return fmt.Errorf("unknown channel type: %s", channel.Type)
	}
}
