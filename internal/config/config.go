package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	// ErrInvalidPort indicates that the port configuration is invalid
	ErrInvalidPort = errors.New("invalid port: must be between 1 and 65535")
	// ErrInvalidDatabaseURL indicates that the database URL is missing or invalid
	ErrInvalidDatabaseURL = errors.New("database URL is required")
	// ErrInvalidRedisURL indicates that the redis URL is missing or invalid
	ErrInvalidRedisURL = errors.New("redis URL is required")
	// ErrInvalidShortDomain indicates that the short domain is missing or invalid
	ErrInvalidShortDomain = errors.New("short domain is required")
	// ErrInvalidRateLimit indicates that the rate limit value is invalid
	ErrInvalidRateLimit = errors.New("invalid rate limit: must be positive")
	// ErrInvalidLogLevel indicates that the log level is not recognized
	ErrInvalidLogLevel = errors.New("invalid log level: must be one of debug, info, warn, error")
)

// Config holds all configuration for the shortening service
type Config struct {
	// Server configuration
	Port    int
	Timeout time.Duration

	// Database configuration
	DatabaseURL         string
	DatabaseMaxConns    int
	DatabaseMinConns    int
	DatabaseMaxLifetime time.Duration
	DatabaseMaxIdleTime time.Duration

	// Redis configuration
	RedisURL            string
	RedisMaxConns       int
	RedisMinIdleConns   int
	RedisConnMaxIdleTime time.Duration

	// Service configuration
	ShortDomain string
	WorkerID    int
	Environment string

	// Rate limiting
	RateLimitPerIP     int
	RateLimitPerIPBurst int
	RateLimitWindow    time.Duration

	// Logging
	LogLevel string

	// GCP configuration (optional, for cloud deployments)
	GCPProject string
	GCPRegion  string
}

// Load loads configuration from environment variables with sensible defaults
func Load() (*Config, error) {
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
		WorkerID:    getEnvAsInt("WORKER_ID", -1), // -1 means auto-detect
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

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// Validate checks that the configuration is valid
func (c *Config) Validate() error {
	// Validate port
	if c.Port < 1 || c.Port > 65535 {
		return ErrInvalidPort
	}

	// Validate database URL
	if c.DatabaseURL == "" {
		return ErrInvalidDatabaseURL
	}

	// Validate Redis URL
	if c.RedisURL == "" {
		return ErrInvalidRedisURL
	}

	// Validate short domain
	if c.ShortDomain == "" {
		return ErrInvalidShortDomain
	}

	// Validate rate limit
	if c.RateLimitPerIP <= 0 {
		return ErrInvalidRateLimit
	}

	// Validate log level
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[strings.ToLower(c.LogLevel)] {
		return ErrInvalidLogLevel
	}

	return nil
}

// IsDevelopment returns true if running in development environment
func (c *Config) IsDevelopment() bool {
	return strings.ToLower(c.Environment) == "development"
}

// IsProduction returns true if running in production environment
func (c *Config) IsProduction() bool {
	return strings.ToLower(c.Environment) == "production"
}

// GetWorkerID returns the worker ID, auto-detecting if necessary
func (c *Config) GetWorkerID() int {
	if c.WorkerID >= 0 {
		return c.WorkerID
	}

	// Auto-detect from Cloud Run instance ID
	instanceID := os.Getenv("CLOUD_RUN_INSTANCE_ID")
	if instanceID != "" {
		// Extract numeric portion from instance ID and use last 10 bits
		hash := hashString(instanceID)
		return hash & 0x3FF // 10 bits = 1024 possible values
	}

	// Fallback to hostname-based worker ID
	hostname, err := os.Hostname()
	if err != nil {
		return 0 // Default worker ID
	}

	hash := hashString(hostname)
	return hash & 0x3FF
}

// hashString creates a simple hash of a string for worker ID generation
func hashString(s string) int {
	hash := 0
	for i := 0; i < len(s); i++ {
		hash = 31*hash + int(s[i])
	}
	if hash < 0 {
		hash = -hash
	}
	return hash
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getEnvAsInt retrieves an environment variable as an integer or returns a default value
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}

// getEnvAsDuration retrieves an environment variable as a duration or returns a default value
func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := time.ParseDuration(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}
