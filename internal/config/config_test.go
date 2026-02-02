package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaults(t *testing.T) {
	cfg, err := Load() // No files, just defaults
	if err != nil {
		t.Fatalf("Load() with no files failed: %v", err)
	}

	if cfg.Server.Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.Server.Port)
	}
	if cfg.Database.Path != "data/inventory.db" {
		t.Errorf("expected database path 'data/inventory.db', got %s", cfg.Database.Path)
	}
	if cfg.Audio.LocalPath != "audio" {
		t.Errorf("expected audio local path 'audio', got %s", cfg.Audio.LocalPath)
	}
}

func TestLoadFromFile(t *testing.T) {
	// Create temp config file
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")

	content := `
server:
  port: 9090
database:
  path: /custom/path.db
audio:
  local_path: /custom/audio
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Server.Port != 9090 {
		t.Errorf("expected port 9090, got %d", cfg.Server.Port)
	}
	if cfg.Database.Path != "/custom/path.db" {
		t.Errorf("expected '/custom/path.db', got %s", cfg.Database.Path)
	}
	if cfg.Audio.LocalPath != "/custom/audio" {
		t.Errorf("expected '/custom/audio', got %s", cfg.Audio.LocalPath)
	}
}

func TestEnvOverride(t *testing.T) {
	_ = os.Setenv("PORT", "3000")
	_ = os.Setenv("DB_PATH", "/env/path.db")
	_ = os.Setenv("AUDIO_STORE_LOCAL_PATH", "/env/audio")
	defer func() {
		_ = os.Unsetenv("PORT")
		_ = os.Unsetenv("DB_PATH")
		_ = os.Unsetenv("AUDIO_STORE_LOCAL_PATH")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Server.Port != 3000 {
		t.Errorf("expected port 3000 from env, got %d", cfg.Server.Port)
	}
	if cfg.Database.Path != "/env/path.db" {
		t.Errorf("expected '/env/path.db' from env, got %s", cfg.Database.Path)
	}
	if cfg.Audio.LocalPath != "/env/audio" {
		t.Errorf("expected '/env/audio' from env, got %s", cfg.Audio.LocalPath)
	}
}

func TestValidation(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Config)
		wantErr bool
	}{
		{
			name:    "valid defaults",
			modify:  func(c *Config) {},
			wantErr: false,
		},
		{
			name:    "invalid port zero",
			modify:  func(c *Config) { c.Server.Port = 0 },
			wantErr: true,
		},
		{
			name:    "invalid port too high",
			modify:  func(c *Config) { c.Server.Port = 70000 },
			wantErr: true,
		},
		{
			name:    "empty database path",
			modify:  func(c *Config) { c.Database.Path = "" },
			wantErr: true,
		},
		{
			name:    "invalid duration",
			modify:  func(c *Config) { c.Server.ReadTimeout = "not-a-duration" },
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := defaults()
			tt.modify(cfg)
			err := validate(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMissingFileIgnored(t *testing.T) {
	cfg, err := Load("nonexistent.yaml", "also-nonexistent.yaml")
	if err != nil {
		t.Fatalf("Load() should ignore missing files, got error: %v", err)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("expected defaults when files missing")
	}
}

func TestFileMergeOrder(t *testing.T) {
	dir := t.TempDir()

	// First file sets port to 9000
	file1 := filepath.Join(dir, "config1.yaml")
	_ = os.WriteFile(file1, []byte("server:\n  port: 9000"), 0644)

	// Second file sets port to 9999
	file2 := filepath.Join(dir, "config2.yaml")
	_ = os.WriteFile(file2, []byte("server:\n  port: 9999"), 0644)

	cfg, err := Load(file1, file2)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Second file should win
	if cfg.Server.Port != 9999 {
		t.Errorf("expected port 9999 (from second file), got %d", cfg.Server.Port)
	}
}
