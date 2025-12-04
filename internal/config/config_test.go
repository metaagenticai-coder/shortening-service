package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		wantErr bool
		errType error
		check   func(*testing.T, *Config)
	}{
		{
			name: "valid configuration with all required fields",
			envVars: map[string]string{
				"PORT":         "8080",
				"DATABASE_URL": "postgres://user:pass@localhost:5432/db",
				"REDIS_URL":    "redis://localhost:6379/0",
				"SHORT_DOMAIN": "short.link",
				"LOG_LEVEL":    "info",
			},
			wantErr: false,
			check: func(t *testing.T, c *Config) {
				if c.Port != 8080 {
					t.Errorf("expected port 8080, got %d", c.Port)
				}
				if c.DatabaseURL != "postgres://user:pass@localhost:5432/db" {
					t.Errorf("unexpected database URL: %s", c.DatabaseURL)
				}
				if c.RedisURL != "redis://localhost:6379/0" {
					t.Errorf("unexpected redis URL: %s", c.RedisURL)
				}
				if c.ShortDomain != "short.link" {
					t.Errorf("unexpected short domain: %s", c.ShortDomain)
				}
			},
		},
		{
			name: "missing database URL",
			envVars: map[string]string{
				"PORT":         "8080",
				"REDIS_URL":    "redis://localhost:6379/0",
				"SHORT_DOMAIN": "short.link",
			},
			wantErr: true,
			errType: ErrInvalidDatabaseURL,
		},
		{
			name: "missing redis URL",
			envVars: map[string]string{
				"PORT":         "8080",
				"DATABASE_URL": "postgres://user:pass@localhost:5432/db",
				"SHORT_DOMAIN": "short.link",
			},
			wantErr: true,
			errType: ErrInvalidRedisURL,
		},
		{
			name: "missing short domain",
			envVars: map[string]string{
				"PORT":         "8080",
				"DATABASE_URL": "postgres://user:pass@localhost:5432/db",
				"REDIS_URL":    "redis://localhost:6379/0",
			},
			wantErr: true,
			errType: ErrInvalidShortDomain,
		},
		{
			name: "invalid port - too low",
			envVars: map[string]string{
				"PORT":         "0",
				"DATABASE_URL": "postgres://user:pass@localhost:5432/db",
				"REDIS_URL":    "redis://localhost:6379/0",
				"SHORT_DOMAIN": "short.link",
			},
			wantErr: true,
			errType: ErrInvalidPort,
		},
		{
			name: "invalid port - too high",
			envVars: map[string]string{
				"PORT":         "65536",
				"DATABASE_URL": "postgres://user:pass@localhost:5432/db",
				"REDIS_URL":    "redis://localhost:6379/0",
				"SHORT_DOMAIN": "short.link",
			},
			wantErr: true,
			errType: ErrInvalidPort,
		},
		{
			name: "invalid log level",
			envVars: map[string]string{
				"PORT":         "8080",
				"DATABASE_URL": "postgres://user:pass@localhost:5432/db",
				"REDIS_URL":    "redis://localhost:6379/0",
				"SHORT_DOMAIN": "short.link",
				"LOG_LEVEL":    "invalid",
			},
			wantErr: true,
			errType: ErrInvalidLogLevel,
		},
		{
			name: "configuration with defaults",
			envVars: map[string]string{
				"DATABASE_URL": "postgres://user:pass@localhost:5432/db",
				"REDIS_URL":    "redis://localhost:6379/0",
				"SHORT_DOMAIN": "short.link",
			},
			wantErr: false,
			check: func(t *testing.T, c *Config) {
				if c.Port != 8080 {
					t.Errorf("expected default port 8080, got %d", c.Port)
				}
				if c.DatabaseMaxConns != 20 {
					t.Errorf("expected default max conns 20, got %d", c.DatabaseMaxConns)
				}
				if c.LogLevel != "info" {
					t.Errorf("expected default log level info, got %s", c.LogLevel)
				}
				if c.RateLimitPerIP != 100 {
					t.Errorf("expected default rate limit 100, got %d", c.RateLimitPerIP)
				}
			},
		},
		{
			name: "configuration with custom values",
			envVars: map[string]string{
				"PORT":                 "9090",
				"DATABASE_URL":         "postgres://user:pass@localhost:5432/db",
				"DATABASE_MAX_CONNS":   "50",
				"REDIS_URL":            "redis://localhost:6379/0",
				"SHORT_DOMAIN":         "short.link",
				"LOG_LEVEL":            "debug",
				"RATE_LIMIT_PER_IP":    "200",
				"ENVIRONMENT":          "production",
			},
			wantErr: false,
			check: func(t *testing.T, c *Config) {
				if c.Port != 9090 {
					t.Errorf("expected port 9090, got %d", c.Port)
				}
				if c.DatabaseMaxConns != 50 {
					t.Errorf("expected max conns 50, got %d", c.DatabaseMaxConns)
				}
				if c.LogLevel != "debug" {
					t.Errorf("expected log level debug, got %s", c.LogLevel)
				}
				if c.RateLimitPerIP != 200 {
					t.Errorf("expected rate limit 200, got %d", c.RateLimitPerIP)
				}
				if c.Environment != "production" {
					t.Errorf("expected environment production, got %s", c.Environment)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			clearEnv()

			// Set environment variables for this test
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			// Load configuration
			config, err := Load()

			// Check error
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Run additional checks
			if tt.check != nil {
				tt.check(t, config)
			}
		})
	}

	// Clean up
	clearEnv()
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr error
	}{
		{
			name: "valid configuration",
			config: &Config{
				Port:           8080,
				DatabaseURL:    "postgres://localhost",
				RedisURL:       "redis://localhost",
				ShortDomain:    "short.link",
				RateLimitPerIP: 100,
				LogLevel:       "info",
			},
			wantErr: nil,
		},
		{
			name: "invalid port - negative",
			config: &Config{
				Port:           -1,
				DatabaseURL:    "postgres://localhost",
				RedisURL:       "redis://localhost",
				ShortDomain:    "short.link",
				RateLimitPerIP: 100,
				LogLevel:       "info",
			},
			wantErr: ErrInvalidPort,
		},
		{
			name: "missing database URL",
			config: &Config{
				Port:           8080,
				RedisURL:       "redis://localhost",
				ShortDomain:    "short.link",
				RateLimitPerIP: 100,
				LogLevel:       "info",
			},
			wantErr: ErrInvalidDatabaseURL,
		},
		{
			name: "missing redis URL",
			config: &Config{
				Port:           8080,
				DatabaseURL:    "postgres://localhost",
				ShortDomain:    "short.link",
				RateLimitPerIP: 100,
				LogLevel:       "info",
			},
			wantErr: ErrInvalidRedisURL,
		},
		{
			name: "invalid rate limit",
			config: &Config{
				Port:           8080,
				DatabaseURL:    "postgres://localhost",
				RedisURL:       "redis://localhost",
				ShortDomain:    "short.link",
				RateLimitPerIP: -10,
				LogLevel:       "info",
			},
			wantErr: ErrInvalidRateLimit,
		},
		{
			name: "invalid log level",
			config: &Config{
				Port:           8080,
				DatabaseURL:    "postgres://localhost",
				RedisURL:       "redis://localhost",
				ShortDomain:    "short.link",
				RateLimitPerIP: 100,
				LogLevel:       "invalid",
			},
			wantErr: ErrInvalidLogLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if err != tt.wantErr {
				t.Errorf("expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestConfigIsDevelopment(t *testing.T) {
	tests := []struct {
		name        string
		environment string
		want        bool
	}{
		{"development", "development", true},
		{"Development uppercase", "DEVELOPMENT", true},
		{"production", "production", false},
		{"staging", "staging", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{Environment: tt.environment}
			if got := c.IsDevelopment(); got != tt.want {
				t.Errorf("IsDevelopment() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigIsProduction(t *testing.T) {
	tests := []struct {
		name        string
		environment string
		want        bool
	}{
		{"production", "production", true},
		{"Production uppercase", "PRODUCTION", true},
		{"development", "development", false},
		{"staging", "staging", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{Environment: tt.environment}
			if got := c.IsProduction(); got != tt.want {
				t.Errorf("IsProduction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetWorkerID(t *testing.T) {
	tests := []struct {
		name       string
		workerID   int
		instanceID string
		hostname   string
		wantRange  bool // Check if within valid range instead of exact value
	}{
		{
			name:      "explicit worker ID",
			workerID:  42,
			wantRange: false,
		},
		{
			name:       "auto-detect from Cloud Run instance ID",
			workerID:   -1,
			instanceID: "instance-123-abc",
			wantRange:  true,
		},
		{
			name:      "auto-detect from hostname",
			workerID:  -1,
			wantRange: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			os.Unsetenv("CLOUD_RUN_INSTANCE_ID")

			// Set Cloud Run instance ID if provided
			if tt.instanceID != "" {
				os.Setenv("CLOUD_RUN_INSTANCE_ID", tt.instanceID)
			}

			c := &Config{WorkerID: tt.workerID}
			got := c.GetWorkerID()

			if tt.wantRange {
				// Check if within valid range (0-1023)
				if got < 0 || got > 1023 {
					t.Errorf("GetWorkerID() = %d, want value between 0 and 1023", got)
				}
			} else {
				// Check exact value
				if got != tt.workerID {
					t.Errorf("GetWorkerID() = %d, want %d", got, tt.workerID)
				}
			}

			// Clean up
			os.Unsetenv("CLOUD_RUN_INSTANCE_ID")
		})
	}
}

func TestGetEnvHelpers(t *testing.T) {
	t.Run("getEnv", func(t *testing.T) {
		os.Setenv("TEST_STRING", "value")
		defer os.Unsetenv("TEST_STRING")

		if got := getEnv("TEST_STRING", "default"); got != "value" {
			t.Errorf("getEnv() = %v, want %v", got, "value")
		}

		if got := getEnv("NONEXISTENT", "default"); got != "default" {
			t.Errorf("getEnv() = %v, want %v", got, "default")
		}
	})

	t.Run("getEnvAsInt", func(t *testing.T) {
		os.Setenv("TEST_INT", "42")
		defer os.Unsetenv("TEST_INT")

		if got := getEnvAsInt("TEST_INT", 10); got != 42 {
			t.Errorf("getEnvAsInt() = %v, want %v", got, 42)
		}

		if got := getEnvAsInt("NONEXISTENT", 10); got != 10 {
			t.Errorf("getEnvAsInt() = %v, want %v", got, 10)
		}

		os.Setenv("TEST_INT", "invalid")
		if got := getEnvAsInt("TEST_INT", 10); got != 10 {
			t.Errorf("getEnvAsInt() with invalid value = %v, want %v", got, 10)
		}
	})

	t.Run("getEnvAsDuration", func(t *testing.T) {
		os.Setenv("TEST_DURATION", "5m")
		defer os.Unsetenv("TEST_DURATION")

		if got := getEnvAsDuration("TEST_DURATION", time.Minute); got != 5*time.Minute {
			t.Errorf("getEnvAsDuration() = %v, want %v", got, 5*time.Minute)
		}

		if got := getEnvAsDuration("NONEXISTENT", time.Minute); got != time.Minute {
			t.Errorf("getEnvAsDuration() = %v, want %v", got, time.Minute)
		}

		os.Setenv("TEST_DURATION", "invalid")
		if got := getEnvAsDuration("TEST_DURATION", time.Minute); got != time.Minute {
			t.Errorf("getEnvAsDuration() with invalid value = %v, want %v", got, time.Minute)
		}
	})
}

func TestHashString(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"simple string", "test"},
		{"complex string", "instance-123-abc-def"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := hashString(tt.input)

			// Hash should be non-negative
			if hash < 0 {
				t.Errorf("hashString() = %d, want non-negative value", hash)
			}

			// Hash should be consistent
			hash2 := hashString(tt.input)
			if hash != hash2 {
				t.Errorf("hashString() not consistent: %d != %d", hash, hash2)
			}
		})
	}
}

// clearEnv clears all environment variables used in tests
func clearEnv() {
	envVars := []string{
		"PORT",
		"SERVER_TIMEOUT",
		"DATABASE_URL",
		"DATABASE_MAX_CONNS",
		"DATABASE_MIN_CONNS",
		"DATABASE_MAX_LIFETIME",
		"DATABASE_MAX_IDLE_TIME",
		"REDIS_URL",
		"REDIS_MAX_CONNS",
		"REDIS_MIN_IDLE_CONNS",
		"REDIS_CONN_MAX_IDLE_TIME",
		"SHORT_DOMAIN",
		"WORKER_ID",
		"ENVIRONMENT",
		"RATE_LIMIT_PER_IP",
		"RATE_LIMIT_PER_IP_BURST",
		"RATE_LIMIT_WINDOW",
		"LOG_LEVEL",
		"GCP_PROJECT",
		"GCP_REGION",
		"CLOUD_RUN_INSTANCE_ID",
	}

	for _, v := range envVars {
		os.Unsetenv(v)
	}
}
