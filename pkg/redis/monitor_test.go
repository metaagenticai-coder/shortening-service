package redis

import (
	"context"
	"testing"
	"time"
)

func TestNewMonitor(t *testing.T) {
	client := &Client{
		closed: false,
	}

	interval := 30 * time.Second
	monitor := NewMonitor(client, interval)

	if monitor.client != client {
		t.Error("expected monitor to reference the client")
	}

	if monitor.interval != interval {
		t.Errorf("expected interval %v, got %v", interval, monitor.interval)
	}

	if monitor.stopCh == nil {
		t.Error("expected stopCh to be initialized")
	}

	if monitor.stoppedCh == nil {
		t.Error("expected stoppedCh to be initialized")
	}
}

func TestNewMonitor_ZeroInterval(t *testing.T) {
	client := &Client{
		closed: false,
	}

	monitor := NewMonitor(client, 0)

	// Should use default interval of 30 seconds
	if monitor.interval != 30*time.Second {
		t.Errorf("expected default interval 30s, got %v", monitor.interval)
	}
}

func TestMonitor_IsHealthy(t *testing.T) {
	client := &Client{
		closed: false,
	}

	monitor := NewMonitor(client, time.Minute)

	// Initially should be unhealthy
	if monitor.IsHealthy() {
		t.Error("expected monitor to be unhealthy initially")
	}

	// Set healthy
	monitor.mu.Lock()
	monitor.status.Healthy = true
	monitor.mu.Unlock()

	if !monitor.IsHealthy() {
		t.Error("expected monitor to be healthy")
	}
}

func TestMonitor_GetLastError(t *testing.T) {
	client := &Client{
		closed: false,
	}

	monitor := NewMonitor(client, time.Minute)

	// Initially should have no error
	if monitor.GetLastError() != nil {
		t.Error("expected no error initially")
	}

	// Set an error
	testErr := ErrHealthCheckFailed
	monitor.mu.Lock()
	monitor.status.LastError = testErr
	monitor.mu.Unlock()

	if monitor.GetLastError() != testErr {
		t.Errorf("expected error %v, got %v", testErr, monitor.GetLastError())
	}
}

func TestMonitor_GetStatus(t *testing.T) {
	client := &Client{
		closed: false,
	}

	monitor := NewMonitor(client, time.Minute)

	status := monitor.GetStatus()

	if status.Healthy {
		t.Error("expected status to be unhealthy initially")
	}

	if status.CheckCount != 0 {
		t.Errorf("expected CheckCount 0, got %d", status.CheckCount)
	}

	if status.ErrorCount != 0 {
		t.Errorf("expected ErrorCount 0, got %d", status.ErrorCount)
	}

	if status.SuccessCount != 0 {
		t.Errorf("expected SuccessCount 0, got %d", status.SuccessCount)
	}
}

func TestMonitor_GetMetrics(t *testing.T) {
	client := &Client{
		closed: false,
	}

	monitor := NewMonitor(client, time.Minute)

	// Set some test data
	monitor.mu.Lock()
	monitor.status.Healthy = true
	monitor.status.CheckCount = 100
	monitor.status.SuccessCount = 95
	monitor.status.ErrorCount = 5
	monitor.status.ResponseTime = 50 * time.Millisecond
	monitor.status.TotalConns = 10
	monitor.status.IdleConns = 5
	monitor.status.LastCheck = time.Now()
	monitor.mu.Unlock()

	metrics := monitor.GetMetrics()

	if metrics["healthy"] != true {
		t.Error("expected healthy to be true")
	}

	if metrics["check_count"] != uint64(100) {
		t.Errorf("expected check_count 100, got %v", metrics["check_count"])
	}

	if metrics["success_count"] != uint64(95) {
		t.Errorf("expected success_count 95, got %v", metrics["success_count"])
	}

	if metrics["error_count"] != uint64(5) {
		t.Errorf("expected error_count 5, got %v", metrics["error_count"])
	}

	if metrics["response_time_ms"] != int64(50) {
		t.Errorf("expected response_time_ms 50, got %v", metrics["response_time_ms"])
	}

	if metrics["total_connections"] != uint32(10) {
		t.Errorf("expected total_connections 10, got %v", metrics["total_connections"])
	}

	if metrics["idle_connections"] != uint32(5) {
		t.Errorf("expected idle_connections 5, got %v", metrics["idle_connections"])
	}

	// Check success_rate calculation
	successRate := metrics["success_rate"]
	if successRate != "95.00%" {
		t.Errorf("expected success_rate 95.00%%, got %v", successRate)
	}
}

func TestMonitor_GetDetailedStatus(t *testing.T) {
	client := &Client{
		closed: false,
	}

	monitor := NewMonitor(client, time.Minute)

	// Set some test data
	monitor.mu.Lock()
	monitor.status.Healthy = true
	monitor.status.CheckCount = 100
	monitor.status.SuccessCount = 95
	monitor.status.ErrorCount = 5
	monitor.status.ResponseTime = 50 * time.Millisecond
	monitor.status.TotalConns = 10
	monitor.status.IdleConns = 5
	monitor.status.LastCheck = time.Now()
	monitor.mu.Unlock()

	status := monitor.GetDetailedStatus()

	if status == "" {
		t.Error("expected non-empty status string")
	}

	// Check for key elements in the status string
	if len(status) < 100 {
		t.Error("expected detailed status to be comprehensive")
	}
}

func TestMonitor_StartStop(t *testing.T) {
	client := &Client{
		closed: true, // Closed client to avoid connection attempts
	}

	monitor := NewMonitor(client, 100*time.Millisecond)

	ctx := context.Background()

	// Start monitoring
	monitor.Start(ctx)

	// Let it run for a short time
	time.Sleep(50 * time.Millisecond)

	// Stop monitoring
	done := make(chan struct{})
	go func() {
		monitor.Stop()
		close(done)
	}()

	// Wait for stop to complete with timeout
	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Error("monitor did not stop within timeout")
	}
}

func TestMonitor_PerformHealthCheck_ClosedClient(t *testing.T) {
	client := &Client{
		closed: true,
	}

	monitor := NewMonitor(client, time.Minute)

	ctx := context.Background()

	// Perform health check on closed client
	monitor.performHealthCheck(ctx)

	status := monitor.GetStatus()

	// Should have performed a check
	if status.CheckCount != 1 {
		t.Errorf("expected CheckCount 1, got %d", status.CheckCount)
	}

	// Should have failed
	if status.Healthy {
		t.Error("expected health check to fail for closed client")
	}

	// Should have an error
	if status.LastError == nil {
		t.Error("expected an error for closed client")
	}

	// Should increment error count
	if status.ErrorCount != 1 {
		t.Errorf("expected ErrorCount 1, got %d", status.ErrorCount)
	}

	// Success count should be 0
	if status.SuccessCount != 0 {
		t.Errorf("expected SuccessCount 0, got %d", status.SuccessCount)
	}
}
