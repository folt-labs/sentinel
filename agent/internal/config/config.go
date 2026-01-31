package config

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

// Config holds all agent configuration
type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Agent      AgentConfig      `mapstructure:"agent"`
	Collectors CollectorConfig  `mapstructure:"collectors"`
	Transport  TransportConfig  `mapstructure:"transport"`
}

// ServerConfig holds the API server connection details
type ServerConfig struct {
	URL    string `mapstructure:"url"`
	APIKey string `mapstructure:"api_key"`
}

// AgentConfig holds agent-specific settings
type AgentConfig struct {
	ID           string        `mapstructure:"id"`
	Hostname     string        `mapstructure:"hostname"`
	CollectEvery time.Duration `mapstructure:"collect_every"`
	SendEvery    time.Duration `mapstructure:"send_every"`
	QueueDir     string        `mapstructure:"queue_dir"`
}

// CollectorConfig holds settings for each collector
type CollectorConfig struct {
	SSH struct {
		Enabled  bool     `mapstructure:"enabled"`
		LogFiles []string `mapstructure:"log_files"`
	} `mapstructure:"ssh"`

	FileIntegrity struct {
		Enabled bool     `mapstructure:"enabled"`
		Paths   []string `mapstructure:"paths"`
	} `mapstructure:"file_integrity"`

	Ports struct {
		Enabled bool `mapstructure:"enabled"`
	} `mapstructure:"ports"`

	Resources struct {
		Enabled bool `mapstructure:"enabled"`
	} `mapstructure:"resources"`
}

// TransportConfig holds HTTP client settings
type TransportConfig struct {
	Timeout       time.Duration `mapstructure:"timeout"`
	RetryAttempts int           `mapstructure:"retry_attempts"`
	RetryDelay    time.Duration `mapstructure:"retry_delay"`
}

// Load reads configuration from file and environment
func Load(path string) (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Read config file if it exists
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

	// Override with env vars
	if url := os.Getenv("SERVERGUARD_SERVER_URL"); url != "" {
		v.Set("server.url", url)
	}
	if key := os.Getenv("SERVERGUARD_API_KEY"); key != "" {
		v.Set("server.api_key", key)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Set hostname if not configured
	if cfg.Agent.Hostname == "" {
		hostname, err := os.Hostname()
		if err != nil {
			return nil, fmt.Errorf("error getting hostname: %w", err)
		}
		cfg.Agent.Hostname = hostname
	}

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.url", "http://localhost:8080")

	// Agent defaults
	v.SetDefault("agent.collect_every", "30s")
	v.SetDefault("agent.send_every", "60s")
	v.SetDefault("agent.queue_dir", "/var/lib/serverguard/queue")

	// SSH collector defaults
	v.SetDefault("collectors.ssh.enabled", true)
	v.SetDefault("collectors.ssh.log_files", []string{
		"/var/log/auth.log",
		"/var/log/secure",
	})

	// File integrity defaults
	v.SetDefault("collectors.file_integrity.enabled", true)
	v.SetDefault("collectors.file_integrity.paths", []string{
		"/etc/passwd",
		"/etc/shadow",
		"/etc/sudoers",
		"/etc/ssh/sshd_config",
	})

	// Ports defaults
	v.SetDefault("collectors.ports.enabled", true)

	// Resources defaults
	v.SetDefault("collectors.resources.enabled", true)

	// Transport defaults
	v.SetDefault("transport.timeout", "30s")
	v.SetDefault("transport.retry_attempts", 3)
	v.SetDefault("transport.retry_delay", "5s")
}
