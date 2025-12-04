package database_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/metaagenticai/shortening-service/pkg/database"
)

// Example demonstrates basic database pool usage
func Example_basic() {
	ctx := context.Background()

	// Create database configuration with defaults
	config := database.DefaultConfig("postgres://user:pass@localhost:5432/mydb")

	// Create connection pool
	pool, err := database.NewPool(ctx, config)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	// Perform a simple query
	var version string
	err = pool.QueryRow(ctx, "SELECT version()").Scan(&version)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Connected to PostgreSQL")
	// Output would be: Connected to PostgreSQL
}

// Example demonstrates health monitoring
func Example_monitoring() {
	ctx := context.Background()

	// Create pool
	config := database.DefaultConfig("postgres://user:pass@localhost:5432/mydb")
	pool, err := database.NewPool(ctx, config)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	// Create monitor with 30-second check interval
	monitor := database.NewMonitor(pool, 30*time.Second)

	// Start monitoring
	monitor.Start(ctx)
	defer monitor.Stop()

	// Check health
	if monitor.IsHealthy() {
		fmt.Println("Database is healthy")

		// Get statistics
		stats := monitor.GetPoolStats()
		fmt.Printf("Active connections: %d\n", stats.AcquiredConns)
		fmt.Printf("Idle connections: %d\n", stats.IdleConns)
		fmt.Printf("Total connections: %d\n", stats.TotalConns)
	}
}

// Example demonstrates initialization with wait-for-ready
func Example_initialize() {
	ctx := context.Background()

	// Create configuration
	config := database.DefaultConfig("postgres://user:pass@localhost:5432/mydb")

	// Create initialization options
	opts := &database.InitOptions{
		WaitForReady:  true,
		MaxRetries:    10,
		RetryInterval: 2 * time.Second,
		VerifySchema:  false, // Skip schema verification for this example
		CreateSchema:  false,
	}

	// Initialize with retries
	pool, err := database.Initialize(ctx, config, opts)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	fmt.Println("Database initialized successfully")
}

// Example demonstrates transaction usage
func Example_transaction() {
	ctx := context.Background()

	config := database.DefaultConfig("postgres://user:pass@localhost:5432/mydb")
	pool, err := database.NewPool(ctx, config)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	// Begin transaction
	tx, err := pool.Begin(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer tx.Rollback(ctx) // Rollback if not committed

	// Execute operations in transaction
	_, err = tx.Exec(ctx, "INSERT INTO users (name) VALUES ($1)", "Alice")
	if err != nil {
		log.Fatal(err)
	}

	_, err = tx.Exec(ctx, "UPDATE accounts SET balance = balance - 100 WHERE user_id = $1", 1)
	if err != nil {
		log.Fatal(err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Transaction completed successfully")
}

// Example demonstrates graceful shutdown
func Example_gracefulShutdown() {
	ctx := context.Background()

	config := database.DefaultConfig("postgres://user:pass@localhost:5432/mydb")
	pool, err := database.NewPool(ctx, config)
	if err != nil {
		log.Fatal(err)
	}

	// Perform some operations...

	// Graceful shutdown with 10-second timeout
	if err := database.GracefulShutdown(pool, 10*time.Second); err != nil {
		log.Printf("Warning during shutdown: %v", err)
	}

	fmt.Println("Database connection closed gracefully")
}

// Example demonstrates health check with timeout
func Example_healthCheckWithTimeout() {
	ctx := context.Background()

	config := database.DefaultConfig("postgres://user:pass@localhost:5432/mydb")
	pool, err := database.NewPool(ctx, config)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	// Create context with timeout
	healthCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Perform health check
	if err := pool.HealthCheck(healthCtx); err != nil {
		fmt.Printf("Health check failed: %v\n", err)
		return
	}

	fmt.Println("Health check passed")
}

// Example demonstrates custom configuration
func Example_customConfiguration() {
	ctx := context.Background()

	// Create custom configuration
	config := &database.Config{
		DatabaseURL:       "postgres://user:pass@localhost:5432/mydb",
		MaxConns:          50,                   // Increase for high load
		MinConns:          10,                   // More idle connections
		MaxConnLifetime:   2 * time.Hour,        // Longer lifetime
		MaxConnIdleTime:   time.Hour,            // Longer idle time
		HealthCheckPeriod: 30 * time.Second,     // More frequent checks
		ConnectTimeout:    15 * time.Second,     // Longer connect timeout
	}

	pool, err := database.NewPool(ctx, config)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	fmt.Println("Pool created with custom configuration")
}

// Example demonstrates monitoring pool statistics
func Example_poolStatistics() {
	ctx := context.Background()

	config := database.DefaultConfig("postgres://user:pass@localhost:5432/mydb")
	pool, err := database.NewPool(ctx, config)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	// Create monitor
	monitor := database.NewMonitor(pool, time.Minute)
	monitor.Start(ctx)
	defer monitor.Stop()

	// Get detailed statistics
	stats := monitor.GetPoolStats()

	fmt.Printf("Pool Statistics:\n")
	fmt.Printf("  Max Connections:     %d\n", stats.MaxConns)
	fmt.Printf("  Total Connections:   %d\n", stats.TotalConns)
	fmt.Printf("  Acquired:            %d\n", stats.AcquiredConns)
	fmt.Printf("  Idle:                %d\n", stats.IdleConns)
	fmt.Printf("  Constructing:        %d\n", stats.ConstructingConns)
	fmt.Printf("  Acquire Count:       %d\n", stats.AcquireCount)
	fmt.Printf("  New Conns:           %d\n", stats.NewConnsCount)
	fmt.Printf("  Empty Acquire:       %d\n", stats.EmptyAcquireCount)
}

// Example demonstrates batch operations
func Example_batchOperations() {
	ctx := context.Background()

	config := database.DefaultConfig("postgres://user:pass@localhost:5432/mydb")
	pool, err := database.NewPool(ctx, config)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	// Begin transaction for batch insert
	tx, err := pool.Begin(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer tx.Rollback(ctx)

	// Batch insert
	for i := 0; i < 100; i++ {
		_, err = tx.Exec(ctx, "INSERT INTO items (name) VALUES ($1)", fmt.Sprintf("Item-%d", i))
		if err != nil {
			log.Fatal(err)
		}
	}

	// Commit batch
	if err := tx.Commit(ctx); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Batch operations completed")
}

// Example demonstrates error handling
func Example_errorHandling() {
	ctx := context.Background()

	config := database.DefaultConfig("postgres://invalid:url@localhost:5432/mydb")

	pool, err := database.NewPool(ctx, config)
	if err != nil {
		// Check error type
		if err == database.ErrConnectionFailed {
			fmt.Println("Failed to connect to database")
		} else {
			fmt.Printf("Error: %v\n", err)
		}
		return
	}
	defer pool.Close()
}

// Example demonstrates waiting for database availability
func Example_waitForDatabase() {
	ctx := context.Background()

	config := database.DefaultConfig("postgres://user:pass@localhost:5432/mydb")

	// Create pool (may fail if DB not ready)
	pool, err := database.NewPool(ctx, config)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	// Wait for database to be ready (with retries)
	err = database.WaitForDatabase(ctx, pool, 10, 2*time.Second)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Database is ready")
}

// Example demonstrates immediate health check
func Example_immediateHealthCheck() {
	ctx := context.Background()

	config := database.DefaultConfig("postgres://user:pass@localhost:5432/mydb")
	pool, err := database.NewPool(ctx, config)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	monitor := database.NewMonitor(pool, time.Minute)
	defer monitor.Stop()

	// Perform immediate health check (don't wait for periodic check)
	if err := monitor.CheckNow(ctx); err != nil {
		fmt.Printf("Health check failed: %v\n", err)
		return
	}

	// Get health status
	status := monitor.GetHealthStatus()
	fmt.Printf("Healthy: %v\n", status.Healthy)
	fmt.Printf("Response Time: %v\n", status.ResponseTime)
	fmt.Printf("Consecutive Fails: %d\n", status.ConsecutiveFails)
}
