package metrics

import (
	"log"
	"net/http"
	"path"
	"strings"
	"time"
)

// responseWriter wraps http.ResponseWriter to capture status code and bytes written.
type responseWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytes += n
	return n, err
}

// clientIP extracts the client IP from X-Forwarded-For (set by Caddy) or falls back to RemoteAddr.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For may contain multiple IPs; first is the client
		if ip, _, ok := strings.Cut(xff, ","); ok {
			return strings.TrimSpace(ip)
		}
		return strings.TrimSpace(xff)
	}
	// RemoteAddr is "ip:port", strip the port
	if idx := strings.LastIndex(r.RemoteAddr, ":"); idx != -1 {
		return r.RemoteAddr[:idx]
	}
	return r.RemoteAddr
}

// staticExts are file extensions to skip in access logs.
var staticExts = map[string]bool{
	".css": true, ".js": true, ".png": true, ".jpg": true,
	".jpeg": true, ".gif": true, ".svg": true, ".ico": true,
	".woff": true, ".woff2": true, ".ttf": true, ".eot": true,
	".map": true, ".webp": true, ".mp3": true, ".webm": true,
}

// skipLog returns true for health probes and static asset requests.
func skipLog(p string) bool {
	if p == "/health" || p == "/ready" {
		return true
	}
	return staticExts[strings.ToLower(path.Ext(p))]
}

// Middleware records request latency and status for every request
// except health/readiness probes (which skew metrics).
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip high-frequency probes
		if r.URL.Path == "/health" || r.URL.Path == "/ready" {
			next.ServeHTTP(w, r)
			return
		}

		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()

		next.ServeHTTP(rw, r)

		duration := time.Since(start)
		Get().RecordRequest(rw.status, duration)

		if skipLog(r.URL.Path) {
			return
		}

		// Access log: remote_ip method path status bytes latency user_agent
		log.Printf("%s %s %s %d %d %.3fms %q",
			clientIP(r), r.Method, r.URL.RequestURI(),
			rw.status, rw.bytes,
			float64(duration.Microseconds())/1000.0,
			r.UserAgent(),
		)
	})
}
