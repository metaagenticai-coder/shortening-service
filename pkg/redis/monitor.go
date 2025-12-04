package redis

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// HealthStatus represents the health status of the Redis connection
type HealthStatus struct {
	Healthy       bool
	LastCheck     time.Time
	LastError     error
	ResponseTime  time.Duration
	TotalConns    uint32
	IdleConns     uint32
	StaleConns    uint32
	CheckCount    uint64
	ErrorCount    uint64
	SuccessCount  uint64
	LastSuccessAt time.Time
	LastErrorAt   time.Time
}

// Monitor provides health monitoring for a Redis client
type Monitor struct {
	client   *Client
	status   HealthStatus
	mu       sync.RWMutex
	interval time.Duration
	stopCh   chan struct{}
	stoppedCh chan struct{}
}

// NewMonitor creates a new Redis monitor with the given check interval
func NewMonitor(client *Client, interval time.Duration) *Monitor {
	if interval == 0 {
		interval = 30 * time.Second // Default interval
	}

	return &Monitor{
		client:   client,
		interval: interval,
		stopCh:   make(chan struct{}),
		stoppedCh: make(chan struct{}),
		status: HealthStatus{
			Healthy:   false,
			LastCheck: time.Time{},
		},
	}
}

// Start begins monitoring the Redis connection health
func (m *Monitor) Start(ctx context.Context) {
	go m.monitorLoop(ctx)
}

// Stop stops the health monitoring
func (m *Monitor) Stop() {
	close(m.stopCh)
	<-m.stoppedCh
}

// GetStatus returns the current health status
func (m *Monitor) GetStatus() HealthStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.status
}

// IsHealthy returns true if the last health check was successful
func (m *Monitor) IsHealthy() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.status.Healthy
}

// GetLastError returns the last error encountered
func (m *Monitor) GetLastError() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.status.LastError
}

// monitorLoop runs the periodic health check loop
func (m *Monitor) monitorLoop(ctx context.Context) {
	defer close(m.stoppedCh)

	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	// Perform initial check
	m.performHealthCheck(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.performHealthCheck(ctx)
		}
	}
}

// performHealthCheck executes a health check and updates the status
func (m *Monitor) performHealthCheck(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()

	start := time.Now()
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := m.client.HealthCheck(checkCtx)
	elapsed := time.Since(start)

	m.status.CheckCount++
	m.status.LastCheck = time.Now()
	m.status.ResponseTime = elapsed

	if err != nil {
		m.status.Healthy = false
		m.status.LastError = err
		m.status.ErrorCount++
		m.status.LastErrorAt = time.Now()
	} else {
		m.status.Healthy = true
		m.status.LastError = nil
		m.status.SuccessCount++
		m.status.LastSuccessAt = time.Now()

		// Update connection pool stats
		if stats := m.client.PoolStats(); stats != nil {
			m.status.TotalConns = stats.TotalConns
			m.status.IdleConns = stats.IdleConns
			m.status.StaleConns = stats.StaleConns
		}
	}
}

// GetMetrics returns formatted metrics for monitoring systems
func (m *Monitor) GetMetrics() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	uptime := time.Since(m.status.LastCheck)
	if m.status.CheckCount == 0 {
		uptime = 0
	}

	successRate := float64(0)
	if m.status.CheckCount > 0 {
		successRate = float64(m.status.SuccessCount) / float64(m.status.CheckCount) * 100
	}

	return map[string]interface{}{
		"healthy":              m.status.Healthy,
		"last_check":           m.status.LastCheck.Unix(),
		"last_check_formatted": m.status.LastCheck.Format(time.RFC3339),
		"response_time_ms":     m.status.ResponseTime.Milliseconds(),
		"check_count":          m.status.CheckCount,
		"error_count":          m.status.ErrorCount,
		"success_count":        m.status.SuccessCount,
		"success_rate":         fmt.Sprintf("%.2f%%", successRate),
		"total_connections":    m.status.TotalConns,
		"idle_connections":     m.status.IdleConns,
		"stale_connections":    m.status.StaleConns,
		"uptime_seconds":       uptime.Seconds(),
	}
}

// GetDetailedStatus returns a human-readable status report
func (m *Monitor) GetDetailedStatus() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := "Redis Health Status\n"
	status += "==================\n\n"

	if m.status.Healthy {
		status += "Status: HEALTHY ✓\n"
	} else {
		status += "Status: UNHEALTHY ✗\n"
	}

	status += fmt.Sprintf("Last Check: %s\n", m.status.LastCheck.Format(time.RFC3339))
	status += fmt.Sprintf("Response Time: %s\n", m.status.ResponseTime)
	status += fmt.Sprintf("Total Checks: %d\n", m.status.CheckCount)
	status += fmt.Sprintf("Success Count: %d\n", m.status.SuccessCount)
	status += fmt.Sprintf("Error Count: %d\n", m.status.ErrorCount)

	if m.status.CheckCount > 0 {
		successRate := float64(m.status.SuccessCount) / float64(m.status.CheckCount) * 100
		status += fmt.Sprintf("Success Rate: %.2f%%\n", successRate)
	}

	status += fmt.Sprintf("\nConnection Pool:\n")
	status += fmt.Sprintf("  Total Connections: %d\n", m.status.TotalConns)
	status += fmt.Sprintf("  Idle Connections: %d\n", m.status.IdleConns)
	status += fmt.Sprintf("  Stale Connections: %d\n", m.status.StaleConns)

	if m.status.LastError != nil {
		status += fmt.Sprintf("\nLast Error: %s\n", m.status.LastError.Error())
		status += fmt.Sprintf("Last Error At: %s\n", m.status.LastErrorAt.Format(time.RFC3339))
	}

	if !m.status.LastSuccessAt.IsZero() {
		status += fmt.Sprintf("\nLast Success At: %s\n", m.status.LastSuccessAt.Format(time.RFC3339))
	}

	return status
}
