package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/1mb-dev/driftfm/internal/audio"
	"github.com/1mb-dev/driftfm/internal/cache"
	"github.com/1mb-dev/driftfm/internal/inventory"
	"github.com/1mb-dev/driftfm/internal/radio"
	"github.com/1mb-dev/driftfm/internal/testutil"
	_ "modernc.org/sqlite"
)

// mockResolver is a test resolver that returns predictable URLs
type mockResolver struct{}

func (m *mockResolver) ResolveURL(filePath string) (string, error) {
	return fmt.Sprintf("/audio/%s", filePath), nil
}

// Ensure mockResolver implements audio.Resolver
var _ audio.Resolver = (*mockResolver)(nil)

// setupTestCache creates a cache for testing
func setupTestCache(t *testing.T) *cache.Cache {
	t.Helper()
	c, err := cache.New()
	if err != nil {
		t.Fatalf("failed to create test cache: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

// setupTestDB creates a temp SQLite database with schema and test data
func setupTestDB(t *testing.T) *inventory.Repository {
	t.Helper()

	tmpDB := t.TempDir() + "/test.db"

	// Create schema and seed data first
	db, err := sql.Open("sqlite", tmpDB)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	_, err = db.Exec(testutil.SchemaDDL + `
		INSERT INTO tracks (file_path, title, mood, duration_seconds, status) VALUES
			('focus/track1.mp3', 'Focus Track 1', 'focus', 180, 'approved'),
			('focus/track2.mp3', 'Focus Track 2', 'focus', 240, 'approved'),
			('calm/track1.mp3', 'Calm Track 1', 'calm', 200, 'approved');
	`)
	if err != nil {
		t.Fatalf("failed to setup test db: %v", err)
	}
	_ = db.Close()

	// Now open via repository
	repo, err := inventory.NewRepository(tmpDB)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	t.Cleanup(func() { _ = repo.Close() })
	return repo
}

func TestListMoods(t *testing.T) {
	repo := setupTestDB(t)
	c := setupTestCache(t)
	h := NewHandler(repo, radio.NewManager(repo), &mockResolver{}, c)

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantMoods  int // expected number of moods in response
	}{
		{"valid path", "/api/moods", http.StatusOK, 2}, // focus and calm
		{"wrong path", "/api/moods/", http.StatusNotFound, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			h.listMoods(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusOK {
				var moods []MoodInfo
				if err := json.NewDecoder(w.Body).Decode(&moods); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if len(moods) != tt.wantMoods {
					t.Errorf("got %d moods, want %d", len(moods), tt.wantMoods)
				}
			}
		})
	}
}

func TestGetPlaylist(t *testing.T) {
	repo := setupTestDB(t)
	c := setupTestCache(t)
	h := NewHandler(repo, radio.NewManager(repo), &mockResolver{}, c)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantTracks int // expected minimum tracks (-1 to skip check)
	}{
		{"valid mood", "/api/moods/focus/playlist", http.StatusOK, 2},
		{"unknown mood", "/api/moods/unknown/playlist", http.StatusNotFound, -1},
		{"missing playlist", "/api/moods/focus", http.StatusNotFound, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			// Validate response body for successful requests
			if tt.wantStatus == http.StatusOK && tt.wantTracks >= 0 {
				var tracks []map[string]any
				if err := json.NewDecoder(w.Body).Decode(&tracks); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if len(tracks) != tt.wantTracks {
					t.Errorf("got %d tracks, want %d", len(tracks), tt.wantTracks)
				}
				// Verify slim track structure for non-empty responses
				if len(tracks) > 0 {
					if _, ok := tracks[0]["id"]; !ok {
						t.Error("track missing 'id' field")
					}
					if _, ok := tracks[0]["file_path"]; !ok {
						t.Error("track missing 'file_path' field")
					}
					// Slim payload should NOT include dropped fields
					for _, dropped := range []string{"status", "play_count", "created_at", "tempo_bpm", "duration_seconds"} {
						if _, ok := tracks[0][dropped]; ok {
							t.Errorf("slim playlist should not include %q field", dropped)
						}
					}
				}
			}
		})
	}
}

func TestRecordPlay(t *testing.T) {
	repo := setupTestDB(t)
	c := setupTestCache(t)
	h := NewHandler(repo, radio.NewManager(repo), &mockResolver{}, c)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		{"valid POST", http.MethodPost, "/api/tracks/1/play", http.StatusOK},
		{"invalid method", http.MethodGet, "/api/tracks/1/play", http.StatusMethodNotAllowed},
		{"invalid ID", http.MethodPost, "/api/tracks/abc/play", http.StatusBadRequest},
		{"unknown action", http.MethodPost, "/api/tracks/1/unknown", http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

// --- Mock types for error-path testing ---

// mockRepo implements Repository with configurable errors
type mockRepo struct {
	getMoodStatsErr        error
	getMoodStatsResult     []inventory.MoodStats
	getByIDErr             error
	getByIDResult          *inventory.Track
	updatePlayStatsErr     error
	recordListenEventErr   error
	recordListenEventCalls []inventory.ListenEvent
	beginTxErr             error

	// in-memory DB for transaction support in tests
	txDB *sql.DB
}

func newMockRepo() *mockRepo {
	db, _ := sql.Open("sqlite", ":memory:")
	return &mockRepo{txDB: db}
}

func (m *mockRepo) GetMoodStats() ([]inventory.MoodStats, error) {
	return m.getMoodStatsResult, m.getMoodStatsErr
}

func (m *mockRepo) GetByID(id int64) (*inventory.Track, error) {
	return m.getByIDResult, m.getByIDErr
}

func (m *mockRepo) BeginTx(_ context.Context) (*sql.Tx, error) {
	if m.beginTxErr != nil {
		return nil, m.beginTxErr
	}
	return m.txDB.Begin()
}

func (m *mockRepo) UpdatePlayStatsTx(_ *sql.Tx, _ int64) error {
	return m.updatePlayStatsErr
}

func (m *mockRepo) RecordListenEventTx(_ *sql.Tx, evt inventory.ListenEvent) error {
	m.recordListenEventCalls = append(m.recordListenEventCalls, evt)
	return m.recordListenEventErr
}

var _ Repository = (*mockRepo)(nil)

// mockRadio implements Radio with configurable errors
type mockRadio struct {
	getPlaylistErr    error
	getPlaylistResult []*inventory.Track
	recordPlayCalled  bool
}

func (m *mockRadio) GetPlaylist(_ string, _ bool) ([]*inventory.Track, error) {
	return m.getPlaylistResult, m.getPlaylistErr
}

func (m *mockRadio) RecordPlay(_ string, _ int64) {
	m.recordPlayCalled = true
}

var _ Radio = (*mockRadio)(nil)

// --- Error path tests ---

func TestRecordPlay_DBFailure(t *testing.T) {
	c := setupTestCache(t)
	repo := newMockRepo()
	repo.updatePlayStatsErr = errors.New("db connection lost")
	r := &mockRadio{}
	h := NewHandler(repo, r, &mockResolver{}, c)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/tracks/1/play", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
	if r.recordPlayCalled {
		t.Error("RecordPlay should not be called when DB write fails")
	}
}

func TestListMoods_DBFailure(t *testing.T) {
	c := setupTestCache(t)
	repo := newMockRepo()
	repo.getMoodStatsErr = errors.New("db error")
	h := NewHandler(repo, &mockRadio{}, &mockResolver{}, c)

	req := httptest.NewRequest(http.MethodGet, "/api/moods", nil)
	w := httptest.NewRecorder()
	h.listMoods(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestGetPlaylist_RadioFailure(t *testing.T) {
	c := setupTestCache(t)
	repo := newMockRepo()
	r := &mockRadio{getPlaylistErr: errors.New("radio error")}
	h := NewHandler(repo, r, &mockResolver{}, c)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/moods/focus/playlist", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestRecordPlay_GetByIDFailure(t *testing.T) {
	c := setupTestCache(t)
	repo := newMockRepo()
	repo.getByIDErr = errors.New("db read error")
	r := &mockRadio{}
	h := NewHandler(repo, r, &mockResolver{}, c)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/tracks/1/play", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// Play was recorded (UpdatePlayStats succeeded), so expect 200
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	// Radio state should not be updated when GetByID fails
	if r.recordPlayCalled {
		t.Error("RecordPlay should not be called when GetByID fails")
	}
}

func TestRecordPlay_EmptyBody_DefaultsToPlay(t *testing.T) {
	repo := setupTestDB(t)
	c := setupTestCache(t)
	h := NewHandler(repo, radio.NewManager(repo), &mockResolver{}, c)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// Body-less POST defaults to a play event
	req := httptest.NewRequest(http.MethodPost, "/api/tracks/1/play", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRecordPlay_WithListenEvent(t *testing.T) {
	c := setupTestCache(t)
	repo := newMockRepo()
	repo.getByIDResult = &inventory.Track{ID: 1, Mood: "focus"}
	r := &mockRadio{}
	h := NewHandler(repo, r, &mockResolver{}, c)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	body := `{"event":"play","listen_seconds":0,"mood":"focus","position":0}`
	req := httptest.NewRequest(http.MethodPost, "/api/tracks/1/play", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if len(repo.recordListenEventCalls) != 1 {
		t.Fatalf("expected 1 listen event, got %d", len(repo.recordListenEventCalls))
	}
	evt := repo.recordListenEventCalls[0]
	if evt.EventType != "play" {
		t.Errorf("event_type = %q, want %q", evt.EventType, "play")
	}
	if evt.Mood != "focus" {
		t.Errorf("mood = %q, want %q", evt.Mood, "focus")
	}
}

func TestRecordPlay_SkipEvent_NoPlayStatsUpdate(t *testing.T) {
	c := setupTestCache(t)
	repo := newMockRepo()
	repo.getByIDResult = &inventory.Track{ID: 1, Mood: "focus"}
	r := &mockRadio{}
	h := NewHandler(repo, r, &mockResolver{}, c)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	body := `{"event":"skip","listen_seconds":45,"mood":"focus","position":2}`
	req := httptest.NewRequest(http.MethodPost, "/api/tracks/1/play", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	// Skip should NOT call RecordPlay on radio
	if r.recordPlayCalled {
		t.Error("RecordPlay should not be called for skip events")
	}
	// But listen event should still be recorded
	if len(repo.recordListenEventCalls) != 1 {
		t.Fatalf("expected 1 listen event, got %d", len(repo.recordListenEventCalls))
	}
	if repo.recordListenEventCalls[0].EventType != "skip" {
		t.Errorf("event_type = %q, want %q", repo.recordListenEventCalls[0].EventType, "skip")
	}
}

func TestRecordPlay_InvalidEventType_Returns400(t *testing.T) {
	c := setupTestCache(t)
	repo := newMockRepo()
	repo.getByIDResult = &inventory.Track{ID: 1, Mood: "focus"}
	r := &mockRadio{}
	h := NewHandler(repo, r, &mockResolver{}, c)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	body := `{"event":"bogus","listen_seconds":10,"mood":"focus"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tracks/1/play", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if r.recordPlayCalled {
		t.Error("RecordPlay should not be called for invalid event types")
	}
}

func TestRecordPlay_MalformedBody_StillSucceeds(t *testing.T) {
	c := setupTestCache(t)
	repo := newMockRepo()
	repo.getByIDResult = &inventory.Track{ID: 1, Mood: "focus"}
	r := &mockRadio{}
	h := NewHandler(repo, r, &mockResolver{}, c)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// Bad JSON — should default to play event
	body := `{invalid json`
	req := httptest.NewRequest(http.MethodPost, "/api/tracks/1/play", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	// Should still record play via radio
	if !r.recordPlayCalled {
		t.Error("RecordPlay should be called even with malformed body (defaults to play)")
	}
}
