package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// FileConfig represents the structure of configuration files (JSON/YAML)
type FileConfig struct {
	Server struct {
		Port    int    `json:"port" yaml:"port"`
		Timeout string `json:"timeout" yaml:"timeout"`
	} `json:"server" yaml:"server"`

	Database struct {
		URL         string `json:"url" yaml:"url"`
		MaxConns    int    `json:"max_conns" yaml:"max_conns"`
		MinConns    int    `json:"min_conns" yaml:"min_conns"`
		MaxLifetime string `json:"max_lifetime" yaml:"max_lifetime"`
		MaxIdleTime string `json:"max_idle_time" yaml:"max_idle_time"`
	} `json:"database" yaml:"database"`

	Redis struct {
		URL             string `json:"url" yaml:"url"`
		MaxConns        int    `json:"max_conns" yaml:"max_conns"`
		MinIdleConns    int    `json:"min_idle_conns" yaml:"min_idle_conns"`
		ConnMaxIdleTime string `json:"conn_max_idle_time" yaml:"conn_max_idle_time"`
	} `json:"redis" yaml:"redis"`

	Service struct {
		ShortDomain string `json:"short_domain" yaml:"short_domain"`
		WorkerID    int    `json:"worker_id" yaml:"worker_id"`
		Environment string `json:"environment" yaml:"environment"`
	} `json:"service" yaml:"service"`

	RateLimit struct {
		PerIP      int    `json:"per_ip" yaml:"per_ip"`
		PerIPBurst int    `json:"per_ip_burst" yaml:"per_ip_burst"`
		Window     string `json:"window" yaml:"window"`
	} `json:"rate_limit" yaml:"rate_limit"`

	Logging struct {
		Level string `json:"level" yaml:"level"`
	} `json:"logging" yaml:"logging"`

	GCP struct {
		Project string `json:"project" yaml:"project"`
		Region  string `json:"region" yaml:"region"`
	} `json:"gcp" yaml:"gcp"`
}

// LoadFromFile loads configuration from a file (JSON or YAML) and merges with environment variables
func LoadFromFile(path string) (*Config, error) {
	// Check if file exists first
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", path)
	}

	// Start with base config with defaults (don't validate yet)
	config := &Config{
		// Server defaults
		Port:    getEnvAsInt("PORT", 8080),
		Timeout: getEnvAsDuration("SERVER_TIMEOUT", 10*time.Second),

		// Database defaults
		DatabaseURL:         getEnv("DATABASE_URL", ""),
		DatabaseMaxConns:    getEnvAsInt("DATABASE_MAX_CONNS", 20),
		DatabaseMinConns:    getEnvAsInt("DATABASE_MIN_CONNS", 5),
		DatabaseMaxLifetime: getEnvAsDuration("DATABASE_MAX_LIFETIME", time.Hour),
		DatabaseMaxIdleTime: getEnvAsDuration("DATABASE_MAX_IDLE_TIME", 30*time.Minute),

		// Redis defaults
		RedisURL:             getEnv("REDIS_URL", ""),
		RedisMaxConns:        getEnvAsInt("REDIS_MAX_CONNS", 10),
		RedisMinIdleConns:    getEnvAsInt("REDIS_MIN_IDLE_CONNS", 2),
		RedisConnMaxIdleTime: getEnvAsDuration("REDIS_CONN_MAX_IDLE_TIME", 5*time.Minute),

		// Service defaults
		ShortDomain: getEnv("SHORT_DOMAIN", ""),
		WorkerID:    getEnvAsInt("WORKER_ID", -1),
		Environment: getEnv("ENVIRONMENT", "development"),

		// Rate limiting defaults
		RateLimitPerIP:      getEnvAsInt("RATE_LIMIT_PER_IP", 100),
		RateLimitPerIPBurst: getEnvAsInt("RATE_LIMIT_PER_IP_BURST", 120),
		RateLimitWindow:     getEnvAsDuration("RATE_LIMIT_WINDOW", time.Minute),

		// Logging defaults
		LogLevel: getEnv("LOG_LEVEL", "info"),

		// GCP configuration (optional)
		GCPProject: getEnv("GCP_PROJECT", ""),
		GCPRegion:  getEnv("GCP_REGION", "us-central1"),
	}

	// Read file content
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse based on file extension
	var fileConfig FileConfig
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".json":
		if err := json.Unmarshal(data, &fileConfig); err != nil {
			return nil, fmt.Errorf("failed to parse JSON config: %w", err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &fileConfig); err != nil {
			return nil, fmt.Errorf("failed to parse YAML config: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported config file format: %s (use .json, .yaml, or .yml)", ext)
	}

	// Merge file config with loaded config (file takes precedence over defaults, env vars take precedence over file)
	if err := mergeFileConfig(config, &fileConfig); err != nil {
		return nil, fmt.Errorf("failed to merge config: %w", err)
	}

	// Validate final configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// mergeFileConfig merges file configuration into the main config
// Environment variables take precedence over file values
func mergeFileConfig(config *Config, fileConfig *FileConfig) error {
	// Server configuration
	if fileConfig.Server.Port > 0 && os.Getenv("PORT") == "" {
		config.Port = fileConfig.Server.Port
	}
	if fileConfig.Server.Timeout != "" && os.Getenv("SERVER_TIMEOUT") == "" {
		timeout, err := time.ParseDuration(fileConfig.Server.Timeout)
		if err != nil {
			return fmt.Errorf("invalid server timeout: %w", err)
		}
		config.Timeout = timeout
	}

	// Database configuration
	if fileConfig.Database.URL != "" && os.Getenv("DATABASE_URL") == "" {
		config.DatabaseURL = fileConfig.Database.URL
	}
	if fileConfig.Database.MaxConns > 0 && os.Getenv("DATABASE_MAX_CONNS") == "" {
		config.DatabaseMaxConns = fileConfig.Database.MaxConns
	}
	if fileConfig.Database.MinConns > 0 && os.Getenv("DATABASE_MIN_CONNS") == "" {
		config.DatabaseMinConns = fileConfig.Database.MinConns
	}
	if fileConfig.Database.MaxLifetime != "" && os.Getenv("DATABASE_MAX_LIFETIME") == "" {
		lifetime, err := time.ParseDuration(fileConfig.Database.MaxLifetime)
		if err != nil {
			return fmt.Errorf("invalid database max lifetime: %w", err)
		}
		config.DatabaseMaxLifetime = lifetime
	}
	if fileConfig.Database.MaxIdleTime != "" && os.Getenv("DATABASE_MAX_IDLE_TIME") == "" {
		idleTime, err := time.ParseDuration(fileConfig.Database.MaxIdleTime)
		if err != nil {
			return fmt.Errorf("invalid database max idle time: %w", err)
		}
		config.DatabaseMaxIdleTime = idleTime
	}

	// Redis configuration
	if fileConfig.Redis.URL != "" && os.Getenv("REDIS_URL") == "" {
		config.RedisURL = fileConfig.Redis.URL
	}
	if fileConfig.Redis.MaxConns > 0 && os.Getenv("REDIS_MAX_CONNS") == "" {
		config.RedisMaxConns = fileConfig.Redis.MaxConns
	}
	if fileConfig.Redis.MinIdleConns > 0 && os.Getenv("REDIS_MIN_IDLE_CONNS") == "" {
		config.RedisMinIdleConns = fileConfig.Redis.MinIdleConns
	}
	if fileConfig.Redis.ConnMaxIdleTime != "" && os.Getenv("REDIS_CONN_MAX_IDLE_TIME") == "" {
		idleTime, err := time.ParseDuration(fileConfig.Redis.ConnMaxIdleTime)
		if err != nil {
			return fmt.Errorf("invalid redis conn max idle time: %w", err)
		}
		config.RedisConnMaxIdleTime = idleTime
	}

	// Service configuration
	if fileConfig.Service.ShortDomain != "" && os.Getenv("SHORT_DOMAIN") == "" {
		config.ShortDomain = fileConfig.Service.ShortDomain
	}
	if fileConfig.Service.WorkerID >= 0 && os.Getenv("WORKER_ID") == "" {
		config.WorkerID = fileConfig.Service.WorkerID
	}
	if fileConfig.Service.Environment != "" && os.Getenv("ENVIRONMENT") == "" {
		config.Environment = fileConfig.Service.Environment
	}

	// Rate limit configuration
	if fileConfig.RateLimit.PerIP > 0 && os.Getenv("RATE_LIMIT_PER_IP") == "" {
		config.RateLimitPerIP = fileConfig.RateLimit.PerIP
	}
	if fileConfig.RateLimit.PerIPBurst > 0 && os.Getenv("RATE_LIMIT_PER_IP_BURST") == "" {
		config.RateLimitPerIPBurst = fileConfig.RateLimit.PerIPBurst
	}
	if fileConfig.RateLimit.Window != "" && os.Getenv("RATE_LIMIT_WINDOW") == "" {
		window, err := time.ParseDuration(fileConfig.RateLimit.Window)
		if err != nil {
			return fmt.Errorf("invalid rate limit window: %w", err)
		}
		config.RateLimitWindow = window
	}

	// Logging configuration
	if fileConfig.Logging.Level != "" && os.Getenv("LOG_LEVEL") == "" {
		config.LogLevel = fileConfig.Logging.Level
	}

	// GCP configuration
	if fileConfig.GCP.Project != "" && os.Getenv("GCP_PROJECT") == "" {
		config.GCPProject = fileConfig.GCP.Project
	}
	if fileConfig.GCP.Region != "" && os.Getenv("GCP_REGION") == "" {
		config.GCPRegion = fileConfig.GCP.Region
	}

	return nil
}

// LoadWithFileOverride loads configuration with file override support
// Priority: Environment Variables > Config File > Defaults
func LoadWithFileOverride(path string) (*Config, error) {
	// If path is empty, just load from environment
	if path == "" {
		return Load()
	}

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// File doesn't exist, fall back to environment only
		return Load()
	}

	// Load from file
	return LoadFromFile(path)
}
