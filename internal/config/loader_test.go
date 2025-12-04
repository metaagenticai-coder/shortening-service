package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromFile(t *testing.T) {
	// Create temp directory for test files
	tempDir := t.TempDir()

	tests := []struct {
		name     string
		fileType string
		content  string
		envVars  map[string]string
		wantErr  bool
		check    func(*testing.T, *Config)
	}{
		{
			name:     "valid JSON config",
			fileType: "json",
			content: `{
				"server": {
					"port": 9090,
					"timeout": "15s"
				},
				"database": {
					"url": "postgres://user:pass@localhost:5432/testdb",
					"max_conns": 30
				},
				"redis": {
					"url": "redis://localhost:6379/1",
					"max_conns": 15
				},
				"service": {
					"short_domain": "test.link",
					"worker_id": 5,
					"environment": "testing"
				},
				"rate_limit": {
					"per_ip": 150,
					"per_ip_burst": 180,
					"window": "2m"
				},
				"logging": {
					"level": "debug"
				}
			}`,
			wantErr: false,
			check: func(t *testing.T, c *Config) {
				if c.Port != 9090 {
					t.Errorf("expected port 9090, got %d", c.Port)
				}
				if c.DatabaseMaxConns != 30 {
					t.Errorf("expected max conns 30, got %d", c.DatabaseMaxConns)
				}
				if c.LogLevel != "debug" {
					t.Errorf("expected log level debug, got %s", c.LogLevel)
				}
			},
		},
		{
			name:     "valid YAML config",
			fileType: "yaml",
			content: `
server:
  port: 9091
  timeout: "20s"
database:
  url: "postgres://user:pass@localhost:5432/testdb"
  max_conns: 35
redis:
  url: "redis://localhost:6379/1"
  max_conns: 20
service:
  short_domain: "test.link"
  worker_id: 10
  environment: "testing"
rate_limit:
  per_ip: 200
  per_ip_burst: 220
  window: "3m"
logging:
  level: "warn"
`,
			wantErr: false,
			check: func(t *testing.T, c *Config) {
				if c.Port != 9091 {
					t.Errorf("expected port 9091, got %d", c.Port)
				}
				if c.DatabaseMaxConns != 35 {
					t.Errorf("expected max conns 35, got %d", c.DatabaseMaxConns)
				}
				if c.LogLevel != "warn" {
					t.Errorf("expected log level warn, got %s", c.LogLevel)
				}
			},
		},
		{
			name:     "env vars override file config",
			fileType: "json",
			content: `{
				"server": {
					"port": 9090
				},
				"database": {
					"url": "postgres://user:pass@localhost:5432/testdb"
				},
				"redis": {
					"url": "redis://localhost:6379/1"
				},
				"service": {
					"short_domain": "test.link"
				},
				"logging": {
					"level": "debug"
				}
			}`,
			envVars: map[string]string{
				"PORT":      "8888",
				"LOG_LEVEL": "error",
			},
			wantErr: false,
			check: func(t *testing.T, c *Config) {
				if c.Port != 8888 {
					t.Errorf("expected env var port 8888, got %d", c.Port)
				}
				if c.LogLevel != "error" {
					t.Errorf("expected env var log level error, got %s", c.LogLevel)
				}
			},
		},
		{
			name:     "minimal config with defaults",
			fileType: "json",
			content: `{
				"database": {
					"url": "postgres://localhost/db"
				},
				"redis": {
					"url": "redis://localhost"
				},
				"service": {
					"short_domain": "short.test"
				}
			}`,
			wantErr: false,
			check: func(t *testing.T, c *Config) {
				// Should use default values for unspecified fields
				if c.Port != 8080 {
					t.Errorf("expected default port 8080, got %d", c.Port)
				}
				if c.DatabaseMaxConns != 20 {
					t.Errorf("expected default max conns 20, got %d", c.DatabaseMaxConns)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			clearEnv()

			// Set environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}
			defer clearEnv()

			// Create temp config file
			ext := tt.fileType
			if ext == "yaml" {
				ext = "yml"
			}
			configPath := filepath.Join(tempDir, "config-"+tt.name+"."+ext)
			if err := os.WriteFile(configPath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write config file: %v", err)
			}

			// Load configuration
			config, err := LoadFromFile(configPath)

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
}

func TestLoadFromFileErrors(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name     string
		fileType string
		content  string
		wantErr  bool
	}{
		{
			name:     "file not found",
			fileType: "json",
			content:  "",
			wantErr:  true,
		},
		{
			name:     "invalid JSON",
			fileType: "json",
			content:  `{"invalid": json}`,
			wantErr:  true,
		},
		{
			name:     "invalid YAML",
			fileType: "yaml",
			content:  "invalid:\n  - yaml\n  content:\n    missing: bracket",
			wantErr:  true,
		},
		{
			name:     "unsupported file format",
			fileType: "txt",
			content: `
database:
  url: "postgres://localhost"
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv()

			var configPath string
			if tt.name == "file not found" {
				configPath = filepath.Join(tempDir, "nonexistent.json")
			} else {
				ext := tt.fileType
				configPath = filepath.Join(tempDir, "config-error-"+tt.name+"."+ext)
				if err := os.WriteFile(configPath, []byte(tt.content), 0644); err != nil {
					t.Fatalf("failed to write config file: %v", err)
				}
			}

			// Load configuration
			_, err := LoadFromFile(configPath)

			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestLoadWithFileOverride(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("load from file when exists", func(t *testing.T) {
		clearEnv()

		content := `{
			"server": {"port": 7777},
			"database": {"url": "postgres://localhost/db"},
			"redis": {"url": "redis://localhost"},
			"service": {"short_domain": "s.test"}
		}`

		configPath := filepath.Join(tempDir, "config.json")
		if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write config file: %v", err)
		}

		config, err := LoadWithFileOverride(configPath)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if config.Port != 7777 {
			t.Errorf("expected port from file 7777, got %d", config.Port)
		}
	})

	t.Run("fallback to env when file not found", func(t *testing.T) {
		clearEnv()

		// Set required env vars
		os.Setenv("PORT", "6666")
		os.Setenv("DATABASE_URL", "postgres://localhost/db")
		os.Setenv("REDIS_URL", "redis://localhost")
		os.Setenv("SHORT_DOMAIN", "short.test")
		defer clearEnv()

		nonexistentPath := filepath.Join(tempDir, "nonexistent.json")
		config, err := LoadWithFileOverride(nonexistentPath)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if config.Port != 6666 {
			t.Errorf("expected port from env 6666, got %d", config.Port)
		}
	})

	t.Run("load from env when path is empty", func(t *testing.T) {
		clearEnv()

		// Set required env vars
		os.Setenv("PORT", "5555")
		os.Setenv("DATABASE_URL", "postgres://localhost/db")
		os.Setenv("REDIS_URL", "redis://localhost")
		os.Setenv("SHORT_DOMAIN", "short.test")
		defer clearEnv()

		config, err := LoadWithFileOverride("")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if config.Port != 5555 {
			t.Errorf("expected port from env 5555, got %d", config.Port)
		}
	})
}

func TestMergeFileConfig(t *testing.T) {
	t.Run("merge with empty base config", func(t *testing.T) {
		clearEnv()

		baseConfig := &Config{}
		fileConfig := &FileConfig{}
		fileConfig.Server.Port = 9999
		fileConfig.Database.URL = "postgres://test"
		fileConfig.Redis.URL = "redis://test"
		fileConfig.Service.ShortDomain = "test.com"

		err := mergeFileConfig(baseConfig, fileConfig)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if baseConfig.Port != 9999 {
			t.Errorf("expected port 9999, got %d", baseConfig.Port)
		}
		if baseConfig.DatabaseURL != "postgres://test" {
			t.Errorf("expected database URL from file, got %s", baseConfig.DatabaseURL)
		}
	})

	t.Run("env vars take precedence over file", func(t *testing.T) {
		clearEnv()

		os.Setenv("PORT", "8888")
		defer clearEnv()

		baseConfig := &Config{Port: 8888}
		fileConfig := &FileConfig{}
		fileConfig.Server.Port = 9999

		err := mergeFileConfig(baseConfig, fileConfig)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Port should remain from env var
		if baseConfig.Port != 8888 {
			t.Errorf("expected port from env 8888, got %d", baseConfig.Port)
		}
	})

	t.Run("invalid duration in file", func(t *testing.T) {
		clearEnv()

		baseConfig := &Config{}
		fileConfig := &FileConfig{}
		fileConfig.Server.Timeout = "invalid"

		err := mergeFileConfig(baseConfig, fileConfig)
		if err == nil {
			t.Error("expected error for invalid timeout duration")
		}
	})
}
