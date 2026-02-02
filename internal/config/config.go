package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds application configuration
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Audio    AudioConfig    `yaml:"audio"`
}

// ServerConfig holds HTTP server settings
type ServerConfig struct {
	Port            int    `yaml:"port"`
	ReadTimeout     string `yaml:"read_timeout"`
	WriteTimeout    string `yaml:"write_timeout"`
	ShutdownTimeout string `yaml:"shutdown_timeout"`
}

// DatabaseConfig holds database settings
type DatabaseConfig struct {
	Path string `yaml:"path"`
}

// AudioConfig holds audio storage settings
type AudioConfig struct {
	LocalPath string `yaml:"local_path"`
}

// defaults returns a Config with sensible defaults
func defaults() *Config {
	return &Config{
		Server: ServerConfig{
			Port:            8080,
			ReadTimeout:     "15s",
			WriteTimeout:    "15s",
			ShutdownTimeout: "30s",
		},
		Database: DatabaseConfig{
			Path: "data/inventory.db",
		},
		Audio: AudioConfig{
			LocalPath: "audio",
		},
	}
}

// Load reads configuration from YAML files and environment variables.
// Files are loaded in order; later files override earlier ones.
// Environment variables override file values.
func Load(paths ...string) (*Config, error) {
	cfg := defaults()

	// Load each config file in order
	for _, path := range paths {
		if err := loadFile(cfg, path); err != nil {
			// Skip missing files silently (config.local.yaml may not exist)
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("loading %s: %w", path, err)
		}
	}

	// Apply environment variable overrides
	applyEnvOverrides(cfg)

	// Validate
	if err := validate(cfg); err != nil {
		return nil, fmt.Errorf("config validation: %w", err)
	}

	return cfg, nil
}

// loadFile reads a YAML file and merges into cfg
func loadFile(cfg *Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Parse YAML into a temporary struct, then merge non-zero values
	var fileCfg Config
	if err := yaml.Unmarshal(data, &fileCfg); err != nil {
		return fmt.Errorf("parsing YAML: %w", err)
	}

	// Merge: file values override defaults (only non-zero values)
	mergeConfig(cfg, &fileCfg)
	return nil
}

// mergeConfig copies non-zero values from src to dst
func mergeConfig(dst, src *Config) {
	// Server
	if src.Server.Port != 0 {
		dst.Server.Port = src.Server.Port
	}
	if src.Server.ReadTimeout != "" {
		dst.Server.ReadTimeout = src.Server.ReadTimeout
	}
	if src.Server.WriteTimeout != "" {
		dst.Server.WriteTimeout = src.Server.WriteTimeout
	}
	if src.Server.ShutdownTimeout != "" {
		dst.Server.ShutdownTimeout = src.Server.ShutdownTimeout
	}

	// Database
	if src.Database.Path != "" {
		dst.Database.Path = src.Database.Path
	}

	// Audio
	if src.Audio.LocalPath != "" {
		dst.Audio.LocalPath = src.Audio.LocalPath
	}
}

// applyEnvOverrides applies environment variable overrides
func applyEnvOverrides(cfg *Config) {
	// Server
	if v := os.Getenv("PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = port
		}
	}

	// Database
	if v := os.Getenv("DB_PATH"); v != "" {
		cfg.Database.Path = v
	}

	// Audio
	if v := os.Getenv("AUDIO_STORE_LOCAL_PATH"); v != "" {
		cfg.Audio.LocalPath = v
	}
}

// validate checks required fields and value constraints
func validate(cfg *Config) error {
	if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
		return fmt.Errorf("server.port must be 1-65535, got %d", cfg.Server.Port)
	}

	if cfg.Database.Path == "" {
		return fmt.Errorf("database.path is required")
	}

	// Validate durations parse correctly
	if _, err := cfg.GetReadTimeout(); err != nil {
		return fmt.Errorf("server.read_timeout invalid: %w", err)
	}
	if _, err := cfg.GetWriteTimeout(); err != nil {
		return fmt.Errorf("server.write_timeout invalid: %w", err)
	}
	if _, err := cfg.GetShutdownTimeout(); err != nil {
		return fmt.Errorf("server.shutdown_timeout invalid: %w", err)
	}

	return nil
}

// Helper methods to get parsed duration values

func (c *Config) GetReadTimeout() (time.Duration, error) {
	return time.ParseDuration(c.Server.ReadTimeout)
}

func (c *Config) GetWriteTimeout() (time.Duration, error) {
	return time.ParseDuration(c.Server.WriteTimeout)
}

func (c *Config) GetShutdownTimeout() (time.Duration, error) {
	return time.ParseDuration(c.Server.ShutdownTimeout)
}
