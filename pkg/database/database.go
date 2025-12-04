package database

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	// ErrPoolClosed indicates that the connection pool is already closed
	ErrPoolClosed = errors.New("database connection pool is closed")
	// ErrConnectionFailed indicates that the database connection failed
	ErrConnectionFailed = errors.New("database connection failed")
	// ErrHealthCheckFailed indicates that the health check failed
	ErrHealthCheckFailed = errors.New("database health check failed")
)

// Config holds database connection configuration
type Config struct {
	// DatabaseURL is the PostgreSQL connection string
	DatabaseURL string
	// MaxConns is the maximum number of connections in the pool
	MaxConns int32
	// MinConns is the minimum number of connections in the pool
	MinConns int32
	// MaxConnLifetime is the maximum lifetime of a connection
	MaxConnLifetime time.Duration
	// MaxConnIdleTime is the maximum idle time of a connection
	MaxConnIdleTime time.Duration
	// HealthCheckPeriod is the interval between health checks
	HealthCheckPeriod time.Duration
	// ConnectTimeout is the timeout for establishing a connection
	ConnectTimeout time.Duration
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig(databaseURL string) *Config {
	return &Config{
		DatabaseURL:       databaseURL,
		MaxConns:          20,
		MinConns:          5,
		MaxConnLifetime:   time.Hour,
		MaxConnIdleTime:   30 * time.Minute,
		HealthCheckPeriod: time.Minute,
		ConnectTimeout:    10 * time.Second,
	}
}

// Pool represents a PostgreSQL connection pool with additional functionality
type Pool struct {
	pool   *pgxpool.Pool
	config *Config
	mu     sync.RWMutex
	closed bool
}

// NewPool creates a new database connection pool
func NewPool(ctx context.Context, config *Config) (*Pool, error) {
	if config == nil {
		return nil, errors.New("config cannot be nil")
	}

	if config.DatabaseURL == "" {
		return nil, errors.New("database URL cannot be empty")
	}

	// Parse the connection string and create pool config
	poolConfig, err := pgxpool.ParseConfig(config.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Apply custom configuration
	poolConfig.MaxConns = config.MaxConns
	poolConfig.MinConns = config.MinConns
	poolConfig.MaxConnLifetime = config.MaxConnLifetime
	poolConfig.MaxConnIdleTime = config.MaxConnIdleTime
	poolConfig.HealthCheckPeriod = config.HealthCheckPeriod

	// Set connection timeout
	poolConfig.ConnConfig.ConnectTimeout = config.ConnectTimeout

	// Create connection pool with timeout context
	connectCtx, cancel := context.WithTimeout(ctx, config.ConnectTimeout)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(connectCtx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}

	// Verify connection with a ping
	if err := pool.Ping(connectCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("%w: ping failed: %v", ErrConnectionFailed, err)
	}

	return &Pool{
		pool:   pool,
		config: config,
		closed: false,
	}, nil
}

// Ping verifies the database connection is alive
func (p *Pool) Ping(ctx context.Context) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return ErrPoolClosed
	}

	if err := p.pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	return nil
}

// HealthCheck performs a comprehensive health check on the database
func (p *Pool) HealthCheck(ctx context.Context) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return ErrPoolClosed
	}

	// Create a timeout context for health check
	healthCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Perform a simple query to verify database functionality
	var result int
	err := p.pool.QueryRow(healthCtx, "SELECT 1").Scan(&result)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrHealthCheckFailed, err)
	}

	if result != 1 {
		return fmt.Errorf("%w: unexpected result from health check query", ErrHealthCheckFailed)
	}

	return nil
}

// Stats returns statistics about the connection pool
func (p *Pool) Stats() *pgxpool.Stat {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return nil
	}

	return p.pool.Stat()
}

// Acquire gets a connection from the pool
func (p *Pool) Acquire(ctx context.Context) (*pgxpool.Conn, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return nil, ErrPoolClosed
	}

	conn, err := p.pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire connection: %w", err)
	}

	return conn, nil
}

// Query executes a query that returns rows
func (p *Pool) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return nil, ErrPoolClosed
	}

	rows, err := p.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	return rows, nil
}

// QueryRow executes a query that returns at most one row
func (p *Pool) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.pool.QueryRow(ctx, sql, args...)
}

// Exec executes a query that doesn't return rows
func (p *Pool) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return pgconn.CommandTag{}, ErrPoolClosed
	}

	tag, err := p.pool.Exec(ctx, sql, args...)
	if err != nil {
		return pgconn.CommandTag{}, fmt.Errorf("exec failed: %w", err)
	}

	return tag, nil
}

// Begin starts a new transaction
func (p *Pool) Begin(ctx context.Context) (pgx.Tx, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return nil, ErrPoolClosed
	}

	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	return tx, nil
}

// BeginTx starts a new transaction with options
func (p *Pool) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return nil, ErrPoolClosed
	}

	tx, err := p.pool.BeginTx(ctx, txOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	return tx, nil
}

// Close closes all connections in the pool
func (p *Pool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return
	}

	if p.pool != nil {
		p.pool.Close()
	}
	p.closed = true
}

// IsClosed returns true if the pool is closed
func (p *Pool) IsClosed() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.closed
}

// Config returns a copy of the pool configuration
func (p *Pool) Config() *Config {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Return a copy to prevent modification
	configCopy := *p.config
	return &configCopy
}
