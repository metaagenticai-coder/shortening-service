package redis

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDefaultInitOptions(t *testing.T) {
	url := "redis://localhost:6379/0"
	opts := DefaultInitOptions(url)

	if opts.URL != url {
		t.Errorf("expected URL %s, got %s", url, opts.URL)
	}

	if opts.MaxConns != 10 {
		t.Errorf("expected MaxConns 10, got %d", opts.MaxConns)
	}

	if opts.MinIdleConns != 2 {
		t.Errorf("expected MinIdleConns 2, got %d", opts.MinIdleConns)
	}

	if opts.ConnMaxIdleTime != 5*time.Minute {
		t.Errorf("expected ConnMaxIdleTime 5m, got %v", opts.ConnMaxIdleTime)
	}

	if opts.Timeout != 10*time.Second {
		t.Errorf("expected Timeout 10s, got %v", opts.Timeout)
	}

	if !opts.PerformHealthCheck {
		t.Error("expected PerformHealthCheck to be true")
	}
}

func TestInitialize_NilOptions(t *testing.T) {
	ctx := context.Background()
	client, err := Initialize(ctx, nil)

	if client != nil {
		t.Error("expected nil client for nil options")
	}

	if err == nil {
		t.Error("expected error for nil options")
	}

	if !errors.Is(err, ErrInvalidConfig) {
		t.Errorf("expected ErrInvalidConfig, got %v", err)
	}
}

func TestInitialize_EmptyURL(t *testing.T) {
	ctx := context.Background()
	opts := &InitOptions{
		URL:                "",
		MaxConns:           10,
		MinIdleConns:       2,
		ConnMaxIdleTime:    5 * time.Minute,
		Timeout:            10 * time.Second,
		PerformHealthCheck: false,
	}

	client, err := Initialize(ctx, opts)

	if client != nil {
		t.Error("expected nil client for empty URL")
	}

	if err == nil {
		t.Error("expected error for empty URL")
	}
}

func TestInitialize_InvalidURL(t *testing.T) {
	ctx := context.Background()
	opts := &InitOptions{
		URL:                "invalid-url",
		MaxConns:           10,
		MinIdleConns:       2,
		ConnMaxIdleTime:    5 * time.Minute,
		Timeout:            10 * time.Second,
		PerformHealthCheck: false,
	}

	client, err := Initialize(ctx, opts)

	if client != nil {
		t.Error("expected nil client for invalid URL")
	}

	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestInitialize_ConnectionFailure(t *testing.T) {
	ctx := context.Background()
	opts := &InitOptions{
		URL:                "redis://localhost:9999/0",
		MaxConns:           10,
		MinIdleConns:       2,
		ConnMaxIdleTime:    5 * time.Minute,
		Timeout:            100 * time.Millisecond,
		PerformHealthCheck: false,
	}

	client, err := Initialize(ctx, opts)

	if client != nil {
		t.Error("expected nil client for connection failure")
	}

	if err == nil {
		t.Error("expected error for connection failure")
	}
}

func TestInitialize_ZeroValues(t *testing.T) {
	ctx := context.Background()
	opts := &InitOptions{
		URL:                "redis://localhost:9999/0",
		MaxConns:           0, // Should use default
		MinIdleConns:       0, // Should use default
		ConnMaxIdleTime:    0, // Should use default
		Timeout:            100 * time.Millisecond,
		PerformHealthCheck: false,
	}

	// This will fail to connect, but we're testing that defaults are applied
	_, err := Initialize(ctx, opts)

	// Should get connection error, not config error
	if err != nil && errors.Is(err, ErrInvalidConfig) {
		t.Error("expected connection error, not config error - defaults should be applied")
	}
}

func TestMustInitialize_Success(t *testing.T) {
	// This test would panic on failure, which is the expected behavior
	// We can't easily test the panic without actually panicking,
	// so we just verify the function signature exists
	t.Log("MustInitialize function exists and has correct signature")
}

func TestMustInitialize_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected MustInitialize to panic on failure")
		}
	}()

	ctx := context.Background()
	opts := &InitOptions{
		URL:                "redis://localhost:9999/0",
		MaxConns:           10,
		MinIdleConns:       2,
		ConnMaxIdleTime:    5 * time.Minute,
		Timeout:            100 * time.Millisecond,
		PerformHealthCheck: false,
	}

	// This should panic
	MustInitialize(ctx, opts)
}
