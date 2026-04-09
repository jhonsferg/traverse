// Package prometheus provides Prometheus metrics integration for traverse OData client.
// It tracks query operations, errors, and latency.
package prometheus

import (
	"sync"
	"time"
)

// maxLatencySamples is the maximum number of latency samples retained per
// operation type. Once the cap is reached the oldest half is discarded,
// keeping memory usage bounded while preserving recent statistics.
const maxLatencySamples = 10_000

// Metrics holds all Prometheus metrics for traverse operations.
type Metrics struct {
	// Query counters
	QueryTotal   int64 // Total queries executed
	QueryErrors  int64 // Total query errors
	QuerySuccess int64 // Total successful queries
	CreateTotal  int64 // Total create operations
	CreateErrors int64 // Total create errors
	UpdateTotal  int64 // Total update operations
	UpdateErrors int64 // Total update errors
	DeleteTotal  int64 // Total delete operations
	DeleteErrors int64 // Total delete errors

	// Latency tracking (capped at maxLatencySamples to prevent unbounded growth)
	QueryLatencies  []time.Duration // Query latencies (for histogram)
	CreateLatencies []time.Duration // Create latencies
	UpdateLatencies []time.Duration // Update latencies
	DeleteLatencies []time.Duration // Delete latencies

	// Cache metrics
	CacheHits   int64 // Metadata cache hits
	CacheMisses int64 // Metadata cache misses

	mu sync.RWMutex
}

// New creates a new Metrics instance.
func New() *Metrics {
	return &Metrics{
		QueryLatencies:  make([]time.Duration, 0),
		CreateLatencies: make([]time.Duration, 0),
		UpdateLatencies: make([]time.Duration, 0),
		DeleteLatencies: make([]time.Duration, 0),
	}
}

// RecordQuery records a query execution.
func (m *Metrics) RecordQuery(latency time.Duration, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.QueryTotal++
	if len(m.QueryLatencies) >= maxLatencySamples {
		// Drop oldest half to bound memory usage while preserving recent samples.
		copy(m.QueryLatencies, m.QueryLatencies[maxLatencySamples/2:])
		m.QueryLatencies = m.QueryLatencies[:maxLatencySamples/2]
	}
	m.QueryLatencies = append(m.QueryLatencies, latency)

	if err != nil {
		m.QueryErrors++
	} else {
		m.QuerySuccess++
	}
}

// RecordCreate records a create operation.
func (m *Metrics) RecordCreate(latency time.Duration, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.CreateTotal++
	if len(m.CreateLatencies) >= maxLatencySamples {
		copy(m.CreateLatencies, m.CreateLatencies[maxLatencySamples/2:])
		m.CreateLatencies = m.CreateLatencies[:maxLatencySamples/2]
	}
	m.CreateLatencies = append(m.CreateLatencies, latency)

	if err != nil {
		m.CreateErrors++
	}
}

// RecordUpdate records an update operation.
func (m *Metrics) RecordUpdate(latency time.Duration, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.UpdateTotal++
	if len(m.UpdateLatencies) >= maxLatencySamples {
		copy(m.UpdateLatencies, m.UpdateLatencies[maxLatencySamples/2:])
		m.UpdateLatencies = m.UpdateLatencies[:maxLatencySamples/2]
	}
	m.UpdateLatencies = append(m.UpdateLatencies, latency)

	if err != nil {
		m.UpdateErrors++
	}
}

// RecordDelete records a delete operation.
func (m *Metrics) RecordDelete(latency time.Duration, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.DeleteTotal++
	if len(m.DeleteLatencies) >= maxLatencySamples {
		copy(m.DeleteLatencies, m.DeleteLatencies[maxLatencySamples/2:])
		m.DeleteLatencies = m.DeleteLatencies[:maxLatencySamples/2]
	}
	m.DeleteLatencies = append(m.DeleteLatencies, latency)

	if err != nil {
		m.DeleteErrors++
	}
}

// RecordCacheHit increments cache hit counter.
func (m *Metrics) RecordCacheHit() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CacheHits++
}

// RecordCacheMiss increments cache miss counter.
func (m *Metrics) RecordCacheMiss() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CacheMisses++
}

// GetQueryCount returns total query count.
func (m *Metrics) GetQueryCount() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.QueryTotal
}

// GetErrorCount returns total error count.
func (m *Metrics) GetErrorCount() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.QueryErrors + m.CreateErrors + m.UpdateErrors + m.DeleteErrors
}

// GetCacheHitRate returns cache hit rate (0-1).
func (m *Metrics) GetCacheHitRate() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total := m.CacheHits + m.CacheMisses
	if total == 0 {
		return 0
	}
	return float64(m.CacheHits) / float64(total)
}

// GetAverageQueryLatency returns average query latency.
func (m *Metrics) GetAverageQueryLatency() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.QueryLatencies) == 0 {
		return 0
	}

	total := time.Duration(0)
	for _, lat := range m.QueryLatencies {
		total += lat
	}
	return total / time.Duration(len(m.QueryLatencies))
}

// GetStats returns a summary of all metrics.
func (m *Metrics) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := map[string]interface{}{
		"queries": map[string]int64{
			"total":   m.QueryTotal,
			"success": m.QuerySuccess,
			"errors":  m.QueryErrors,
		},
		"operations": map[string]int64{
			"create":        m.CreateTotal,
			"create_errors": m.CreateErrors,
			"update":        m.UpdateTotal,
			"update_errors": m.UpdateErrors,
			"delete":        m.DeleteTotal,
			"delete_errors": m.DeleteErrors,
		},
		"cache": map[string]interface{}{
			"hits":   m.CacheHits,
			"misses": m.CacheMisses,
		},
	}
	return stats
}

// Reset clears all metrics.
func (m *Metrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.QueryTotal = 0
	m.QueryErrors = 0
	m.QuerySuccess = 0
	m.CreateTotal = 0
	m.CreateErrors = 0
	m.UpdateTotal = 0
	m.UpdateErrors = 0
	m.DeleteTotal = 0
	m.DeleteErrors = 0
	m.CacheHits = 0
	m.CacheMisses = 0
	m.QueryLatencies = make([]time.Duration, 0)
	m.CreateLatencies = make([]time.Duration, 0)
	m.UpdateLatencies = make([]time.Duration, 0)
	m.DeleteLatencies = make([]time.Duration, 0)
}
