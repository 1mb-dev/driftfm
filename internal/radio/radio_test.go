package radio

import (
	"database/sql"
	"math/rand"
	"sync"
	"testing"

	"github.com/1mb-dev/driftfm/internal/inventory"
	"github.com/1mb-dev/driftfm/internal/testutil"
	_ "modernc.org/sqlite"
)

// setupTestRepo creates a test repository with seeded data
func setupTestRepo(t *testing.T) *inventory.Repository {
	t.Helper()

	tmpDB := t.TempDir() + "/test.db"

	db, err := sql.Open("sqlite", tmpDB)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	_, err = db.Exec(testutil.SchemaDDL + `
		INSERT INTO tracks (id, file_path, title, mood, duration_seconds, status) VALUES
			(1, 'focus/track1.mp3', 'Focus Track 1', 'focus', 180, 'approved'),
			(2, 'focus/track2.mp3', 'Focus Track 2', 'focus', 240, 'approved'),
			(3, 'focus/track3.mp3', 'Focus Track 3', 'focus', 200, 'approved'),
			(4, 'calm/track1.mp3', 'Calm Track 1', 'calm', 200, 'approved');
		INSERT INTO play_stats (file_path, play_count) VALUES
			('focus/track1.mp3', 10),
			('focus/track3.mp3', 5),
			('calm/track1.mp3', 2);
	`)
	if err != nil {
		t.Fatalf("failed to setup test db: %v", err)
	}
	_ = db.Close()

	repo, err := inventory.NewRepository(tmpDB)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	t.Cleanup(func() { _ = repo.Close() })
	return repo
}

func TestRecordPlay(t *testing.T) {
	r := &Radio{
		recentlyPlayed: make([]int64, 0),
		maxRecent:      3,
	}

	// Record plays
	r.RecordPlay(1)
	r.RecordPlay(2)
	r.RecordPlay(3)

	if len(r.recentlyPlayed) != 3 {
		t.Errorf("got %d recent, want 3", len(r.recentlyPlayed))
	}

	// Record 4th - should trim oldest
	r.RecordPlay(4)
	if len(r.recentlyPlayed) != 3 {
		t.Errorf("got %d recent, want 3 (should trim)", len(r.recentlyPlayed))
	}
	if r.recentlyPlayed[0] != 2 {
		t.Errorf("oldest should be 2, got %d", r.recentlyPlayed[0])
	}

	// Duplicate should be ignored
	r.RecordPlay(4)
	if len(r.recentlyPlayed) != 3 {
		t.Error("duplicate should not add to recent list")
	}
}

func TestShuffleWithRecency(t *testing.T) {
	r := &Radio{
		recentlyPlayed: []int64{1, 2}, // tracks 1,2 recently played
		maxRecent:      3,
		rng:            rand.New(rand.NewSource(42)), // deterministic
	}

	tracks := []*inventory.Track{
		{ID: 1},
		{ID: 2},
		{ID: 3},
		{ID: 4},
	}

	r.mu.Lock()
	r.shuffleWithRecencyLocked(tracks)
	r.mu.Unlock()

	// Fresh tracks should be first, recent tracks last
	foundRecent := false
	for _, track := range tracks {
		isRecent := track.ID == 1 || track.ID == 2
		if !isRecent && foundRecent {
			t.Errorf("fresh track %d found after recent tracks", track.ID)
		}
		if isRecent {
			foundRecent = true
		}
	}

	// Verify recent tracks are at the end
	lastTwo := tracks[len(tracks)-2:]
	for _, track := range lastTwo {
		if track.ID != 1 && track.ID != 2 {
			t.Errorf("expected recent track at end, got ID %d", track.ID)
		}
	}
}

// TestGetPlaylist tests the core playlist generation
func TestGetPlaylist(t *testing.T) {
	repo := setupTestRepo(t)

	tests := []struct {
		name    string
		mood    string
		wantMin int
		wantMax int
	}{
		{"focus mood", "focus", 3, 3},
		{"calm mood", "calm", 1, 1},
		{"unknown mood returns empty", "unknown", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			radio := NewRadio(repo, tt.mood)
			tracks, err := radio.GetPlaylist(false)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(tracks) < tt.wantMin || len(tracks) > tt.wantMax {
				t.Errorf("got %d tracks, want between %d and %d", len(tracks), tt.wantMin, tt.wantMax)
			}
		})
	}
}

// TestManagerGetPlaylist tests the manager's playlist delegation
func TestManagerGetPlaylist(t *testing.T) {
	repo := setupTestRepo(t)
	mgr := NewManager(repo)

	tests := []struct {
		name    string
		mood    string
		wantMin int
	}{
		{"focus playlist", "focus", 3},
		{"calm playlist", "calm", 1},
		{"unknown mood", "unknown", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracks, err := mgr.GetPlaylist(tt.mood, false)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(tracks) < tt.wantMin {
				t.Errorf("got %d tracks, want at least %d", len(tracks), tt.wantMin)
			}
		})
	}
}

// TestManagerGetRadio tests radio caching and concurrent access
func TestManagerGetRadio(t *testing.T) {
	repo := setupTestRepo(t)
	mgr := NewManager(repo)

	// Get same radio twice - should return same instance
	radio1 := mgr.GetRadio("focus")
	radio2 := mgr.GetRadio("focus")

	if radio1 != radio2 {
		t.Error("GetRadio should return cached instance")
	}

	// Different mood should return different radio
	radio3 := mgr.GetRadio("calm")
	if radio1 == radio3 {
		t.Error("different moods should have different radios")
	}
}

// TestConcurrentAccess verifies no data races when multiple goroutines
// call GetPlaylist and RecordPlay concurrently.
// Run with: go test -race ./internal/radio/...
func TestConcurrentAccess(t *testing.T) {
	repo := setupTestRepo(t)
	r := NewRadio(repo, "focus")

	var wg sync.WaitGroup
	for range 10 {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, _ = r.GetPlaylist(false)
		}()
		go func() {
			defer wg.Done()
			r.RecordPlay(1)
		}()
	}
	wg.Wait()
}

// TestManagerRecordPlay tests play recording through manager
func TestManagerRecordPlay(t *testing.T) {
	repo := setupTestRepo(t)
	mgr := NewManager(repo)

	// Record play through manager
	mgr.RecordPlay("focus", 1)

	// Verify it was recorded in the radio
	radio := mgr.GetRadio("focus")
	if len(radio.recentlyPlayed) != 1 || radio.recentlyPlayed[0] != 1 {
		t.Errorf("expected track 1 in recent, got %v", radio.recentlyPlayed)
	}
}
