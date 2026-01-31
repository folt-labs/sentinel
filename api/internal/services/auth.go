package services

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"github.com/folt-labs/sentinel/api/internal/config"
	"github.com/folt-labs/sentinel/api/internal/database"
	"github.com/folt-labs/sentinel/api/internal/models"
)

var (
	ErrEmailExists        = errors.New("email already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
	ErrWeakPassword       = errors.New("password must be at least 8 characters and contain uppercase, lowercase, and a number")
	ErrInvalidEmail       = errors.New("invalid email address")
)

// ValidatePassword checks password complexity requirements
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return ErrWeakPassword
	}

	var hasUpper, hasLower, hasNumber bool
	for _, c := range password {
		switch {
		case c >= 'A' && c <= 'Z':
			hasUpper = true
		case c >= 'a' && c <= 'z':
			hasLower = true
		case c >= '0' && c <= '9':
			hasNumber = true
		}
	}

	if !hasUpper || !hasLower || !hasNumber {
		return ErrWeakPassword
	}

	return nil
}

// ValidateEmail checks if email format is valid
func ValidateEmail(email string) error {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return ErrInvalidEmail
	}
	return nil
}

// AuthService handles authentication operations
type AuthService struct {
	db  *database.DB
	cfg *config.Config
}

// NewAuthService creates a new auth service
func NewAuthService(db *database.DB, cfg *config.Config) *AuthService {
	return &AuthService{db: db, cfg: cfg}
}

// RegisterInput represents registration input
type RegisterInput struct {
	Email            string
	Password         string
	Name             string
	OrganizationName string
}

// AuthResult represents authentication result
type AuthResult struct {
	Token        string            `json:"token"`
	User         *models.User      `json:"user"`
	Organization *models.Organization `json:"organization"`
}

// Register creates a new user and organization
func (s *AuthService) Register(ctx context.Context, input RegisterInput) (*AuthResult, error) {
	// Validate email format
	if err := ValidateEmail(input.Email); err != nil {
		return nil, err
	}

	// Validate password complexity
	if err := ValidatePassword(input.Password); err != nil {
		return nil, err
	}

	// Check if email exists
	var count int
	err := s.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE email = $1", input.Email).Scan(&count)
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, ErrEmailExists
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Create organization
	orgID := uuid.New()
	slug := generateSlug(input.OrganizationName)
	now := time.Now()

	_, err = s.db.Pool.Exec(ctx,
		`INSERT INTO organizations (id, name, slug, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $4)`,
		orgID, input.OrganizationName, slug, now)
	if err != nil {
		return nil, err
	}

	// Create user
	userID := uuid.New()
	_, err = s.db.Pool.Exec(ctx,
		`INSERT INTO users (id, organization_id, email, password_hash, name, role, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $7)`,
		userID, orgID, input.Email, string(hashedPassword), input.Name, models.RoleOwner, now)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		ID:             userID,
		OrganizationID: orgID,
		Email:          input.Email,
		Name:           input.Name,
		Role:           models.RoleOwner,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	org := &models.Organization{
		ID:        orgID,
		Name:      input.OrganizationName,
		Slug:      slug,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Generate token
	token, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		Token:        token,
		User:         user,
		Organization: org,
	}, nil
}

// Login authenticates a user
func (s *AuthService) Login(ctx context.Context, email, password string) (*AuthResult, error) {
	var user models.User
	err := s.db.Pool.QueryRow(ctx,
		`SELECT id, organization_id, email, password_hash, name, role, created_at, updated_at
		 FROM users WHERE email = $1`, email).Scan(
		&user.ID, &user.OrganizationID, &user.Email, &user.PasswordHash,
		&user.Name, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Get organization
	var org models.Organization
	err = s.db.Pool.QueryRow(ctx,
		`SELECT id, name, slug, created_at, updated_at
		 FROM organizations WHERE id = $1`, user.OrganizationID).Scan(
		&org.ID, &org.Name, &org.Slug, &org.CreatedAt, &org.UpdatedAt)
	if err != nil {
		return nil, err
	}

	// Generate token
	token, err := s.generateToken(&user)
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		Token:        token,
		User:         &user,
		Organization: &org,
	}, nil
}

// GetUserByID retrieves a user by ID
func (s *AuthService) GetUserByID(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	var user models.User
	err := s.db.Pool.QueryRow(ctx,
		`SELECT id, organization_id, email, name, role, created_at, updated_at
		 FROM users WHERE id = $1`, userID).Scan(
		&user.ID, &user.OrganizationID, &user.Email,
		&user.Name, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, ErrUserNotFound
	}

	return &user, nil
}

// UpdateUserByID updates a user's profile
func (s *AuthService) UpdateUserByID(ctx context.Context, userID uuid.UUID, name string) (*models.User, error) {
	now := time.Now()
	_, err := s.db.Pool.Exec(ctx,
		`UPDATE users SET name = $1, updated_at = $2 WHERE id = $3`,
		name, now, userID)
	if err != nil {
		return nil, err
	}

	return s.GetUserByID(ctx, userID)
}

// GetUser retrieves a user by ID string (for backward compatibility)
func (s *AuthService) GetUser(ctx context.Context, userID string) (*models.User, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return s.GetUserByID(ctx, id)
}

// UpdateUser updates a user's profile (for backward compatibility)
func (s *AuthService) UpdateUser(ctx context.Context, userID, name string) (*models.User, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return s.UpdateUserByID(ctx, id, name)
}

func (s *AuthService) generateToken(user *models.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id":         user.ID.String(),
		"organization_id": user.OrganizationID.String(),
		"email":           user.Email,
		"role":            user.Role,
		"exp":             time.Now().Add(time.Hour * time.Duration(s.cfg.JWT.ExpirationHours)).Unix(),
		"iat":             time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWT.Secret))
}

func generateSlug(name string) string {
	slug := strings.ToLower(name)
	reg := regexp.MustCompile("[^a-z0-9]+")
	slug = reg.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	return slug
}
