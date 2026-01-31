package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/folt-labs/sentinel/api/internal/database"
	"github.com/folt-labs/sentinel/api/internal/models"
)

var (
	ErrServerNotFound = errors.New("server not found")
	ErrInvalidAPIKey  = errors.New("invalid API key")
)

// ServerService handles server operations
type ServerService struct {
	db *database.DB
}

// NewServerService creates a new server service
func NewServerService(db *database.DB) *ServerService {
	return &ServerService{db: db}
}

// CreateServerResult contains the created server and its API key
type CreateServerResult struct {
	Server *models.Server `json:"server"`
	APIKey string         `json:"api_key"`
}

// List returns all servers for an organization
func (s *ServerService) List(ctx context.Context, orgID uuid.UUID) ([]models.Server, error) {
	rows, err := s.db.Pool.Query(ctx,
		`SELECT id, organization_id, hostname, COALESCE(ip_address::text, ''), status, last_seen, agent_version, created_at, updated_at
		 FROM servers WHERE organization_id = $1 ORDER BY hostname`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var servers []models.Server
	for rows.Next() {
		var server models.Server
		err := rows.Scan(
			&server.ID, &server.OrganizationID, &server.Hostname, &server.IPAddress,
			&server.Status, &server.LastSeen, &server.AgentVersion, &server.CreatedAt, &server.UpdatedAt)
		if err != nil {
			return nil, err
		}
		servers = append(servers, server)
	}

	return servers, nil
}

// Get returns a specific server
func (s *ServerService) Get(ctx context.Context, orgID, serverID uuid.UUID) (*models.Server, error) {
	var server models.Server
	err := s.db.Pool.QueryRow(ctx,
		`SELECT id, organization_id, hostname, COALESCE(ip_address::text, ''), status, last_seen, agent_version, created_at, updated_at
		 FROM servers WHERE id = $1 AND organization_id = $2`, serverID, orgID).Scan(
		&server.ID, &server.OrganizationID, &server.Hostname, &server.IPAddress,
		&server.Status, &server.LastSeen, &server.AgentVersion, &server.CreatedAt, &server.UpdatedAt)
	if err != nil {
		return nil, ErrServerNotFound
	}
	return &server, nil
}

// Create creates a new server with an API key
func (s *ServerService) Create(ctx context.Context, orgID uuid.UUID, hostname, ipAddress string) (*CreateServerResult, error) {
	serverID := uuid.New()
	apiKey := generateAPIKey()
	now := time.Now()

	// Convert empty IP to nil for PostgreSQL INET type
	var ipParam interface{}
	if ipAddress != "" {
		ipParam = ipAddress
	}

	_, err := s.db.Pool.Exec(ctx,
		`INSERT INTO servers (id, organization_id, hostname, ip_address, status, api_key, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $7)`,
		serverID, orgID, hostname, ipParam, models.StatusOffline, apiKey, now)
	if err != nil {
		return nil, err
	}

	server := &models.Server{
		ID:             serverID,
		OrganizationID: orgID,
		Hostname:       hostname,
		IPAddress:      ipAddress,
		Status:         models.StatusOffline,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	return &CreateServerResult{
		Server: server,
		APIKey: apiKey,
	}, nil
}

// Delete removes a server
func (s *ServerService) Delete(ctx context.Context, orgID, serverID uuid.UUID) error {
	result, err := s.db.Pool.Exec(ctx,
		`DELETE FROM servers WHERE id = $1 AND organization_id = $2`, serverID, orgID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrServerNotFound
	}

	return nil
}

// ValidateAPIKey validates an API key and returns the server
func (s *ServerService) ValidateAPIKey(ctx context.Context, apiKey string) (*models.Server, error) {
	var server models.Server
	err := s.db.Pool.QueryRow(ctx,
		`SELECT id, organization_id, hostname, COALESCE(ip_address::text, ''), status, last_seen, agent_version, created_at, updated_at
		 FROM servers WHERE api_key = $1`, apiKey).Scan(
		&server.ID, &server.OrganizationID, &server.Hostname, &server.IPAddress,
		&server.Status, &server.LastSeen, &server.AgentVersion, &server.CreatedAt, &server.UpdatedAt)
	if err != nil {
		return nil, ErrInvalidAPIKey
	}
	return &server, nil
}

// UpdateAgentInfo updates server info from agent registration
func (s *ServerService) UpdateAgentInfo(ctx context.Context, serverID uuid.UUID, hostname, ipAddress, version string) error {
	now := time.Now()

	// Convert empty IP to nil for PostgreSQL INET type
	var ipParam interface{}
	if ipAddress != "" {
		ipParam = ipAddress
	}

	_, err := s.db.Pool.Exec(ctx,
		`UPDATE servers SET hostname = $1, ip_address = $2, agent_version = $3, status = $4, last_seen = $5, updated_at = $5
		 WHERE id = $6`,
		hostname, ipParam, version, models.StatusOnline, now, serverID)
	return err
}

// UpdateLastSeen updates the server's last seen timestamp
func (s *ServerService) UpdateLastSeen(ctx context.Context, serverID uuid.UUID) error {
	now := time.Now()
	_, err := s.db.Pool.Exec(ctx,
		`UPDATE servers SET last_seen = $1, status = $2, updated_at = $1 WHERE id = $3`,
		now, models.StatusOnline, serverID)
	return err
}

// UpdateStatus updates the server's status
func (s *ServerService) UpdateStatus(ctx context.Context, serverID uuid.UUID, status string) error {
	now := time.Now()
	_, err := s.db.Pool.Exec(ctx,
		`UPDATE servers SET status = $1, updated_at = $2 WHERE id = $3`,
		status, now, serverID)
	return err
}

func generateAPIKey() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return "sg_" + hex.EncodeToString(bytes)
}
