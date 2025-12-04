package redis

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	url := "redis://localhost:6379/0"
	config := DefaultConfig(url)

	if config.URL != url {
		t.Errorf("expected URL %s, got %s", url, config.URL)
	}

	if config.MaxConns != 10 {
		t.Errorf("expected MaxConns 10, got %d", config.MaxConns)
	}

	if config.MinIdleConns != 2 {
		t.Errorf("expected MinIdleConns 2, got %d", config.MinIdleConns)
	}

	if config.ConnMaxIdleTime != 5*time.Minute {
		t.Errorf("expected ConnMaxIdleTime 5m, got %v", config.ConnMaxIdleTime)
	}

	if config.DialTimeout != 5*time.Second {
		t.Errorf("expected DialTimeout 5s, got %v", config.DialTimeout)
	}

	if config.ReadTimeout != 3*time.Second {
		t.Errorf("expected ReadTimeout 3s, got %v", config.ReadTimeout)
	}

	if config.WriteTimeout != 3*time.Second {
		t.Errorf("expected WriteTimeout 3s, got %v", config.WriteTimeout)
	}

	if config.PoolTimeout != 4*time.Second {
		t.Errorf("expected PoolTimeout 4s, got %v", config.PoolTimeout)
	}
}

func TestNewClient_NilConfig(t *testing.T) {
	ctx := context.Background()
	client, err := NewClient(ctx, nil)

	if client != nil {
		t.Error("expected nil client for nil config")
	}

	if err == nil {
		t.Error("expected error for nil config")
	}

	if !errors.Is(err, ErrInvalidConfig) {
		t.Errorf("expected ErrInvalidConfig, got %v", err)
	}
}

func TestNewClient_EmptyURL(t *testing.T) {
	ctx := context.Background()
	config := &Config{
		URL:             "",
		MaxConns:        10,
		MinIdleConns:    2,
		ConnMaxIdleTime: 5 * time.Minute,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
		PoolTimeout:     4 * time.Second,
	}

	client, err := NewClient(ctx, config)

	if client != nil {
		t.Error("expected nil client for empty URL")
	}

	if err == nil {
		t.Error("expected error for empty URL")
	}

	if !errors.Is(err, ErrInvalidConfig) {
		t.Errorf("expected ErrInvalidConfig, got %v", err)
	}
}

func TestNewClient_InvalidURL(t *testing.T) {
	ctx := context.Background()
	config := &Config{
		URL:             "invalid-url",
		MaxConns:        10,
		MinIdleConns:    2,
		ConnMaxIdleTime: 5 * time.Minute,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
		PoolTimeout:     4 * time.Second,
	}

	client, err := NewClient(ctx, config)

	if client != nil {
		t.Error("expected nil client for invalid URL")
	}

	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestNewClient_ConnectionFailure(t *testing.T) {
	ctx := context.Background()
	// Use a URL that will fail to connect
	config := DefaultConfig("redis://localhost:9999/0")
	config.DialTimeout = 100 * time.Millisecond

	client, err := NewClient(ctx, config)

	if client != nil {
		t.Error("expected nil client for connection failure")
	}

	if err == nil {
		t.Error("expected error for connection failure")
	}

	if !errors.Is(err, ErrConnectionFailed) {
		t.Errorf("expected ErrConnectionFailed, got %v", err)
	}
}

func TestClient_IsClosed(t *testing.T) {
	client := &Client{
		closed: false,
	}

	if client.IsClosed() {
		t.Error("expected client to not be closed")
	}

	client.closed = true

	if !client.IsClosed() {
		t.Error("expected client to be closed")
	}
}

func TestClient_Config(t *testing.T) {
	originalConfig := DefaultConfig("redis://localhost:6379/0")
	client := &Client{
		config: originalConfig,
		closed: false,
	}

	returnedConfig := client.Config()

	// Verify it's a copy
	if returnedConfig == originalConfig {
		t.Error("expected Config() to return a copy, not the original")
	}

	// Verify values are the same
	if returnedConfig.URL != originalConfig.URL {
		t.Errorf("expected URL %s, got %s", originalConfig.URL, returnedConfig.URL)
	}

	if returnedConfig.MaxConns != originalConfig.MaxConns {
		t.Errorf("expected MaxConns %d, got %d", originalConfig.MaxConns, returnedConfig.MaxConns)
	}
}

func TestClient_Close_AlreadyClosed(t *testing.T) {
	client := &Client{
		closed: true,
	}

	err := client.Close()
	if err != nil {
		t.Errorf("expected no error closing already closed client, got %v", err)
	}
}

func TestClient_Ping_ClosedClient(t *testing.T) {
	ctx := context.Background()
	client := &Client{
		closed: true,
	}

	err := client.Ping(ctx)
	if err == nil {
		t.Error("expected error pinging closed client")
	}

	if !errors.Is(err, ErrClientClosed) {
		t.Errorf("expected ErrClientClosed, got %v", err)
	}
}

func TestClient_HealthCheck_ClosedClient(t *testing.T) {
	ctx := context.Background()
	client := &Client{
		closed: true,
	}

	err := client.HealthCheck(ctx)
	if err == nil {
		t.Error("expected error for health check on closed client")
	}

	if !errors.Is(err, ErrClientClosed) {
		t.Errorf("expected ErrClientClosed, got %v", err)
	}
}

func TestClient_Get_ClosedClient(t *testing.T) {
	ctx := context.Background()
	client := &Client{
		closed: true,
	}

	_, err := client.Get(ctx, "test-key")
	if err == nil {
		t.Error("expected error for get on closed client")
	}

	if !errors.Is(err, ErrClientClosed) {
		t.Errorf("expected ErrClientClosed, got %v", err)
	}
}

func TestClient_Set_ClosedClient(t *testing.T) {
	ctx := context.Background()
	client := &Client{
		closed: true,
	}

	err := client.Set(ctx, "test-key", "test-value", 0)
	if err == nil {
		t.Error("expected error for set on closed client")
	}

	if !errors.Is(err, ErrClientClosed) {
		t.Errorf("expected ErrClientClosed, got %v", err)
	}
}

func TestClient_Del_ClosedClient(t *testing.T) {
	ctx := context.Background()
	client := &Client{
		closed: true,
	}

	err := client.Del(ctx, "test-key")
	if err == nil {
		t.Error("expected error for del on closed client")
	}

	if !errors.Is(err, ErrClientClosed) {
		t.Errorf("expected ErrClientClosed, got %v", err)
	}
}

func TestClient_Incr_ClosedClient(t *testing.T) {
	ctx := context.Background()
	client := &Client{
		closed: true,
	}

	_, err := client.Incr(ctx, "test-key")
	if err == nil {
		t.Error("expected error for incr on closed client")
	}

	if !errors.Is(err, ErrClientClosed) {
		t.Errorf("expected ErrClientClosed, got %v", err)
	}
}

func TestClient_Decr_ClosedClient(t *testing.T) {
	ctx := context.Background()
	client := &Client{
		closed: true,
	}

	_, err := client.Decr(ctx, "test-key")
	if err == nil {
		t.Error("expected error for decr on closed client")
	}

	if !errors.Is(err, ErrClientClosed) {
		t.Errorf("expected ErrClientClosed, got %v", err)
	}
}

func TestClient_Expire_ClosedClient(t *testing.T) {
	ctx := context.Background()
	client := &Client{
		closed: true,
	}

	err := client.Expire(ctx, "test-key", time.Minute)
	if err == nil {
		t.Error("expected error for expire on closed client")
	}

	if !errors.Is(err, ErrClientClosed) {
		t.Errorf("expected ErrClientClosed, got %v", err)
	}
}

func TestClient_TTL_ClosedClient(t *testing.T) {
	ctx := context.Background()
	client := &Client{
		closed: true,
	}

	_, err := client.TTL(ctx, "test-key")
	if err == nil {
		t.Error("expected error for ttl on closed client")
	}

	if !errors.Is(err, ErrClientClosed) {
		t.Errorf("expected ErrClientClosed, got %v", err)
	}
}

func TestClient_Exists_ClosedClient(t *testing.T) {
	ctx := context.Background()
	client := &Client{
		closed: true,
	}

	_, err := client.Exists(ctx, "test-key")
	if err == nil {
		t.Error("expected error for exists on closed client")
	}

	if !errors.Is(err, ErrClientClosed) {
		t.Errorf("expected ErrClientClosed, got %v", err)
	}
}

func TestClient_PoolStats_ClosedClient(t *testing.T) {
	client := &Client{
		closed: true,
	}

	stats := client.PoolStats()
	if stats != nil {
		t.Error("expected nil stats for closed client")
	}
}

func TestClient_GetClient(t *testing.T) {
	client := &Client{
		closed: false,
	}

	underlyingClient := client.GetClient()
	if underlyingClient != nil {
		// Expected behavior - returns the underlying client even if nil
		t.Log("GetClient() returned underlying client")
	}
}
