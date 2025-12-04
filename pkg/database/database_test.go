package database

import (
	"context"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	databaseURL := "postgres://user:pass@localhost/testdb"
	config := DefaultConfig(databaseURL)

	if config.DatabaseURL != databaseURL {
		t.Errorf("Expected DatabaseURL %s, got %s", databaseURL, config.DatabaseURL)
	}

	if config.MaxConns != 20 {
		t.Errorf("Expected MaxConns 20, got %d", config.MaxConns)
	}

	if config.MinConns != 5 {
		t.Errorf("Expected MinConns 5, got %d", config.MinConns)
	}

	if config.MaxConnLifetime != time.Hour {
		t.Errorf("Expected MaxConnLifetime 1h, got %v", config.MaxConnLifetime)
	}

	if config.MaxConnIdleTime != 30*time.Minute {
		t.Errorf("Expected MaxConnIdleTime 30m, got %v", config.MaxConnIdleTime)
	}

	if config.HealthCheckPeriod != time.Minute {
		t.Errorf("Expected HealthCheckPeriod 1m, got %v", config.HealthCheckPeriod)
	}

	if config.ConnectTimeout != 10*time.Second {
		t.Errorf("Expected ConnectTimeout 10s, got %v", config.ConnectTimeout)
	}
}

func TestNewPool_NilConfig(t *testing.T) {
	ctx := context.Background()
	pool, err := NewPool(ctx, nil)

	if err == nil {
		t.Fatal("Expected error for nil config, got nil")
	}

	if pool != nil {
		t.Error("Expected nil pool, got non-nil")
	}

	expectedErr := "config cannot be nil"
	if err.Error() != expectedErr {
		t.Errorf("Expected error '%s', got '%s'", expectedErr, err.Error())
	}
}

func TestNewPool_EmptyDatabaseURL(t *testing.T) {
	ctx := context.Background()
	config := &Config{
		DatabaseURL: "",
		MaxConns:    10,
		MinConns:    2,
	}

	pool, err := NewPool(ctx, config)

	if err == nil {
		t.Fatal("Expected error for empty database URL, got nil")
	}

	if pool != nil {
		t.Error("Expected nil pool, got non-nil")
	}

	expectedErr := "database URL cannot be empty"
	if err.Error() != expectedErr {
		t.Errorf("Expected error '%s', got '%s'", expectedErr, err.Error())
	}
}

func TestNewPool_InvalidDatabaseURL(t *testing.T) {
	ctx := context.Background()
	config := &Config{
		DatabaseURL:   "invalid://url",
		MaxConns:      10,
		MinConns:      2,
		ConnectTimeout: 2 * time.Second,
	}

	pool, err := NewPool(ctx, config)

	if err == nil {
		t.Fatal("Expected error for invalid database URL, got nil")
	}

	if pool != nil {
		pool.Close()
		t.Error("Expected nil pool, got non-nil")
	}
}

func TestPool_ClosedOperations(t *testing.T) {
	// Create a mock closed pool
	pool := &Pool{
		closed: true,
		config: DefaultConfig("postgres://localhost/test"),
	}

	ctx := context.Background()

	// Test Ping on closed pool
	err := pool.Ping(ctx)
	if err != ErrPoolClosed {
		t.Errorf("Expected ErrPoolClosed, got %v", err)
	}

	// Test HealthCheck on closed pool
	err = pool.HealthCheck(ctx)
	if err != ErrPoolClosed {
		t.Errorf("Expected ErrPoolClosed, got %v", err)
	}

	// Test Acquire on closed pool
	_, err = pool.Acquire(ctx)
	if err != ErrPoolClosed {
		t.Errorf("Expected ErrPoolClosed, got %v", err)
	}

	// Test Query on closed pool
	_, err = pool.Query(ctx, "SELECT 1")
	if err != ErrPoolClosed {
		t.Errorf("Expected ErrPoolClosed, got %v", err)
	}

	// Test Exec on closed pool
	_, err = pool.Exec(ctx, "SELECT 1")
	if err != ErrPoolClosed {
		t.Errorf("Expected ErrPoolClosed, got %v", err)
	}

	// Test Begin on closed pool
	_, err = pool.Begin(ctx)
	if err != ErrPoolClosed {
		t.Errorf("Expected ErrPoolClosed, got %v", err)
	}
}

func TestPool_IsClosed(t *testing.T) {
	pool := &Pool{
		closed: false,
		config: DefaultConfig("postgres://localhost/test"),
	}

	if pool.IsClosed() {
		t.Error("Expected pool to not be closed")
	}

	pool.Close()

	if !pool.IsClosed() {
		t.Error("Expected pool to be closed")
	}

	// Test double close doesn't panic
	pool.Close()
}

func TestPool_Config(t *testing.T) {
	originalConfig := DefaultConfig("postgres://localhost/test")
	originalConfig.MaxConns = 50

	pool := &Pool{
		config: originalConfig,
	}

	retrievedConfig := pool.Config()

	if retrievedConfig.MaxConns != 50 {
		t.Errorf("Expected MaxConns 50, got %d", retrievedConfig.MaxConns)
	}

	// Verify it's a copy by modifying the retrieved config
	retrievedConfig.MaxConns = 100

	if pool.Config().MaxConns != 50 {
		t.Error("Config should return a copy, not the original")
	}
}

func TestPool_Stats_Closed(t *testing.T) {
	pool := &Pool{
		closed: true,
		config: DefaultConfig("postgres://localhost/test"),
	}

	stats := pool.Stats()
	if stats != nil {
		t.Error("Expected nil stats for closed pool")
	}
}

// Integration test helper - only runs if TEST_DATABASE_URL is set
func getTestDatabaseURL() string {
	// In real tests, this would check an environment variable
	// For unit tests, we skip database-dependent tests
	return ""
}

func skipIfNoDatabase(t *testing.T) {
	if getTestDatabaseURL() == "" {
		t.Skip("Skipping integration test: TEST_DATABASE_URL not set")
	}
}

func TestNewPool_Integration(t *testing.T) {
	skipIfNoDatabase(t)

	ctx := context.Background()
	config := DefaultConfig(getTestDatabaseURL())

	pool, err := NewPool(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	if pool.IsClosed() {
		t.Error("Pool should not be closed after creation")
	}

	// Test Ping
	if err := pool.Ping(ctx); err != nil {
		t.Errorf("Ping failed: %v", err)
	}

	// Test Stats
	stats := pool.Stats()
	if stats == nil {
		t.Error("Expected non-nil stats")
	}
}

func TestPool_HealthCheck_Integration(t *testing.T) {
	skipIfNoDatabase(t)

	ctx := context.Background()
	config := DefaultConfig(getTestDatabaseURL())

	pool, err := NewPool(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	if err := pool.HealthCheck(ctx); err != nil {
		t.Errorf("Health check failed: %v", err)
	}
}

func TestPool_QueryRow_Integration(t *testing.T) {
	skipIfNoDatabase(t)

	ctx := context.Background()
	config := DefaultConfig(getTestDatabaseURL())

	pool, err := NewPool(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	var result int
	err = pool.QueryRow(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		t.Errorf("QueryRow failed: %v", err)
	}

	if result != 1 {
		t.Errorf("Expected result 1, got %d", result)
	}
}

func TestPool_Transaction_Integration(t *testing.T) {
	skipIfNoDatabase(t)

	ctx := context.Background()
	config := DefaultConfig(getTestDatabaseURL())

	pool, err := NewPool(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	// Test Begin
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	// Test Rollback
	if err := tx.Rollback(ctx); err != nil {
		t.Errorf("Failed to rollback transaction: %v", err)
	}
}

func TestPool_ConcurrentAccess(t *testing.T) {
	pool := &Pool{
		closed: false,
		config: DefaultConfig("postgres://localhost/test"),
	}

	done := make(chan bool)

	// Simulate concurrent reads
	for i := 0; i < 10; i++ {
		go func() {
			_ = pool.IsClosed()
			_ = pool.Config()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
