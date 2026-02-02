package audio

import (
	"testing"
)

func TestLocalResolver(t *testing.T) {
	resolver := NewResolver("audio")

	tests := []struct {
		name     string
		filePath string
		want     string
	}{
		{"simple path", "track.mp3", "/audio/track.mp3"},
		{"nested path", "focus/track1.mp3", "/audio/focus/track1.mp3"},
		{"deep path", "focus/ambient/track.mp3", "/audio/focus/ambient/track.mp3"},
		{"traversal attempt", "../../../etc/passwd", "/audio/etc/passwd"},
		{"leading slash", "/focus/track.mp3", "/audio/focus/track.mp3"},
		{"double dots in middle", "focus/../calm/track.mp3", "/audio/calm/track.mp3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolver.ResolveURL(tt.filePath)
			if err != nil {
				t.Fatalf("ResolveURL() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("ResolveURL(%q) = %q, want %q", tt.filePath, got, tt.want)
			}
		})
	}
}
