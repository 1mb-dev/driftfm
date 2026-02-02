package metrics

import (
	"sync"
	"testing"
	"time"
)

func TestRecordRequest(t *testing.T) {
	m := &Metrics{startTime: time.Now()}

	// Record some requests
	m.RecordRequest(200, 100*time.Millisecond)
	m.RecordRequest(201, 50*time.Millisecond)
	m.RecordRequest(404, 10*time.Millisecond)
	m.RecordRequest(500, 200*time.Millisecond)

	snap := m.Snapshot()

	if snap["requests_total"].(uint64) != 4 {
		t.Errorf("expected 4 total requests, got %v", snap["requests_total"])
	}
	if snap["requests_success"].(uint64) != 2 {
		t.Errorf("expected 2 success requests, got %v", snap["requests_success"])
	}
	if snap["requests_error"].(uint64) != 2 {
		t.Errorf("expected 2 error requests, got %v", snap["requests_error"])
	}
}

func TestRecordPlay(t *testing.T) {
	m := &Metrics{startTime: time.Now()}

	m.RecordPlay()
	m.RecordPlay()
	m.RecordPlay()

	snap := m.Snapshot()

	if snap["plays_total"].(uint64) != 3 {
		t.Errorf("expected 3 plays, got %v", snap["plays_total"])
	}
}

func TestLatencyAverage(t *testing.T) {
	m := &Metrics{startTime: time.Now()}

	m.RecordRequest(200, 100*time.Millisecond)
	m.RecordRequest(200, 200*time.Millisecond)
	m.RecordRequest(200, 300*time.Millisecond)

	snap := m.Snapshot()

	// Average should be 200ms
	avgLatency := snap["avg_latency_ms"].(float64)
	if avgLatency < 199 || avgLatency > 201 {
		t.Errorf("expected ~200ms average latency, got %v", avgLatency)
	}
}

func TestConcurrentAccess(t *testing.T) {
	m := &Metrics{startTime: time.Now()}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			m.RecordRequest(200, 10*time.Millisecond)
		}()
		go func() {
			defer wg.Done()
			m.RecordPlay()
		}()
	}
	wg.Wait()

	snap := m.Snapshot()

	if snap["requests_total"].(uint64) != 100 {
		t.Errorf("expected 100 requests, got %v", snap["requests_total"])
	}
	if snap["plays_total"].(uint64) != 100 {
		t.Errorf("expected 100 plays, got %v", snap["plays_total"])
	}
}
