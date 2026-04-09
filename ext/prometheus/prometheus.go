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
// All fields are unexported; use the provided methods to read or record data.
type Metrics struct {
	// query counters
	queryTotal   int64
	queryErrors  int64
	querySuccess int64
	createTotal  int64
	createErrors int64
	updateTotal  int64
	updateErrors int64
	deleteTotal  int64
	deleteErrors int64

	// latency tracking (capped at maxLatencySamples to prevent unbounded growth)
	queryLatencies  []time.Duration
	createLatencies []time.Duration
	updateLatencies []time.Duration
	deleteLatencies []time.Duration

	// cache metrics
	cacheHits   int64
	cacheMisses int64

	mu sync.RWMutex
}

// New creates a new Metrics instance.
func New() *Metrics {
	return &Metrics{
		queryLatencies:  make([]time.Duration, 0),
		createLatencies: make([]time.Duration, 0),
		updateLatencies: make([]time.Duration, 0),
		deleteLatencies: make([]time.Duration, 0),
	}
}

// RecordQuery records a query execution.
func (m *Metrics) RecordQuery(latency time.Duration, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.queryTotal++
	if len(m.queryLatencies) >= maxLatencySamples {
		// Drop oldest half to bound memory usage while preserving recent samples.
		copy(m.queryLatencies, m.queryLatencies[maxLatencySamples/2:])
		m.queryLatencies = m.queryLatencies[:maxLatencySamples/2]
	}
	m.queryLatencies = append(m.queryLatencies, latency)

	if err != nil {
		m.queryErrors++
	} else {
		m.querySuccess++
	}
}

// RecordCreate records a create operation.
func (m *Metrics) RecordCreate(latency time.Duration, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.createTotal++
	if len(m.createLatencies) >= maxLatencySamples {
		copy(m.createLatencies, m.createLatencies[maxLatencySamples/2:])
		m.createLatencies = m.createLatencies[:maxLatencySamples/2]
	}
	m.createLatencies = append(m.createLatencies, latency)

	if err != nil {
		m.createErrors++
	}
}

// RecordUpdate records an update operation.
func (m *Metrics) RecordUpdate(latency time.Duration, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.updateTotal++
	if len(m.updateLatencies) >= maxLatencySamples {
		copy(m.updateLatencies, m.updateLatencies[maxLatencySamples/2:])
		m.updateLatencies = m.updateLatencies[:maxLatencySamples/2]
	}
	m.updateLatencies = append(m.updateLatencies, latency)

	if err != nil {
		m.updateErrors++
	}
}

// RecordDelete records a delete operation.
func (m *Metrics) RecordDelete(latency time.Duration, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.deleteTotal++
	if len(m.deleteLatencies) >= maxLatencySamples {
		copy(m.deleteLatencies, m.deleteLatencies[maxLatencySamples/2:])
		m.deleteLatencies = m.deleteLatencies[:maxLatencySamples/2]
	}
	m.deleteLatencies = append(m.deleteLatencies, latency)

	if err != nil {
		m.deleteErrors++
	}
}

// RecordCacheHit increments cache hit counter.
func (m *Metrics) RecordCacheHit() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cacheHits++
}

// RecordCacheMiss increments cache miss counter.
func (m *Metrics) RecordCacheMiss() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cacheMisses++
}

// GetQueryCount returns total query count.
func (m *Metrics) GetQueryCount() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.queryTotal
}

// GetErrorCount returns total error count.
func (m *Metrics) GetErrorCount() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.queryErrors + m.createErrors + m.updateErrors + m.deleteErrors
}

// GetCacheHitRate returns cache hit rate (0-1).
func (m *Metrics) GetCacheHitRate() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total := m.cacheHits + m.cacheMisses
	if total == 0 {
		return 0
	}
	return float64(m.cacheHits) / float64(total)
}

// GetAverageQueryLatency returns average query latency.
func (m *Metrics) GetAverageQueryLatency() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.queryLatencies) == 0 {
		return 0
	}

	total := time.Duration(0)
	for _, lat := range m.queryLatencies {
		total += lat
	}
	return total / time.Duration(len(m.queryLatencies))
}

// GetStats returns a summary of all metrics.
func (m *Metrics) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := map[string]interface{}{
		"queries": map[string]int64{
			"total":   m.queryTotal,
			"success": m.querySuccess,
			"errors":  m.queryErrors,
		},
		"operations": map[string]int64{
			"create":        m.createTotal,
			"create_errors": m.createErrors,
			"update":        m.updateTotal,
			"update_errors": m.updateErrors,
			"delete":        m.deleteTotal,
			"delete_errors": m.deleteErrors,
		},
		"cache": map[string]interface{}{
			"hits":   m.cacheHits,
			"misses": m.cacheMisses,
		},
	}
	return stats
}

// GetQueryLatencyCount returns the number of query latency samples currently retained.
func (m *Metrics) GetQueryLatencyCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.queryLatencies)
}

// GetCreateLatencyCount returns the number of create latency samples currently retained.
func (m *Metrics) GetCreateLatencyCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.createLatencies)
}

// GetUpdateLatencyCount returns the number of update latency samples currently retained.
func (m *Metrics) GetUpdateLatencyCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.updateLatencies)
}

// GetDeleteLatencyCount returns the number of delete latency samples currently retained.
func (m *Metrics) GetDeleteLatencyCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.deleteLatencies)
}

// Reset clears all metrics.
func (m *Metrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.queryTotal = 0
	m.queryErrors = 0
	m.querySuccess = 0
	m.createTotal = 0
	m.createErrors = 0
	m.updateTotal = 0
	m.updateErrors = 0
	m.deleteTotal = 0
	m.deleteErrors = 0
	m.cacheHits = 0
	m.cacheMisses = 0
	m.queryLatencies = make([]time.Duration, 0)
	m.createLatencies = make([]time.Duration, 0)
	m.updateLatencies = make([]time.Duration, 0)
	m.deleteLatencies = make([]time.Duration, 0)
}
