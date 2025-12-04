# Redis Client Package

A robust Redis client wrapper with connection pooling, health checking, and monitoring capabilities for the URL shortening service.

## Features

- **Connection Pooling**: Configurable connection pool with min/max connections
- **Health Checks**: Comprehensive health checking with read/write verification
- **Monitoring**: Built-in monitoring with statistics and metrics
- **Thread-Safe**: All operations are safe for concurrent use
- **Error Handling**: Proper error types and detailed error messages
- **Context Support**: All operations support context cancellation

## Installation

The package uses `github.com/redis/go-redis/v9` as the underlying Redis client.

```bash
go get github.com/redis/go-redis/v9
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/metaagenticai/shortening-service/pkg/redis"
)

func main() {
    ctx := context.Background()

    // Create a client with default configuration
    config := redis.DefaultConfig("redis://localhost:6379/0")
    client, err := redis.NewClient(ctx, config)
    if err != nil {
        log.Fatalf("Failed to create Redis client: %v", err)
    }
    defer client.Close()

    // Set a value with 1 hour expiration
    err = client.Set(ctx, "mykey", "myvalue", time.Hour)
    if err != nil {
        log.Fatalf("Failed to set key: %v", err)
    }

    // Get a value
    value, err := client.Get(ctx, "mykey")
    if err != nil {
        log.Fatalf("Failed to get key: %v", err)
    }

    log.Printf("Value: %s", value)
}
```

### Using Initialize Helper

```go
ctx := context.Background()
opts := redis.DefaultInitOptions("redis://localhost:6379/0")

client, err := redis.Initialize(ctx, opts)
if err != nil {
    log.Fatalf("Failed to initialize: %v", err)
}
defer client.Close()
```

## Configuration

### Config Structure

```go
type Config struct {
    URL             string        // Redis connection URL
    MaxConns        int           // Maximum connections (default: 10)
    MinIdleConns    int           // Minimum idle connections (default: 2)
    ConnMaxIdleTime time.Duration // Max idle time (default: 5 minutes)
    DialTimeout     time.Duration // Connection timeout (default: 5 seconds)
    ReadTimeout     time.Duration // Read timeout (default: 3 seconds)
    WriteTimeout    time.Duration // Write timeout (default: 3 seconds)
    PoolTimeout     time.Duration // Pool timeout (default: 4 seconds)
}
```

### Default Configuration

```go
config := redis.DefaultConfig("redis://localhost:6379/0")
// Returns config with sensible defaults for rate limiting and caching
```

### Custom Configuration

```go
config := &redis.Config{
    URL:             "redis://localhost:6379/0",
    MaxConns:        20,
    MinIdleConns:    5,
    ConnMaxIdleTime: 10 * time.Minute,
    DialTimeout:     10 * time.Second,
    ReadTimeout:     5 * time.Second,
    WriteTimeout:    5 * time.Second,
    PoolTimeout:     6 * time.Second,
}
```

## Core Operations

### Key-Value Operations

```go
// Set a key with expiration
err := client.Set(ctx, "key", "value", time.Hour)

// Get a value (returns "" if key doesn't exist)
value, err := client.Get(ctx, "key")

// Delete keys
err := client.Del(ctx, "key1", "key2")

// Check if keys exist
count, err := client.Exists(ctx, "key1", "key2")
```

### Counter Operations

```go
// Increment a counter
count, err := client.Incr(ctx, "counter")

// Decrement a counter
count, err := client.Decr(ctx, "counter")
```

### Expiration Operations

```go
// Set expiration on a key
err := client.Expire(ctx, "key", time.Minute)

// Get remaining TTL
ttl, err := client.TTL(ctx, "key")
```

## Health Monitoring

### Basic Health Check

```go
// Perform comprehensive health check
err := client.HealthCheck(ctx)
if err != nil {
    log.Printf("Redis unhealthy: %v", err)
}
```

### Continuous Monitoring

```go
// Create a monitor with 30-second check interval
monitor := redis.NewMonitor(client, 30*time.Second)

// Start monitoring
monitor.Start(ctx)
defer monitor.Stop()

// Check health status
if monitor.IsHealthy() {
    log.Println("Redis is healthy")
}

// Get detailed status
status := monitor.GetStatus()
log.Printf("Checks: %d, Errors: %d", status.CheckCount, status.ErrorCount)

// Get metrics for monitoring systems
metrics := monitor.GetMetrics()
log.Printf("Success rate: %s", metrics["success_rate"])
```

## Use Cases

### Rate Limiting

```go
// Implement rate limiting for an IP address
func checkRateLimit(client *redis.Client, ipAddress string) (bool, error) {
    ctx := context.Background()
    key := fmt.Sprintf("ratelimit:ip:%s", ipAddress)
    maxRequests := int64(100)
    window := time.Minute

    // Increment counter
    count, err := client.Incr(ctx, key)
    if err != nil {
        return false, err
    }

    // Set expiration on first request
    if count == 1 {
        err = client.Expire(ctx, key, window)
        if err != nil {
            return false, err
        }
    }

    // Check if limit exceeded
    return count <= maxRequests, nil
}
```

### URL Caching

```go
// Cache URL mappings for fast lookups
func cacheURL(client *redis.Client, shortCode, longURL string) error {
    ctx := context.Background()
    key := fmt.Sprintf("url:%s", shortCode)
    return client.Set(ctx, key, longURL, time.Hour)
}

func getCachedURL(client *redis.Client, shortCode string) (string, error) {
    ctx := context.Background()
    key := fmt.Sprintf("url:%s", shortCode)
    return client.Get(ctx, key)
}
```

## Connection Pool Statistics

```go
// Get pool statistics
stats := client.PoolStats()
if stats != nil {
    log.Printf("Total connections: %d", stats.TotalConns)
    log.Printf("Idle connections: %d", stats.IdleConns)
    log.Printf("Stale connections: %d", stats.StaleConns)
}
```

## Error Handling

The package defines several error types:

- `ErrClientClosed`: Client has been closed
- `ErrConnectionFailed`: Failed to connect to Redis
- `ErrHealthCheckFailed`: Health check failed
- `ErrInvalidConfig`: Configuration is invalid

```go
if err != nil {
    if errors.Is(err, redis.ErrClientClosed) {
        log.Println("Client is closed")
    } else if errors.Is(err, redis.ErrConnectionFailed) {
        log.Println("Connection failed")
    }
}
```

## Thread Safety

All client operations are thread-safe and can be called from multiple goroutines concurrently. The client uses mutex locks to protect internal state.

## Testing

The package includes comprehensive unit tests and example tests:

```bash
# Run tests
go test ./pkg/redis/...

# Run tests with coverage
go test ./pkg/redis/... -cover

# Run tests with verbose output
go test ./pkg/redis/... -v
```

## Best Practices

1. **Reuse Clients**: Create one client and reuse it throughout your application
2. **Use Context**: Always pass context with appropriate timeouts
3. **Handle Errors**: Check and handle all errors appropriately
4. **Monitor Health**: Use the monitoring functionality in production
5. **Close Properly**: Always defer `client.Close()` after creating a client
6. **Set Expiration**: Always set expiration on keys to prevent memory leaks
7. **Connection Pooling**: Configure pool size based on your workload

## Performance Considerations

- **Connection Pool**: Default pool size (10) is suitable for most use cases
- **Timeouts**: Adjust timeouts based on your latency requirements
- **Idle Connections**: Keep min idle connections (2) for faster request handling
- **Expiration**: Use appropriate TTL values to balance cache hit rate and memory

## Integration with Config Package

The Redis client integrates with the service configuration:

```go
import (
    "github.com/metaagenticai/shortening-service/internal/config"
    "github.com/metaagenticai/shortening-service/pkg/redis"
)

// Load service configuration
cfg, err := config.Load()
if err != nil {
    log.Fatal(err)
}

// Create Redis client from config
redisConfig := &redis.Config{
    URL:             cfg.RedisURL,
    MaxConns:        cfg.RedisMaxConns,
    MinIdleConns:    cfg.RedisMinIdleConns,
    ConnMaxIdleTime: cfg.RedisConnMaxIdleTime,
    DialTimeout:     5 * time.Second,
    ReadTimeout:     3 * time.Second,
    WriteTimeout:    3 * time.Second,
    PoolTimeout:     4 * time.Second,
}

client, err := redis.NewClient(ctx, redisConfig)
```

## License

This package is part of the URL shortening service project.
