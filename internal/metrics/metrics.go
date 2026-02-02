package metrics

import (
	"sync"
	"sync/atomic"
	"time"
)

// Metrics holds application runtime metrics
type Metrics struct {
	startTime time.Time

	// Request counters
	requestsTotal   uint64
	requestsSuccess uint64
	requestsError   uint64

	// Audio metrics
	playsTotal uint64

	// Latency tracking
	mu           sync.RWMutex
	latencySum   time.Duration
	latencyCount uint64
}

// Global metrics instance
var global = &Metrics{
	startTime: time.Now(),
}

// Get returns the global metrics instance
func Get() *Metrics {
	return global
}

// RecordRequest records a request with status and latency
func (m *Metrics) RecordRequest(status int, latency time.Duration) {
	atomic.AddUint64(&m.requestsTotal, 1)
	if status >= 200 && status < 400 {
		atomic.AddUint64(&m.requestsSuccess, 1)
	} else if status >= 400 {
		atomic.AddUint64(&m.requestsError, 1)
	}

	m.mu.Lock()
	m.latencySum += latency
	m.latencyCount++
	m.mu.Unlock()
}

// RecordPlay records an audio play event
func (m *Metrics) RecordPlay() {
	atomic.AddUint64(&m.playsTotal, 1)
}

// Snapshot returns current metrics as a map
func (m *Metrics) Snapshot() map[string]any {
	m.mu.RLock()
	avgLatency := float64(0)
	if m.latencyCount > 0 {
		avgLatency = float64(m.latencySum.Milliseconds()) / float64(m.latencyCount)
	}
	m.mu.RUnlock()

	return map[string]any{
		"uptime_seconds":   time.Since(m.startTime).Seconds(),
		"requests_total":   atomic.LoadUint64(&m.requestsTotal),
		"requests_success": atomic.LoadUint64(&m.requestsSuccess),
		"requests_error":   atomic.LoadUint64(&m.requestsError),
		"plays_total":      atomic.LoadUint64(&m.playsTotal),
		"avg_latency_ms":   avgLatency,
	}
}
