# Configuration Management

This package provides a robust configuration management system for the shortening service. It supports loading configuration from multiple sources with a clear precedence order: **Environment Variables > Config Files > Defaults**.

## Features

- **Environment Variable Support**: Load configuration from environment variables
- **Config File Support**: Load from JSON or YAML configuration files
- **Sensible Defaults**: All settings have sensible default values
- **Validation**: Comprehensive validation of all configuration values
- **Type Safety**: Strong typing for all configuration fields
- **Auto-detection**: Automatic worker ID detection from Cloud Run or hostname
- **Helper Methods**: Convenient methods for environment checks

## Usage

### Loading from Environment Variables

```go
import "github.com/metaagenticai/shortening-service/internal/config"

// Load configuration from environment variables with defaults
cfg, err := config.Load()
if err != nil {
    log.Fatal(err)
}
```

### Loading from Config File

```go
// Load from JSON or YAML file (supports both .json, .yaml, .yml extensions)
cfg, err := config.LoadFromFile("config.json")
if err != nil {
    log.Fatal(err)
}
```

### Loading with File Override

```go
// Try to load from file, fall back to environment if file doesn't exist
cfg, err := config.LoadWithFileOverride("config.yaml")
if err != nil {
    log.Fatal(err)
}
```

## Configuration Precedence

The configuration system follows this precedence order (highest to lowest):

1. **Environment Variables** - Always take precedence
2. **Config File Values** - Used if no environment variable is set
3. **Default Values** - Used if neither env var nor file value exists

Example:
```bash
# If PORT is set in environment
export PORT=9090

# And config.json has:
{
  "server": {
    "port": 8080
  }
}

# The final value will be 9090 (env var wins)
```

## Configuration Fields

### Server Configuration

| Field | Environment Variable | Default | Description |
|-------|---------------------|---------|-------------|
| Port | `PORT` | `8080` | HTTP server port (1-65535) |
| Timeout | `SERVER_TIMEOUT` | `10s` | Server timeout duration |

### Database Configuration

| Field | Environment Variable | Default | Description |
|-------|---------------------|---------|-------------|
| DatabaseURL | `DATABASE_URL` | - | PostgreSQL connection URL (required) |
| DatabaseMaxConns | `DATABASE_MAX_CONNS` | `20` | Maximum database connections |
| DatabaseMinConns | `DATABASE_MIN_CONNS` | `5` | Minimum database connections |
| DatabaseMaxLifetime | `DATABASE_MAX_LIFETIME` | `1h` | Max connection lifetime |
| DatabaseMaxIdleTime | `DATABASE_MAX_IDLE_TIME` | `30m` | Max connection idle time |

### Redis Configuration

| Field | Environment Variable | Default | Description |
|-------|---------------------|---------|-------------|
| RedisURL | `REDIS_URL` | - | Redis connection URL (required) |
| RedisMaxConns | `REDIS_MAX_CONNS` | `10` | Maximum Redis connections |
| RedisMinIdleConns | `REDIS_MIN_IDLE_CONNS` | `2` | Minimum idle connections |
| RedisConnMaxIdleTime | `REDIS_CONN_MAX_IDLE_TIME` | `5m` | Max connection idle time |

### Service Configuration

| Field | Environment Variable | Default | Description |
|-------|---------------------|---------|-------------|
| ShortDomain | `SHORT_DOMAIN` | - | Short URL domain (required) |
| WorkerID | `WORKER_ID` | `-1` | Worker ID for distributed ID generation (-1 = auto) |
| Environment | `ENVIRONMENT` | `development` | Environment name (development/production) |

### Rate Limiting Configuration

| Field | Environment Variable | Default | Description |
|-------|---------------------|---------|-------------|
| RateLimitPerIP | `RATE_LIMIT_PER_IP` | `100` | Requests per IP per window |
| RateLimitPerIPBurst | `RATE_LIMIT_PER_IP_BURST` | `120` | Burst limit for rate limiting |
| RateLimitWindow | `RATE_LIMIT_WINDOW` | `1m` | Rate limit time window |

### Logging Configuration

| Field | Environment Variable | Default | Description |
|-------|---------------------|---------|-------------|
| LogLevel | `LOG_LEVEL` | `info` | Log level (debug/info/warn/error) |

### GCP Configuration (Optional)

| Field | Environment Variable | Default | Description |
|-------|---------------------|---------|-------------|
| GCPProject | `GCP_PROJECT` | - | GCP project ID |
| GCPRegion | `GCP_REGION` | `us-central1` | GCP region |

## Config File Format

### JSON Format

```json
{
  "server": {
    "port": 8080,
    "timeout": "10s"
  },
  "database": {
    "url": "postgres://user:pass@localhost:5432/db",
    "max_conns": 20,
    "min_conns": 5,
    "max_lifetime": "1h",
    "max_idle_time": "30m"
  },
  "redis": {
    "url": "redis://localhost:6379/0",
    "max_conns": 10,
    "min_idle_conns": 2,
    "conn_max_idle_time": "5m"
  },
  "service": {
    "short_domain": "short.link",
    "worker_id": -1,
    "environment": "development"
  },
  "rate_limit": {
    "per_ip": 100,
    "per_ip_burst": 120,
    "window": "1m"
  },
  "logging": {
    "level": "info"
  },
  "gcp": {
    "project": "your-project-id",
    "region": "us-central1"
  }
}
```

### YAML Format

```yaml
server:
  port: 8080
  timeout: "10s"

database:
  url: "postgres://user:pass@localhost:5432/db"
  max_conns: 20
  min_conns: 5
  max_lifetime: "1h"
  max_idle_time: "30m"

redis:
  url: "redis://localhost:6379/0"
  max_conns: 10
  min_idle_conns: 2
  conn_max_idle_time: "5m"

service:
  short_domain: "short.link"
  worker_id: -1
  environment: "development"

rate_limit:
  per_ip: 100
  per_ip_burst: 120
  window: "1m"

logging:
  level: "info"

gcp:
  project: "your-project-id"
  region: "us-central1"
```

## Validation

The configuration system validates all values on load:

- **Port**: Must be between 1 and 65535
- **Database URL**: Required, cannot be empty
- **Redis URL**: Required, cannot be empty
- **Short Domain**: Required, cannot be empty
- **Rate Limit**: Must be positive
- **Log Level**: Must be one of: debug, info, warn, error

Invalid configuration will return an error with details about what failed validation.

## Helper Methods

### Environment Checks

```go
// Check if running in development
if cfg.IsDevelopment() {
    // Enable debug features
}

// Check if running in production
if cfg.IsProduction() {
    // Enable production optimizations
}
```

### Worker ID

```go
// Get worker ID (auto-detects if not explicitly set)
workerID := cfg.GetWorkerID()
// Returns value between 0-1023, suitable for Snowflake ID generation
```

Worker ID auto-detection logic:
1. If `WORKER_ID` env var is set to >= 0, use that value
2. If `CLOUD_RUN_INSTANCE_ID` env var exists, hash it to get worker ID
3. Otherwise, hash the hostname to get worker ID
4. Result is always masked to 10 bits (0-1023)

## Error Handling

All configuration errors are returned with descriptive messages:

```go
cfg, err := config.Load()
if err != nil {
    // Possible errors:
    // - ErrInvalidPort: Port out of range
    // - ErrInvalidDatabaseURL: Missing database URL
    // - ErrInvalidRedisURL: Missing Redis URL
    // - ErrInvalidShortDomain: Missing short domain
    // - ErrInvalidRateLimit: Invalid rate limit value
    // - ErrInvalidLogLevel: Invalid log level
    log.Fatalf("Configuration error: %v", err)
}
```

## Examples

### Minimal Configuration (Environment Variables)

```bash
export DATABASE_URL="postgres://localhost/db"
export REDIS_URL="redis://localhost:6379"
export SHORT_DOMAIN="short.link"
```

All other values will use defaults.

### Production Configuration

```bash
export PORT=8080
export DATABASE_URL="postgres://user:pass@prod-db:5432/shortener"
export DATABASE_MAX_CONNS=50
export REDIS_URL="redis://:pass@prod-redis:6379/0"
export SHORT_DOMAIN="s.example.com"
export ENVIRONMENT="production"
export LOG_LEVEL="warn"
export RATE_LIMIT_PER_IP=200
```

### Development with Config File

Create `config.yaml`:
```yaml
database:
  url: "postgres://localhost:5432/shortener_dev"
redis:
  url: "redis://localhost:6379/0"
service:
  short_domain: "localhost:8080"
  environment: "development"
logging:
  level: "debug"
```

Load in code:
```go
cfg, err := config.LoadFromFile("config.yaml")
```

## Testing

The package includes comprehensive unit tests:

```bash
# Run all tests
go test ./internal/config/...

# Run with coverage
go test -cover ./internal/config/...

# Run with verbose output
go test -v ./internal/config/...
```

Test coverage includes:
- Environment variable loading
- Config file loading (JSON and YAML)
- Precedence rules (env > file > defaults)
- Validation of all fields
- Error cases
- Helper methods
- Worker ID auto-detection

## Best Practices

1. **Use Environment Variables in Production**: For sensitive values like database passwords, always use environment variables rather than config files
2. **Use Config Files for Development**: Keep a local config file for development settings
3. **Never Commit Secrets**: Add `config.json` and `config.yaml` to `.gitignore`
4. **Validate Early**: Load and validate configuration at service startup
5. **Use Helper Methods**: Utilize `IsDevelopment()` and `IsProduction()` for environment-specific logic
6. **Set Reasonable Defaults**: The defaults are production-ready, adjust if needed

## Related Files

- `config.example.json` - Example JSON configuration
- `config.example.yaml` - Example YAML configuration
- `.env.example` - Example environment variables

## Dependencies

- `gopkg.in/yaml.v3` - YAML parsing support
