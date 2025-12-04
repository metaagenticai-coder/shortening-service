package database

import (
	"context"
	"sync"
	"time"
)

// HealthStatus represents the health status of the database
type HealthStatus struct {
	Healthy         bool
	LastChecked     time.Time
	LastError       error
	ConsecutiveFails int
	ResponseTime    time.Duration
}

// PoolStats represents statistics about the connection pool
type PoolStats struct {
	AcquireCount          int64
	AcquireDuration       time.Duration
	AcquiredConns         int32
	CanceledAcquireCount  int64
	ConstructingConns     int32
	EmptyAcquireCount     int64
	IdleConns             int32
	MaxConns              int32
	TotalConns            int32
	NewConnsCount         int64
	MaxLifetimeDestroyCount int64
	MaxIdleDestroyCount    int64
}

// Monitor provides health monitoring for a database connection pool
type Monitor struct {
	pool          *Pool
	healthStatus  HealthStatus
	mu            sync.RWMutex
	stopCh        chan struct{}
	stopped       bool
	checkInterval time.Duration
}

// NewMonitor creates a new database monitor
func NewMonitor(pool *Pool, checkInterval time.Duration) *Monitor {
	if checkInterval == 0 {
		checkInterval = 30 * time.Second
	}

	return &Monitor{
		pool:          pool,
		checkInterval: checkInterval,
		stopCh:        make(chan struct{}),
		healthStatus: HealthStatus{
			Healthy:     true,
			LastChecked: time.Now(),
		},
	}
}

// Start begins periodic health checks
func (m *Monitor) Start(ctx context.Context) {
	go m.runHealthChecks(ctx)
}

// Stop stops the health check routine
func (m *Monitor) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.stopped {
		close(m.stopCh)
		m.stopped = true
	}
}

// runHealthChecks performs periodic health checks
func (m *Monitor) runHealthChecks(ctx context.Context) {
	ticker := time.NewTicker(m.checkInterval)
	defer ticker.Stop()

	// Perform initial health check
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

// performHealthCheck executes a single health check
func (m *Monitor) performHealthCheck(ctx context.Context) {
	start := time.Now()

	// Create a timeout context for the health check
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := m.pool.HealthCheck(checkCtx)
	responseTime := time.Since(start)

	m.mu.Lock()
	defer m.mu.Unlock()

	m.healthStatus.LastChecked = time.Now()
	m.healthStatus.ResponseTime = responseTime

	if err != nil {
		m.healthStatus.Healthy = false
		m.healthStatus.LastError = err
		m.healthStatus.ConsecutiveFails++
	} else {
		m.healthStatus.Healthy = true
		m.healthStatus.LastError = nil
		m.healthStatus.ConsecutiveFails = 0
	}
}

// GetHealthStatus returns the current health status
func (m *Monitor) GetHealthStatus() HealthStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to prevent external modification
	return m.healthStatus
}

// IsHealthy returns true if the database is healthy
func (m *Monitor) IsHealthy() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.healthStatus.Healthy
}

// GetPoolStats returns current connection pool statistics
func (m *Monitor) GetPoolStats() *PoolStats {
	stat := m.pool.Stats()
	if stat == nil {
		return nil
	}

	return &PoolStats{
		AcquireCount:            stat.AcquireCount(),
		AcquireDuration:         stat.AcquireDuration(),
		AcquiredConns:           stat.AcquiredConns(),
		CanceledAcquireCount:    stat.CanceledAcquireCount(),
		ConstructingConns:       stat.ConstructingConns(),
		EmptyAcquireCount:       stat.EmptyAcquireCount(),
		IdleConns:               stat.IdleConns(),
		MaxConns:                stat.MaxConns(),
		TotalConns:              stat.TotalConns(),
		NewConnsCount:           stat.NewConnsCount(),
		MaxLifetimeDestroyCount: stat.MaxLifetimeDestroyCount(),
		MaxIdleDestroyCount:     stat.MaxIdleDestroyCount(),
	}
}

// CheckNow performs an immediate health check
func (m *Monitor) CheckNow(ctx context.Context) error {
	m.performHealthCheck(ctx)
	return m.healthStatus.LastError
}
