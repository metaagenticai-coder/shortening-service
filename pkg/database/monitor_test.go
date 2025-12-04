package database

import (
	"context"
	"testing"
	"time"
)

func TestNewMonitor(t *testing.T) {
	pool := &Pool{
		config: DefaultConfig("postgres://localhost/test"),
	}

	monitor := NewMonitor(pool, 10*time.Second)

	if monitor.pool != pool {
		t.Error("Monitor pool not set correctly")
	}

	if monitor.checkInterval != 10*time.Second {
		t.Errorf("Expected checkInterval 10s, got %v", monitor.checkInterval)
	}

	if !monitor.healthStatus.Healthy {
		t.Error("Expected initial health status to be healthy")
	}
}

func TestNewMonitor_DefaultInterval(t *testing.T) {
	pool := &Pool{
		config: DefaultConfig("postgres://localhost/test"),
	}

	monitor := NewMonitor(pool, 0)

	if monitor.checkInterval != 30*time.Second {
		t.Errorf("Expected default checkInterval 30s, got %v", monitor.checkInterval)
	}
}

func TestMonitor_GetHealthStatus(t *testing.T) {
	pool := &Pool{
		config: DefaultConfig("postgres://localhost/test"),
	}

	monitor := NewMonitor(pool, 10*time.Second)

	status := monitor.GetHealthStatus()

	if !status.Healthy {
		t.Error("Expected healthy status")
	}

	if status.ConsecutiveFails != 0 {
		t.Errorf("Expected 0 consecutive fails, got %d", status.ConsecutiveFails)
	}

	if status.LastError != nil {
		t.Errorf("Expected no error, got %v", status.LastError)
	}
}

func TestMonitor_IsHealthy(t *testing.T) {
	pool := &Pool{
		config: DefaultConfig("postgres://localhost/test"),
	}

	monitor := NewMonitor(pool, 10*time.Second)

	if !monitor.IsHealthy() {
		t.Error("Expected monitor to report healthy")
	}

	// Simulate unhealthy status
	monitor.mu.Lock()
	monitor.healthStatus.Healthy = false
	monitor.mu.Unlock()

	if monitor.IsHealthy() {
		t.Error("Expected monitor to report unhealthy")
	}
}

func TestMonitor_Stop(t *testing.T) {
	pool := &Pool{
		config: DefaultConfig("postgres://localhost/test"),
	}

	monitor := NewMonitor(pool, 10*time.Second)

	// Stop should not panic
	monitor.Stop()

	if !monitor.stopped {
		t.Error("Expected monitor to be stopped")
	}

	// Double stop should not panic
	monitor.Stop()
}

func TestMonitor_GetPoolStats_NilPool(t *testing.T) {
	pool := &Pool{
		closed: true,
		config: DefaultConfig("postgres://localhost/test"),
	}

	monitor := NewMonitor(pool, 10*time.Second)

	stats := monitor.GetPoolStats()
	if stats != nil {
		t.Error("Expected nil stats for closed pool")
	}
}

func TestMonitor_StartStop(t *testing.T) {
	pool := &Pool{
		closed: true, // Closed pool won't panic, just fail health checks
		config: DefaultConfig("postgres://localhost/test"),
	}

	monitor := NewMonitor(pool, 100*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Start the monitor
	monitor.Start(ctx)

	// Let it run for a bit
	time.Sleep(200 * time.Millisecond)

	// Stop the monitor
	monitor.Stop()

	// Verify it stopped
	if !monitor.stopped {
		t.Error("Expected monitor to be stopped")
	}
}

func TestMonitor_PerformHealthCheck_ClosedPool(t *testing.T) {
	pool := &Pool{
		closed: true,
		config: DefaultConfig("postgres://localhost/test"),
	}

	monitor := NewMonitor(pool, 10*time.Second)

	ctx := context.Background()
	monitor.performHealthCheck(ctx)

	status := monitor.GetHealthStatus()

	if status.Healthy {
		t.Error("Expected unhealthy status for closed pool")
	}

	if status.ConsecutiveFails != 1 {
		t.Errorf("Expected 1 consecutive fail, got %d", status.ConsecutiveFails)
	}

	if status.LastError != ErrPoolClosed {
		t.Errorf("Expected ErrPoolClosed, got %v", status.LastError)
	}
}

func TestMonitor_PerformHealthCheck_MultipleFailures(t *testing.T) {
	pool := &Pool{
		closed: true,
		config: DefaultConfig("postgres://localhost/test"),
	}

	monitor := NewMonitor(pool, 10*time.Second)

	ctx := context.Background()

	// Perform multiple health checks
	for i := 1; i <= 5; i++ {
		monitor.performHealthCheck(ctx)

		status := monitor.GetHealthStatus()

		if status.ConsecutiveFails != i {
			t.Errorf("Expected %d consecutive fails, got %d", i, status.ConsecutiveFails)
		}
	}
}

func TestMonitor_CheckNow(t *testing.T) {
	pool := &Pool{
		closed: true,
		config: DefaultConfig("postgres://localhost/test"),
	}

	monitor := NewMonitor(pool, 10*time.Second)

	ctx := context.Background()
	err := monitor.CheckNow(ctx)

	if err != ErrPoolClosed {
		t.Errorf("Expected ErrPoolClosed, got %v", err)
	}

	status := monitor.GetHealthStatus()
	if status.Healthy {
		t.Error("Expected unhealthy status")
	}
}

func TestMonitor_Integration(t *testing.T) {
	skipIfNoDatabase(t)

	ctx := context.Background()
	config := DefaultConfig(getTestDatabaseURL())

	pool, err := NewPool(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	monitor := NewMonitor(pool, 100*time.Millisecond)
	defer monitor.Stop()

	// Start monitoring
	monitorCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	monitor.Start(monitorCtx)

	// Wait for a health check to run
	time.Sleep(200 * time.Millisecond)

	if !monitor.IsHealthy() {
		t.Error("Expected monitor to report healthy")
	}

	status := monitor.GetHealthStatus()
	if status.ConsecutiveFails != 0 {
		t.Errorf("Expected 0 consecutive fails, got %d", status.ConsecutiveFails)
	}

	if status.ResponseTime == 0 {
		t.Error("Expected non-zero response time")
	}

	// Check pool stats
	stats := monitor.GetPoolStats()
	if stats == nil {
		t.Error("Expected non-nil pool stats")
	}
}

func TestMonitor_ContextCancellation(t *testing.T) {
	pool := &Pool{
		closed: true, // Closed pool won't panic, just fail health checks
		config: DefaultConfig("postgres://localhost/test"),
	}

	monitor := NewMonitor(pool, 50*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())

	monitor.Start(ctx)

	// Cancel context after a short delay
	time.Sleep(100 * time.Millisecond)
	cancel()

	// Give it time to stop
	time.Sleep(100 * time.Millisecond)

	// Manually stop to clean up
	monitor.Stop()
}

func TestPoolStats_Fields(t *testing.T) {
	skipIfNoDatabase(t)

	ctx := context.Background()
	config := DefaultConfig(getTestDatabaseURL())

	pool, err := NewPool(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	monitor := NewMonitor(pool, 10*time.Second)
	stats := monitor.GetPoolStats()

	if stats == nil {
		t.Fatal("Expected non-nil stats")
	}

	// Verify stats structure
	if stats.MaxConns != config.MaxConns {
		t.Errorf("Expected MaxConns %d, got %d", config.MaxConns, stats.MaxConns)
	}

	// TotalConns should be at least MinConns
	if stats.TotalConns < config.MinConns {
		t.Errorf("Expected TotalConns >= %d, got %d", config.MinConns, stats.TotalConns)
	}
}
