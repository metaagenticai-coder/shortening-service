# Database Package

PostgreSQL database connection pool implementation using pgx/v5 with health checks, monitoring, and graceful shutdown support.

## Features

- **Connection Pooling**: Efficient connection pool management with configurable min/max connections
- **Health Checks**: Comprehensive health checking with automatic monitoring
- **Error Handling**: Proper error handling and recovery mechanisms
- **Thread Safety**: Concurrent-safe operations with mutex protection
- **Monitoring**: Real-time pool statistics and health status
- **Graceful Shutdown**: Clean shutdown with timeout support
- **Context Support**: Full context.Context support for cancellation and timeouts

## Installation

```bash
go get github.com/jackc/pgx/v5
go get github.com/jackc/pgx/v5/pgxpool
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/metaagenticai/shortening-service/pkg/database"
)

func main() {
    ctx := context.Background()

    // Create database configuration
    config := database.DefaultConfig("postgres://user:pass@localhost:5432/mydb")

    // Create connection pool
    pool, err := database.NewPool(ctx, config)
    if err != nil {
        log.Fatal(err)
    }
    defer pool.Close()

    // Use the pool
    var count int
    err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("User count: %d", count)
}
```

### With Initialization Options

```go
// Initialize with wait-for-ready and schema verification
config := database.DefaultConfig("postgres://user:pass@localhost:5432/mydb")

opts := &database.InitOptions{
    WaitForReady:  true,
    MaxRetries:    10,
    RetryInterval: 2 * time.Second,
    VerifySchema:  true,
    CreateSchema:  false,
}

pool, err := database.Initialize(ctx, config, opts)
if err != nil {
    log.Fatal(err)
}
defer pool.Close()
```

### With Health Monitoring

```go
// Create pool
pool, err := database.NewPool(ctx, config)
if err != nil {
    log.Fatal(err)
}
defer pool.Close()

// Create and start monitor
monitor := database.NewMonitor(pool, 30*time.Second)
monitor.Start(ctx)
defer monitor.Stop()

// Check health status
if monitor.IsHealthy() {
    log.Println("Database is healthy")
}

// Get detailed health status
status := monitor.GetHealthStatus()
log.Printf("Consecutive fails: %d", status.ConsecutiveFails)
log.Printf("Response time: %v", status.ResponseTime)

// Get pool statistics
stats := monitor.GetPoolStats()
log.Printf("Active connections: %d", stats.AcquiredConns)
log.Printf("Idle connections: %d", stats.IdleConns)
log.Printf("Total connections: %d", stats.TotalConns)
```

## Configuration

### Database Configuration

```go
config := &database.Config{
    DatabaseURL:       "postgres://user:pass@localhost:5432/mydb",
    MaxConns:          20,                    // Maximum connections in pool
    MinConns:          5,                     // Minimum connections in pool
    MaxConnLifetime:   time.Hour,             // Maximum lifetime of a connection
    MaxConnIdleTime:   30 * time.Minute,      // Maximum idle time
    HealthCheckPeriod: time.Minute,           // Health check interval
    ConnectTimeout:    10 * time.Second,      // Connection timeout
}
```

### Default Configuration

```go
config := database.DefaultConfig(databaseURL)
// Uses sensible defaults:
// - MaxConns: 20
// - MinConns: 5
// - MaxConnLifetime: 1 hour
// - MaxConnIdleTime: 30 minutes
// - HealthCheckPeriod: 1 minute
// - ConnectTimeout: 10 seconds
```

## API Reference

### Pool Operations

#### NewPool

Create a new database connection pool.

```go
pool, err := database.NewPool(ctx context.Context, config *Config) (*Pool, error)
```

#### Ping

Verify the database connection is alive.

```go
err := pool.Ping(ctx context.Context) error
```

#### HealthCheck

Perform a comprehensive health check.

```go
err := pool.HealthCheck(ctx context.Context) error
```

#### Query

Execute a query that returns rows.

```go
rows, err := pool.Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
```

#### QueryRow

Execute a query that returns at most one row.

```go
row := pool.QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
```

#### Exec

Execute a query that doesn't return rows.

```go
tag, err := pool.Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
```

#### Begin/BeginTx

Start a new transaction.

```go
tx, err := pool.Begin(ctx context.Context) (pgx.Tx, error)
tx, err := pool.BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
```

#### Stats

Get connection pool statistics.

```go
stats := pool.Stats() *pgxpool.Stat
```

#### Close

Close all connections in the pool.

```go
pool.Close()
```

### Monitoring

#### NewMonitor

Create a new database monitor.

```go
monitor := database.NewMonitor(pool *Pool, checkInterval time.Duration) *Monitor
```

#### Start/Stop

Start or stop health monitoring.

```go
monitor.Start(ctx context.Context)
monitor.Stop()
```

#### IsHealthy

Check if the database is healthy.

```go
healthy := monitor.IsHealthy() bool
```

#### GetHealthStatus

Get detailed health status.

```go
status := monitor.GetHealthStatus() HealthStatus
// Fields: Healthy, LastChecked, LastError, ConsecutiveFails, ResponseTime
```

#### GetPoolStats

Get connection pool statistics.

```go
stats := monitor.GetPoolStats() *PoolStats
// Fields: AcquireCount, AcquiredConns, IdleConns, TotalConns, etc.
```

### Initialization Helpers

#### Initialize

Initialize database connection with advanced options.

```go
pool, err := database.Initialize(ctx context.Context, config *Config, opts *InitOptions) (*Pool, error)
```

#### WaitForDatabase

Wait for database to become available.

```go
err := database.WaitForDatabase(ctx context.Context, pool *Pool, maxRetries int, retryInterval time.Duration) error
```

#### VerifySchema

Check if required database schema exists.

```go
err := database.VerifySchema(ctx context.Context, pool *Pool) error
```

#### GracefulShutdown

Perform graceful shutdown of the pool.

```go
err := database.GracefulShutdown(pool *Pool, timeout time.Duration) error
```

## Error Handling

The package defines several error types:

- `ErrPoolClosed`: Connection pool is closed
- `ErrConnectionFailed`: Database connection failed
- `ErrHealthCheckFailed`: Health check failed
- `ErrDatabaseNotReady`: Database is not ready

```go
if errors.Is(err, database.ErrPoolClosed) {
    log.Println("Pool is closed")
}
```

## Best Practices

### 1. Use Context for Timeouts

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

err := pool.Ping(ctx)
```

### 2. Monitor Pool Health

```go
monitor := database.NewMonitor(pool, 30*time.Second)
monitor.Start(ctx)
defer monitor.Stop()

// Check health before critical operations
if !monitor.IsHealthy() {
    return errors.New("database unhealthy")
}
```

### 3. Handle Graceful Shutdown

```go
// In your shutdown handler
err := database.GracefulShutdown(pool, 10*time.Second)
if err != nil {
    log.Printf("Shutdown warning: %v", err)
}
```

### 4. Use Transactions for Consistency

```go
tx, err := pool.Begin(ctx)
if err != nil {
    return err
}
defer tx.Rollback(ctx)

// Perform operations
_, err = tx.Exec(ctx, "INSERT INTO ...")
if err != nil {
    return err
}

// Commit if all operations succeed
return tx.Commit(ctx)
```

### 5. Monitor Pool Statistics

```go
stats := monitor.GetPoolStats()

if stats.AcquiredConns > stats.MaxConns * 0.8 {
    log.Println("Warning: Pool utilization > 80%")
}

if stats.IdleConns == 0 && stats.AcquiredConns == stats.MaxConns {
    log.Println("Warning: Pool exhausted")
}
```

## Integration with Service Configuration

```go
import (
    "github.com/metaagenticai/shortening-service/internal/config"
    "github.com/metaagenticai/shortening-service/pkg/database"
)

func InitDatabase(cfg *config.Config) (*database.Pool, error) {
    dbConfig := &database.Config{
        DatabaseURL:       cfg.DatabaseURL,
        MaxConns:          int32(cfg.DatabaseMaxConns),
        MinConns:          int32(cfg.DatabaseMinConns),
        MaxConnLifetime:   cfg.DatabaseMaxLifetime,
        MaxConnIdleTime:   cfg.DatabaseMaxIdleTime,
        HealthCheckPeriod: time.Minute,
        ConnectTimeout:    10 * time.Second,
    }

    opts := &database.InitOptions{
        WaitForReady:  true,
        MaxRetries:    10,
        RetryInterval: 2 * time.Second,
        VerifySchema:  !cfg.IsDevelopment(),
    }

    return database.Initialize(context.Background(), dbConfig, opts)
}
```

## Testing

### Unit Tests

Run unit tests (no database required):

```bash
go test ./pkg/database/...
```

### Integration Tests

Run with a test database:

```bash
export TEST_DATABASE_URL="postgres://user:pass@localhost:5432/testdb"
go test ./pkg/database/... -v
```

## Performance Considerations

1. **Connection Pool Size**: Set `MaxConns` based on your workload. For Cloud Run with 1000 req/s and 100ms latency, 20 connections is usually sufficient.

2. **Connection Lifetime**: Use `MaxConnLifetime` to ensure connections are recycled periodically (default: 1 hour).

3. **Health Check Frequency**: Balance between responsiveness and overhead. Default is 1 minute.

4. **Context Timeouts**: Always use appropriate timeouts to prevent hanging operations.

## Troubleshooting

### Connection Pool Exhausted

If you see connection exhaustion:

```go
stats := monitor.GetPoolStats()
log.Printf("Acquired: %d, Max: %d", stats.AcquiredConns, stats.MaxConns)
```

Solutions:
- Increase `MaxConns`
- Reduce operation latency
- Check for connection leaks (not releasing connections)

### Slow Health Checks

If health checks are slow:

```go
status := monitor.GetHealthStatus()
log.Printf("Response time: %v", status.ResponseTime)
```

Solutions:
- Check database performance
- Check network latency
- Increase health check interval

### Connection Failures

If connections fail:

```go
err := pool.Ping(ctx)
if err != nil {
    log.Printf("Connection error: %v", err)
}
```

Solutions:
- Verify database URL
- Check network connectivity
- Verify database is running
- Check authentication credentials

## License

This package is part of the shortening-service project.
