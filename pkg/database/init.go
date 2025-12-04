package database

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var (
	// ErrDatabaseNotReady indicates that the database is not ready
	ErrDatabaseNotReady = errors.New("database is not ready")
)

// WaitForDatabase waits for the database to become available
func WaitForDatabase(ctx context.Context, pool *Pool, maxRetries int, retryInterval time.Duration) error {
	if maxRetries <= 0 {
		maxRetries = 10
	}
	if retryInterval == 0 {
		retryInterval = 2 * time.Second
	}

	var lastErr error
	for i := 0; i < maxRetries; i++ {
		if i > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(retryInterval):
			}
		}

		checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := pool.HealthCheck(checkCtx)
		cancel()

		if err == nil {
			return nil
		}

		lastErr = err
	}

	return fmt.Errorf("%w: %v (tried %d times)", ErrDatabaseNotReady, lastErr, maxRetries)
}

// VerifySchema checks if the required database schema exists
func VerifySchema(ctx context.Context, pool *Pool) error {
	// Check if url_mappings table exists
	var exists bool
	query := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'public'
			AND table_name = 'url_mappings'
		)
	`

	err := pool.QueryRow(ctx, query).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to verify schema: %w", err)
	}

	if !exists {
		return errors.New("url_mappings table does not exist")
	}

	return nil
}

// InitOptions holds options for database initialization
type InitOptions struct {
	// WaitForReady determines if initialization should wait for DB to be ready
	WaitForReady bool
	// MaxRetries is the maximum number of connection retries
	MaxRetries int
	// RetryInterval is the time between retries
	RetryInterval time.Duration
	// VerifySchema determines if schema verification should be performed
	VerifySchema bool
	// CreateSchema determines if schema should be created if missing
	CreateSchema bool
}

// DefaultInitOptions returns InitOptions with sensible defaults
func DefaultInitOptions() *InitOptions {
	return &InitOptions{
		WaitForReady:  true,
		MaxRetries:    10,
		RetryInterval: 2 * time.Second,
		VerifySchema:  true,
		CreateSchema:  false, // Don't auto-create in production
	}
}

// Initialize initializes the database connection with options
func Initialize(ctx context.Context, config *Config, opts *InitOptions) (*Pool, error) {
	if opts == nil {
		opts = DefaultInitOptions()
	}

	// Create connection pool
	pool, err := NewPool(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Wait for database to be ready if requested
	if opts.WaitForReady {
		if err := WaitForDatabase(ctx, pool, opts.MaxRetries, opts.RetryInterval); err != nil {
			pool.Close()
			return nil, err
		}
	}

	// Verify schema if requested
	if opts.VerifySchema {
		if err := VerifySchema(ctx, pool); err != nil {
			if !opts.CreateSchema {
				pool.Close()
				return nil, fmt.Errorf("schema verification failed: %w", err)
			}
			// If CreateSchema is true, we would create the schema here
			// For now, we just return the error as schema creation should be
			// handled by migration tools
			pool.Close()
			return nil, fmt.Errorf("schema verification failed (auto-creation not implemented): %w", err)
		}
	}

	return pool, nil
}

// GracefulShutdown performs a graceful shutdown of the database pool
func GracefulShutdown(pool *Pool, timeout time.Duration) error {
	if pool == nil || pool.IsClosed() {
		return nil
	}

	// Create a context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Try to finish any ongoing operations
	done := make(chan struct{})
	go func() {
		pool.Close()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("graceful shutdown timed out after %v", timeout)
	}
}
