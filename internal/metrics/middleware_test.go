package metrics

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestMiddleware_StatusCapture(t *testing.T) {
	tests := []struct {
		name        string
		status      int
		wantSuccess bool
	}{
		{"200 OK", http.StatusOK, true},
		{"404 Not Found", http.StatusNotFound, false},
		{"500 Internal", http.StatusInternalServerError, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Metrics{}
			old := global
			global = m
			t.Cleanup(func() { global = old })

			handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
			}))

			req := httptest.NewRequest(http.MethodGet, "/api/moods", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if got := atomic.LoadUint64(&m.requestsTotal); got != 1 {
				t.Errorf("requestsTotal = %d, want 1", got)
			}

			if tt.wantSuccess {
				if got := atomic.LoadUint64(&m.requestsSuccess); got != 1 {
					t.Errorf("requestsSuccess = %d, want 1", got)
				}
			} else {
				if got := atomic.LoadUint64(&m.requestsError); got != 1 {
					t.Errorf("requestsError = %d, want 1", got)
				}
			}
		})
	}
}

func TestMiddleware_LatencyRecorded(t *testing.T) {
	m := &Metrics{}
	old := global
	global = m
	t.Cleanup(func() { global = old })

	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/moods", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.latencyCount != 1 {
		t.Errorf("latencyCount = %d, want 1", m.latencyCount)
	}
	if m.latencySum <= 0 {
		t.Error("latencySum should be > 0")
	}
}

func TestMiddleware_SkipsProbes(t *testing.T) {
	for _, path := range []string{"/health", "/ready"} {
		t.Run(path, func(t *testing.T) {
			m := &Metrics{}
			old := global
			global = m
			t.Cleanup(func() { global = old })

			handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if got := atomic.LoadUint64(&m.requestsTotal); got != 0 {
				t.Errorf("requestsTotal = %d for %s, want 0 (should be skipped)", got, path)
			}
		})
	}
}
