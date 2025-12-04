package redis

import (
	"context"
	"fmt"
	"time"
)

// InitOptions holds options for initializing a Redis client
type InitOptions struct {
	// URL is the Redis connection URL
	URL string
	// MaxConns is the maximum number of connections (0 = use default)
	MaxConns int
	// MinIdleConns is the minimum number of idle connections (0 = use default)
	MinIdleConns int
	// ConnMaxIdleTime is the maximum idle time for connections (0 = use default)
	ConnMaxIdleTime time.Duration
	// Timeout is the timeout for establishing initial connection
	Timeout time.Duration
	// PerformHealthCheck determines if a health check should be performed on init
	PerformHealthCheck bool
}

// DefaultInitOptions returns InitOptions with sensible defaults
func DefaultInitOptions(url string) *InitOptions {
	return &InitOptions{
		URL:                url,
		MaxConns:           10,
		MinIdleConns:       2,
		ConnMaxIdleTime:    5 * time.Minute,
		Timeout:            10 * time.Second,
		PerformHealthCheck: true,
	}
}

// Initialize creates and validates a new Redis client with the given options
func Initialize(ctx context.Context, opts *InitOptions) (*Client, error) {
	if opts == nil {
		return nil, fmt.Errorf("%w: options cannot be nil", ErrInvalidConfig)
	}

	// Create config from options
	config := &Config{
		URL:             opts.URL,
		MaxConns:        opts.MaxConns,
		MinIdleConns:    opts.MinIdleConns,
		ConnMaxIdleTime: opts.ConnMaxIdleTime,
		DialTimeout:     opts.Timeout,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
		PoolTimeout:     4 * time.Second,
	}

	// Use defaults if values are zero
	if config.MaxConns == 0 {
		config.MaxConns = 10
	}
	if config.MinIdleConns == 0 {
		config.MinIdleConns = 2
	}
	if config.ConnMaxIdleTime == 0 {
		config.ConnMaxIdleTime = 5 * time.Minute
	}
	if config.DialTimeout == 0 {
		config.DialTimeout = 10 * time.Second
	}

	// Create context with timeout
	initCtx, cancel := context.WithTimeout(ctx, config.DialTimeout)
	defer cancel()

	// Create client
	client, err := NewClient(initCtx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize redis client: %w", err)
	}

	// Perform health check if requested
	if opts.PerformHealthCheck {
		healthCtx, healthCancel := context.WithTimeout(ctx, 5*time.Second)
		defer healthCancel()

		if err := client.HealthCheck(healthCtx); err != nil {
			client.Close()
			return nil, fmt.Errorf("redis health check failed: %w", err)
		}
	}

	return client, nil
}

// MustInitialize is like Initialize but panics on error
// Use this only in initialization code where startup should fail on error
func MustInitialize(ctx context.Context, opts *InitOptions) *Client {
	client, err := Initialize(ctx, opts)
	if err != nil {
		panic(fmt.Sprintf("failed to initialize redis client: %v", err))
	}
	return client
}
