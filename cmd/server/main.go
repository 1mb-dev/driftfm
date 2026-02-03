package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/1mb-dev/driftfm/internal/api"
	"github.com/1mb-dev/driftfm/internal/audio"
	"github.com/1mb-dev/driftfm/internal/cache"
	"github.com/1mb-dev/driftfm/internal/config"
	"github.com/1mb-dev/driftfm/internal/inventory"
	"github.com/1mb-dev/driftfm/internal/metrics"
	"github.com/1mb-dev/driftfm/internal/radio"
)

// version is set at build time via -ldflags "-X main.version=..."
var version = "dev"

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// Load configuration: defaults → config.yaml → config.local.yaml → env vars
	cfg, err := config.Load("config.yaml", "config.local.yaml")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize repository
	repo, err := inventory.NewRepository(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("failed to initialize repository: %w", err)
	}
	defer func() {
		if err := repo.Close(); err != nil {
			log.Printf("Error closing repository: %v", err)
		}
	}()

	// Initialize audio resolver
	audioResolver := audio.NewResolver(cfg.Audio.LocalPath)

	// Initialize cache
	appCache, err := cache.New()
	if err != nil {
		return fmt.Errorf("failed to initialize cache: %w", err)
	}
	defer func() {
		if err := appCache.Close(); err != nil {
			log.Printf("Error closing cache: %v", err)
		}
	}()

	// Create radio manager and API handler
	radioMgr := radio.NewManager(repo)
	handler := api.NewHandler(repo, radioMgr, audioResolver, appCache)

	// Create mux
	mux := http.NewServeMux()

	// Health check (liveness probe)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("ok " + version)); err != nil {
			log.Printf("Error writing health response: %v", err)
		}
	})

	// Readiness check (verifies database connectivity)
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := repo.Ping(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			if _, err := w.Write([]byte("not ready")); err != nil {
				log.Printf("Error writing ready response: %v", err)
			}
			return
		}
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("ready")); err != nil {
			log.Printf("Error writing ready response: %v", err)
		}
	})

	// Metrics endpoint (runtime + application stats) — localhost only
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		host, _, _ := net.SplitHostPort(r.RemoteAddr)
		if host != "127.0.0.1" && host != "::1" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		var mem runtime.MemStats
		runtime.ReadMemStats(&mem)

		output := map[string]any{
			"version": version,
			"runtime": map[string]any{
				"goroutines":        runtime.NumGoroutine(),
				"memory_alloc_mb":   float64(mem.Alloc) / 1024 / 1024,
				"memory_sys_mb":     float64(mem.Sys) / 1024 / 1024,
				"gc_runs":           mem.NumGC,
				"gc_pause_total_ms": float64(mem.PauseTotalNs) / 1e6,
			},
			"app":   metrics.Get().Snapshot(),
			"cache": appCache.Stats(),
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(output); err != nil {
			log.Printf("Error encoding metrics: %v", err)
		}
	})

	// Register API routes
	handler.RegisterRoutes(mux)

	// Serve static files from web/
	webFS := http.FileServer(http.Dir("web"))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Root path and paths with file extensions: serve normally via FileServer
		if r.URL.Path == "/" || path.Ext(r.URL.Path) != "" {
			webFS.ServeHTTP(w, r)
			return
		}
		// Extensionless paths: check if file exists on disk, else 404
		cleanPath := filepath.Clean(filepath.Join("web", filepath.FromSlash(r.URL.Path)))
		// Prevent path traversal: ensure resolved path stays under web/
		if strings.HasPrefix(cleanPath, "web"+string(filepath.Separator)) {
			if _, err := os.Stat(cleanPath); err == nil {
				webFS.ServeHTTP(w, r)
				return
			}
		}
		http.NotFound(w, r)
	})

	// Serve audio files from local directory
	audioFS := http.FileServer(http.Dir(cfg.Audio.LocalPath))
	mux.Handle("/audio/", http.StripPrefix("/audio/", audioFS))

	// Get parsed timeouts (validated during config.Load, errors should not occur)
	readTimeout, err := cfg.GetReadTimeout()
	if err != nil {
		return fmt.Errorf("invalid read timeout: %w", err)
	}
	writeTimeout, err := cfg.GetWriteTimeout()
	if err != nil {
		return fmt.Errorf("invalid write timeout: %w", err)
	}
	shutdownTimeout, err := cfg.GetShutdownTimeout()
	if err != nil {
		return fmt.Errorf("invalid shutdown timeout: %w", err)
	}

	// Create server with production timeouts
	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:           securityHeaders(metrics.Middleware(mux)),
		ReadTimeout:       readTimeout,
		ReadHeaderTimeout: readTimeout / 3,
		WriteTimeout:      writeTimeout * 4, // Long for potential audio streaming
		IdleTimeout:       writeTimeout * 8,
		MaxHeaderBytes:    1 << 20, // 1 MB
	}

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		log.Printf("Drift FM %s starting on http://localhost:%d", version, cfg.Server.Port)
		log.Printf("Database: %s", cfg.Database.Path)
		log.Printf("Audio path: %s", cfg.Audio.LocalPath)

		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			serverErr <- fmt.Errorf("server error: %w", err)
		}
		close(serverErr)
	}()

	// Wait for shutdown signal or server error
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case <-quit:
	case err := <-serverErr:
		if err != nil {
			return err
		}
	}

	log.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
	return nil
}

// securityHeaders adds standard security headers to all responses.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}
