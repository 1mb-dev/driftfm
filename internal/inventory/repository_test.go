package inventory

import (
	"database/sql"
	"testing"

	"github.com/1mb-dev/driftfm/internal/testutil"
	_ "modernc.org/sqlite"
)

func openTestDB(t *testing.T, seedSQL string) *Repository {
	t.Helper()

	tmpDB := t.TempDir() + "/test.db"
	db, err := sql.Open("sqlite", tmpDB)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	_, err = db.Exec(testutil.SchemaDDL + seedSQL)
	if err != nil {
		t.Fatalf("failed to setup test db: %v", err)
	}
	_ = db.Close()

	repo, err := NewRepository(tmpDB)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	t.Cleanup(func() { _ = repo.Close() })
	return repo
}

func setupTestRepo(t *testing.T) *Repository {
	t.Helper()
	return openTestDB(t, `
		INSERT INTO tracks (id, file_path, title, mood, duration_seconds, status, has_vocals) VALUES
			(1, 'focus/track1.mp3', 'Focus Track 1', 'focus', 180, 'approved', 0),
			(2, 'focus/track2.mp3', 'Focus Track 2', 'focus', 240, 'approved', 1),
			(3, 'calm/track1.mp3', 'Calm Track 1', 'calm', 200, 'approved', 0),
			(4, 'focus/pending.mp3', 'Pending Track', 'focus', 150, 'pending', 0);
		INSERT INTO play_stats (file_path, play_count) VALUES
			('focus/track1.mp3', 5),
			('calm/track1.mp3', 2);
	`)
}

func TestGetByMood(t *testing.T) {
	repo := setupTestRepo(t)

	tests := []struct {
		name      string
		mood      string
		wantCount int
		wantFirst string // expected first track (sorted by play_count ASC)
	}{
		{"focus mood returns approved only", "focus", 2, "Focus Track 2"}, // track2 has 0 plays
		{"calm mood", "calm", 1, "Calm Track 1"},
		{"unknown mood returns empty", "unknown", 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracks, err := repo.GetByMood(tt.mood, false)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(tracks) != tt.wantCount {
				t.Errorf("got %d tracks, want %d", len(tracks), tt.wantCount)
			}
			if tt.wantCount > 0 && (tracks[0].Title == nil || *tracks[0].Title != tt.wantFirst) {
				t.Errorf("first track = %v, want %q (should sort by play_count ASC)", tracks[0].Title, tt.wantFirst)
			}
		})
	}
}

func TestGetByMood_InstrumentalOnly(t *testing.T) {
	repo := setupTestRepo(t)

	// Focus has 2 approved: track1 (instrumental), track2 (vocals)
	all, err := repo.GetByMood("focus", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("got %d tracks, want 2", len(all))
	}

	instrumental, err := repo.GetByMood("focus", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(instrumental) != 1 {
		t.Fatalf("got %d instrumental tracks, want 1", len(instrumental))
	}
	if instrumental[0].HasVocals {
		t.Error("instrumental-only returned a vocal track")
	}
}

func TestUpdatePlayStats(t *testing.T) {
	repo := setupTestRepo(t)

	// Get initial state
	track, _ := repo.GetByID(1)
	initialCount := track.PlayCount

	// Update play stats
	err := repo.UpdatePlayStats(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify increment
	track, _ = repo.GetByID(1)
	if track.PlayCount != initialCount+1 {
		t.Errorf("play_count = %d, want %d", track.PlayCount, initialCount+1)
	}
	if track.LastPlayedAt == nil {
		t.Error("last_played_at should be set")
	}
}

func TestUpdatePlayStats_NewTrack(t *testing.T) {
	repo := setupTestRepo(t)

	// Track 2 has no play_stats row — tests the INSERT path
	track, _ := repo.GetByID(2)
	if track.PlayCount != 0 {
		t.Fatalf("expected 0 initial plays, got %d", track.PlayCount)
	}

	err := repo.UpdatePlayStats(2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	track, _ = repo.GetByID(2)
	if track.PlayCount != 1 {
		t.Errorf("play_count = %d, want 1", track.PlayCount)
	}
	if track.LastPlayedAt == nil {
		t.Error("last_played_at should be set")
	}
}

func TestUpdatePlayStats_NonExistent(t *testing.T) {
	repo := setupTestRepo(t)

	err := repo.UpdatePlayStats(999)
	if err == nil {
		t.Error("expected error for non-existent track")
	}
}

func TestGetMoodStats(t *testing.T) {
	repo := setupTestRepo(t)

	stats, err := repo.GetMoodStats()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have 2 moods (focus=2, calm=1) - pending tracks excluded
	if len(stats) != 2 {
		t.Fatalf("got %d moods, want 2", len(stats))
	}

	// Find focus stats
	var focusStats *MoodStats
	for i := range stats {
		if stats[i].Mood == "focus" {
			focusStats = &stats[i]
			break
		}
	}

	if focusStats == nil {
		t.Fatal("focus mood not found in stats")
	}
	if focusStats.TrackCount != 2 {
		t.Errorf("focus track_count = %d, want 2", focusStats.TrackCount)
	}
	if focusStats.TotalSeconds != 420 { // 180 + 240
		t.Errorf("focus total_seconds = %d, want 420", focusStats.TotalSeconds)
	}
}

func TestGetByID(t *testing.T) {
	repo := setupTestRepo(t)

	tests := []struct {
		name      string
		id        int64
		wantTitle string
		wantNil   bool
	}{
		{"existing track", 1, "Focus Track 1", false},
		{"non-existent track", 999, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			track, err := repo.GetByID(tt.id)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantNil && track != nil {
				t.Error("expected nil, got track")
			}
			if !tt.wantNil && (track.Title == nil || *track.Title != tt.wantTitle) {
				t.Errorf("title = %v, want %q", track.Title, tt.wantTitle)
			}
		})
	}
}

func TestPing(t *testing.T) {
	repo := setupTestRepo(t)

	err := repo.Ping()
	if err != nil {
		t.Errorf("Ping should succeed on valid repo: %v", err)
	}
}
