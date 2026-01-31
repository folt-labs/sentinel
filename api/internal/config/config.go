package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// Config holds all API configuration
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	SMTP     SMTPConfig     `mapstructure:"smtp"`
}

// ServerConfig holds HTTP server settings
type ServerConfig struct {
	Port         int    `mapstructure:"port"`
	Host         string `mapstructure:"host"`
	Environment  string `mapstructure:"environment"`
	AllowOrigins string `mapstructure:"allow_origins"`
}

// DatabaseConfig holds PostgreSQL connection settings
type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"name"`
	SSLMode  string `mapstructure:"ssl_mode"`
}

// ConnectionString returns the PostgreSQL connection string
func (d DatabaseConfig) ConnectionString() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode,
	)
}

// RedisConfig holds Redis connection settings
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// Address returns the Redis address
func (r RedisConfig) Address() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

// JWTConfig holds JWT settings
type JWTConfig struct {
	Secret          string `mapstructure:"secret"`
	ExpirationHours int    `mapstructure:"expiration_hours"`
}

// SMTPConfig holds email settings
type SMTPConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	From     string `mapstructure:"from"`
}

// Load reads configuration from file and environment
func Load(path string) (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Read config file if exists
	if path != "" {
		v.SetConfigFile(path)
		if err := v.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return nil, fmt.Errorf("error reading config file: %w", err)
			}
		}
	}

	// Environment variables
	v.SetEnvPrefix("SERVERGUARD")
	v.AutomaticEnv()

	// Override with environment variables
	bindEnvVars(v)

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.environment", "development")
	v.SetDefault("server.allow_origins", "*")

	// Database defaults
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.user", "serverguard")
	v.SetDefault("database.password", "serverguard")
	v.SetDefault("database.name", "serverguard")
	v.SetDefault("database.ssl_mode", "disable")

	// Redis defaults
	v.SetDefault("redis.host", "localhost")
	v.SetDefault("redis.port", 6379)
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)

	// JWT defaults
	v.SetDefault("jwt.expiration_hours", 24)

	// SMTP defaults (Mailhog for dev)
	v.SetDefault("smtp.host", "localhost")
	v.SetDefault("smtp.port", 1025)
	v.SetDefault("smtp.from", "noreply@serverguard.local")
}

func bindEnvVars(v *viper.Viper) {
	// Database
	if val := os.Getenv("DATABASE_HOST"); val != "" {
		v.Set("database.host", val)
	}
	if val := os.Getenv("DATABASE_PORT"); val != "" {
		v.Set("database.port", val)
	}
	if val := os.Getenv("DATABASE_USER"); val != "" {
		v.Set("database.user", val)
	}
	if val := os.Getenv("DATABASE_PASSWORD"); val != "" {
		v.Set("database.password", val)
	}
	if val := os.Getenv("DATABASE_NAME"); val != "" {
		v.Set("database.name", val)
	}

	// Redis
	if val := os.Getenv("REDIS_HOST"); val != "" {
		v.Set("redis.host", val)
	}
	if val := os.Getenv("REDIS_PASSWORD"); val != "" {
		v.Set("redis.password", val)
	}

	// JWT
	if val := os.Getenv("JWT_SECRET"); val != "" {
		v.Set("jwt.secret", val)
	}
}
