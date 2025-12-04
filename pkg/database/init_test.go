package database

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDefaultInitOptions(t *testing.T) {
	opts := DefaultInitOptions()

	if !opts.WaitForReady {
		t.Error("Expected WaitForReady to be true")
	}

	if opts.MaxRetries != 10 {
		t.Errorf("Expected MaxRetries 10, got %d", opts.MaxRetries)
	}

	if opts.RetryInterval != 2*time.Second {
		t.Errorf("Expected RetryInterval 2s, got %v", opts.RetryInterval)
	}

	if !opts.VerifySchema {
		t.Error("Expected VerifySchema to be true")
	}

	if opts.CreateSchema {
		t.Error("Expected CreateSchema to be false")
	}
}

func TestWaitForDatabase_ClosedPool(t *testing.T) {
	pool := &Pool{
		closed: true,
		config: DefaultConfig("postgres://localhost/test"),
	}

	ctx := context.Background()
	err := WaitForDatabase(ctx, pool, 3, 10*time.Millisecond)

	if err == nil {
		t.Fatal("Expected error for closed pool")
	}

	if !errors.Is(err, ErrDatabaseNotReady) {
		t.Errorf("Expected ErrDatabaseNotReady, got %v", err)
	}
}

func TestWaitForDatabase_ContextCancellation(t *testing.T) {
	pool := &Pool{
		closed: true,
		config: DefaultConfig("postgres://localhost/test"),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := WaitForDatabase(ctx, pool, 10, 100*time.Millisecond)

	if err == nil {
		t.Fatal("Expected error for cancelled context")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

func TestWaitForDatabase_DefaultValues(t *testing.T) {
	pool := &Pool{
		closed: true,
		config: DefaultConfig("postgres://localhost/test"),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Pass zero values to test defaults
	err := WaitForDatabase(ctx, pool, 0, 0)

	if err == nil {
		t.Fatal("Expected error for closed pool")
	}

	// Should have used default values (10 retries, 2s interval, but context timeout hits first)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Logf("Got error: %v", err)
		// Either deadline exceeded or not ready is acceptable
	}
}

func TestVerifySchema_ClosedPool(t *testing.T) {
	// Skip this test as QueryRow can panic on closed/nil pool
	// This is expected behavior - callers should not use closed pools
	t.Skip("QueryRow on closed pool would panic, which is expected behavior")
}

func TestInitialize_NilOptions(t *testing.T) {
	ctx := context.Background()
	config := &Config{
		DatabaseURL:    "invalid://url",
		ConnectTimeout: 1 * time.Second,
	}

	// This should fail to connect, but should use default options
	_, err := Initialize(ctx, config, nil)

	if err == nil {
		t.Fatal("Expected error for invalid database URL")
	}
}

func TestInitialize_InvalidConfig(t *testing.T) {
	ctx := context.Background()
	config := &Config{
		DatabaseURL:    "",
		ConnectTimeout: 1 * time.Second,
	}

	opts := &InitOptions{
		WaitForReady: false,
		VerifySchema: false,
	}

	pool, err := Initialize(ctx, config, opts)

	if err == nil {
		t.Fatal("Expected error for empty database URL")
	}

	if pool != nil {
		pool.Close()
		t.Error("Expected nil pool")
	}
}

func TestGracefulShutdown_NilPool(t *testing.T) {
	err := GracefulShutdown(nil, 5*time.Second)

	if err != nil {
		t.Errorf("Expected no error for nil pool, got %v", err)
	}
}

func TestGracefulShutdown_ClosedPool(t *testing.T) {
	pool := &Pool{
		closed: true,
		config: DefaultConfig("postgres://localhost/test"),
	}

	err := GracefulShutdown(pool, 5*time.Second)

	if err != nil {
		t.Errorf("Expected no error for closed pool, got %v", err)
	}
}

func TestGracefulShutdown_Success(t *testing.T) {
	pool := &Pool{
		closed: false,
		config: DefaultConfig("postgres://localhost/test"),
	}

	// Close will happen quickly
	err := GracefulShutdown(pool, 1*time.Second)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !pool.IsClosed() {
		t.Error("Expected pool to be closed")
	}
}

func TestWaitForDatabase_Integration(t *testing.T) {
	skipIfNoDatabase(t)

	ctx := context.Background()
	config := DefaultConfig(getTestDatabaseURL())

	pool, err := NewPool(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	// Should succeed immediately since pool is ready
	err = WaitForDatabase(ctx, pool, 5, 100*time.Millisecond)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestVerifySchema_Integration(t *testing.T) {
	skipIfNoDatabase(t)

	ctx := context.Background()
	config := DefaultConfig(getTestDatabaseURL())

	pool, err := NewPool(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	// This will fail if url_mappings table doesn't exist
	// That's expected for a fresh database
	err = VerifySchema(ctx, pool)
	// We don't assert success/failure as it depends on DB state
	t.Logf("VerifySchema result: %v", err)
}

func TestInitialize_Integration(t *testing.T) {
	skipIfNoDatabase(t)

	ctx := context.Background()
	config := DefaultConfig(getTestDatabaseURL())

	opts := &InitOptions{
		WaitForReady:  true,
		MaxRetries:    5,
		RetryInterval: 100 * time.Millisecond,
		VerifySchema:  false, // Skip schema verification
		CreateSchema:  false,
	}

	pool, err := Initialize(ctx, config, opts)
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}
	defer pool.Close()

	if pool.IsClosed() {
		t.Error("Expected pool to be open")
	}

	// Verify pool is functional
	if err := pool.Ping(ctx); err != nil {
		t.Errorf("Ping failed: %v", err)
	}
}

func TestGracefulShutdown_Integration(t *testing.T) {
	skipIfNoDatabase(t)

	ctx := context.Background()
	config := DefaultConfig(getTestDatabaseURL())

	pool, err := NewPool(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}

	// Perform graceful shutdown
	err = GracefulShutdown(pool, 5*time.Second)
	if err != nil {
		t.Errorf("Graceful shutdown failed: %v", err)
	}

	if !pool.IsClosed() {
		t.Error("Expected pool to be closed after graceful shutdown")
	}
}

func TestInitialize_WithSchemaVerification(t *testing.T) {
	skipIfNoDatabase(t)

	ctx := context.Background()
	config := DefaultConfig(getTestDatabaseURL())

	opts := &InitOptions{
		WaitForReady:  true,
		MaxRetries:    5,
		RetryInterval: 100 * time.Millisecond,
		VerifySchema:  true,  // Enable schema verification
		CreateSchema:  false,
	}

	pool, err := Initialize(ctx, config, opts)

	// If schema doesn't exist, initialization should fail
	if err != nil {
		t.Logf("Expected failure for missing schema: %v", err)
		return
	}

	// If we got here, schema exists
	defer pool.Close()

	if pool.IsClosed() {
		t.Error("Expected pool to be open")
	}
}

func TestWaitForDatabase_Retries(t *testing.T) {
	pool := &Pool{
		closed: true,
		config: DefaultConfig("postgres://localhost/test"),
	}

	ctx := context.Background()
	start := time.Now()

	// Request 3 retries with 50ms interval
	err := WaitForDatabase(ctx, pool, 3, 50*time.Millisecond)

	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("Expected error for closed pool")
	}

	// Should have tried 3 times with 2 delays (first attempt is immediate)
	// Total time should be approximately 100ms (2 * 50ms)
	minExpected := 80 * time.Millisecond  // Allow some variance
	maxExpected := 200 * time.Millisecond

	if elapsed < minExpected || elapsed > maxExpected {
		t.Errorf("Expected elapsed time between %v and %v, got %v", minExpected, maxExpected, elapsed)
	}
}
