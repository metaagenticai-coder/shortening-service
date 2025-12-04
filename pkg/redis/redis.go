package redis

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	// ErrClientClosed indicates that the Redis client is already closed
	ErrClientClosed = errors.New("redis client is closed")
	// ErrConnectionFailed indicates that the Redis connection failed
	ErrConnectionFailed = errors.New("redis connection failed")
	// ErrHealthCheckFailed indicates that the health check failed
	ErrHealthCheckFailed = errors.New("redis health check failed")
	// ErrInvalidConfig indicates that the configuration is invalid
	ErrInvalidConfig = errors.New("invalid redis configuration")
)

// Config holds Redis client configuration
type Config struct {
	// URL is the Redis connection URL (e.g., redis://localhost:6379/0)
	URL string
	// MaxConns is the maximum number of connections in the pool
	MaxConns int
	// MinIdleConns is the minimum number of idle connections
	MinIdleConns int
	// ConnMaxIdleTime is the maximum idle time for a connection
	ConnMaxIdleTime time.Duration
	// DialTimeout is the timeout for establishing a connection
	DialTimeout time.Duration
	// ReadTimeout is the timeout for read operations
	ReadTimeout time.Duration
	// WriteTimeout is the timeout for write operations
	WriteTimeout time.Duration
	// PoolTimeout is the timeout when all connections are busy
	PoolTimeout time.Duration
}

// DefaultConfig returns a Config with sensible defaults for rate limiting
func DefaultConfig(url string) *Config {
	return &Config{
		URL:             url,
		MaxConns:        10,
		MinIdleConns:    2,
		ConnMaxIdleTime: 5 * time.Minute,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
		PoolTimeout:     4 * time.Second,
	}
}

// Client represents a Redis client wrapper with additional functionality
type Client struct {
	client *redis.Client
	config *Config
	mu     sync.RWMutex
	closed bool
}

// NewClient creates a new Redis client with the given configuration
func NewClient(ctx context.Context, config *Config) (*Client, error) {
	if config == nil {
		return nil, fmt.Errorf("%w: config cannot be nil", ErrInvalidConfig)
	}

	if config.URL == "" {
		return nil, fmt.Errorf("%w: URL cannot be empty", ErrInvalidConfig)
	}

	// Parse Redis URL
	opt, err := redis.ParseURL(config.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis URL: %w", err)
	}

	// Apply custom configuration
	opt.PoolSize = config.MaxConns
	opt.MinIdleConns = config.MinIdleConns
	opt.ConnMaxIdleTime = config.ConnMaxIdleTime
	opt.DialTimeout = config.DialTimeout
	opt.ReadTimeout = config.ReadTimeout
	opt.WriteTimeout = config.WriteTimeout
	opt.PoolTimeout = config.PoolTimeout

	// Create Redis client
	client := redis.NewClient(opt)

	// Verify connection with a ping
	pingCtx, cancel := context.WithTimeout(ctx, config.DialTimeout)
	defer cancel()

	if err := client.Ping(pingCtx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("%w: ping failed: %v", ErrConnectionFailed, err)
	}

	return &Client{
		client: client,
		config: config,
		closed: false,
	}, nil
}

// Ping verifies the Redis connection is alive
func (c *Client) Ping(ctx context.Context) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return ErrClientClosed
	}

	if err := c.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	return nil
}

// HealthCheck performs a comprehensive health check on the Redis connection
func (c *Client) HealthCheck(ctx context.Context) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return ErrClientClosed
	}

	// Create a timeout context for health check
	healthCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	// Perform ping to verify connectivity
	if err := c.client.Ping(healthCtx).Err(); err != nil {
		return fmt.Errorf("%w: %v", ErrHealthCheckFailed, err)
	}

	// Verify we can write and read a value
	testKey := "__health_check__"
	testValue := "ok"

	// Set test value with short expiration
	if err := c.client.Set(healthCtx, testKey, testValue, 5*time.Second).Err(); err != nil {
		return fmt.Errorf("%w: set operation failed: %v", ErrHealthCheckFailed, err)
	}

	// Get test value
	val, err := c.client.Get(healthCtx, testKey).Result()
	if err != nil {
		return fmt.Errorf("%w: get operation failed: %v", ErrHealthCheckFailed, err)
	}

	if val != testValue {
		return fmt.Errorf("%w: unexpected value from health check", ErrHealthCheckFailed)
	}

	// Clean up test key
	c.client.Del(healthCtx, testKey)

	return nil
}

// PoolStats returns statistics about the connection pool
func (c *Client) PoolStats() *redis.PoolStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return nil
	}

	return c.client.PoolStats()
}

// Get retrieves the value for a key
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return "", ErrClientClosed
	}

	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			// Key does not exist - return empty string and nil error
			return "", nil
		}
		return "", fmt.Errorf("get failed: %w", err)
	}

	return val, nil
}

// Set sets a key-value pair with optional expiration
func (c *Client) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return ErrClientClosed
	}

	if err := c.client.Set(ctx, key, value, expiration).Err(); err != nil {
		return fmt.Errorf("set failed: %w", err)
	}

	return nil
}

// Del deletes one or more keys
func (c *Client) Del(ctx context.Context, keys ...string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return ErrClientClosed
	}

	if err := c.client.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("del failed: %w", err)
	}

	return nil
}

// Incr increments the value of a key by 1
func (c *Client) Incr(ctx context.Context, key string) (int64, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return 0, ErrClientClosed
	}

	val, err := c.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("incr failed: %w", err)
	}

	return val, nil
}

// Decr decrements the value of a key by 1
func (c *Client) Decr(ctx context.Context, key string) (int64, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return 0, ErrClientClosed
	}

	val, err := c.client.Decr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("decr failed: %w", err)
	}

	return val, nil
}

// Expire sets a timeout on a key
func (c *Client) Expire(ctx context.Context, key string, expiration time.Duration) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return ErrClientClosed
	}

	if err := c.client.Expire(ctx, key, expiration).Err(); err != nil {
		return fmt.Errorf("expire failed: %w", err)
	}

	return nil
}

// TTL returns the remaining time to live of a key
func (c *Client) TTL(ctx context.Context, key string) (time.Duration, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return 0, ErrClientClosed
	}

	ttl, err := c.client.TTL(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("ttl failed: %w", err)
	}

	return ttl, nil
}

// Exists checks if one or more keys exist
func (c *Client) Exists(ctx context.Context, keys ...string) (int64, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return 0, ErrClientClosed
	}

	count, err := c.client.Exists(ctx, keys...).Result()
	if err != nil {
		return 0, fmt.Errorf("exists failed: %w", err)
	}

	return count, nil
}

// Close closes the Redis client
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	if c.client != nil {
		if err := c.client.Close(); err != nil {
			return fmt.Errorf("failed to close redis client: %w", err)
		}
	}

	c.closed = true
	return nil
}

// IsClosed returns true if the client is closed
func (c *Client) IsClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.closed
}

// Config returns a copy of the client configuration
func (c *Client) Config() *Config {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Return a copy to prevent modification
	configCopy := *c.config
	return &configCopy
}

// GetClient returns the underlying redis.Client for advanced operations
// Use with caution - direct access bypasses the wrapper's safety checks
func (c *Client) GetClient() *redis.Client {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.client
}
