package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/1mb-dev/driftfm/internal/audio"
	"github.com/1mb-dev/driftfm/internal/cache"
	"github.com/1mb-dev/driftfm/internal/inventory"
	"github.com/1mb-dev/driftfm/internal/metrics"
)

// Repository defines the data operations the handler needs
type Repository interface {
	GetMoodStats() ([]inventory.MoodStats, error)
	GetByID(id int64) (*inventory.Track, error)
	BeginTx(ctx context.Context) (*sql.Tx, error)
	UpdatePlayStatsTx(tx *sql.Tx, id int64) error
	RecordListenEventTx(tx *sql.Tx, evt inventory.ListenEvent) error
}

// Radio provides playlist retrieval and play tracking
type Radio interface {
	GetPlaylist(mood string, instrumentalOnly bool) ([]*inventory.Track, error)
	RecordPlay(mood string, trackID int64)
}

// Handler holds dependencies for API handlers
type Handler struct {
	repo          Repository
	radio         Radio
	audioResolver audio.Resolver
	cache         *cache.Cache
}

// NewHandler creates a new API handler
func NewHandler(repo Repository, radio Radio, audioResolver audio.Resolver, c *cache.Cache) *Handler {
	return &Handler{
		repo:          repo,
		radio:         radio,
		audioResolver: audioResolver,
		cache:         c,
	}
}

// RegisterRoutes registers API routes on the given mux
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/moods", h.listMoods)
	mux.HandleFunc("/api/moods/", h.handleMoods)
	mux.HandleFunc("/api/tracks/", h.handleTracks)
}

// MoodInfo contains metadata about a mood
type MoodInfo struct {
	Name        string  `json:"name"`
	DisplayName string  `json:"display_name"`
	TrackCount  int     `json:"track_count"`
	TotalMins   float64 `json:"total_minutes"`
}

func (h *Handler) listMoods(w http.ResponseWriter, r *http.Request) {
	// Only handle exact /api/moods path
	if r.URL.Path != "/api/moods" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	// Check cache first
	if cached, found := h.cache.Get(cache.KeyMoodsList); found {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=300")
		w.Header().Set("X-Cache", "HIT")
		if err := json.NewEncoder(w).Encode(cached); err != nil {
			log.Printf("Error encoding cached moods: %v", err)
		}
		return
	}

	moods, err := h.repo.GetMoodStats()
	if err != nil {
		log.Printf("Error fetching moods: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// Convert to MoodInfo with display names
	displayNames := map[string]string{
		"focus":      "Focus",
		"calm":       "Calm",
		"late_night": "Late Night",
		"energize":   "Energize",
	}

	var result []MoodInfo
	for _, m := range moods {
		displayName := displayNames[m.Mood]
		if displayName == "" {
			displayName = m.Mood
		}
		result = append(result, MoodInfo{
			Name:        m.Mood,
			DisplayName: displayName,
			TrackCount:  m.TrackCount,
			TotalMins:   float64(m.TotalSeconds) / 60.0,
		})
	}

	// Cache the result
	if err := h.cache.Set(cache.KeyMoodsList, result); err != nil {
		log.Printf("Warning: failed to cache moods list: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.Header().Set("X-Cache", "MISS")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Printf("Error encoding moods: %v", err)
	}
}

// PlaylistTrack is a slim view of a track for playlist responses.
// The frontend uses only these 8 fields; dropping the other 12 reduces
// payload size by ~60%.
type PlaylistTrack struct {
	ID        int64   `json:"id"`
	FilePath  string  `json:"file_path"`
	AudioURL  string  `json:"audio_url,omitempty"`
	Title     *string `json:"title,omitempty"`
	Artist    *string `json:"artist,omitempty"`
	Energy    string  `json:"energy"`
	Intensity *int    `json:"intensity,omitempty"`
	Lyrics    *string `json:"lyrics,omitempty"`
}

func toPlaylistTracks(tracks []*inventory.Track) []PlaylistTrack {
	out := make([]PlaylistTrack, len(tracks))
	for i, t := range tracks {
		out[i] = PlaylistTrack{
			ID:        t.ID,
			FilePath:  t.FilePath,
			AudioURL:  t.AudioURL,
			Title:     t.Title,
			Artist:    t.Artist,
			Energy:    t.Energy,
			Intensity: t.Intensity,
			Lyrics:    t.Lyrics,
		}
	}
	return out
}

// validMoods contains the known mood identifiers
var validMoods = map[string]bool{
	"focus":      true,
	"calm":       true,
	"late_night": true,
	"energize":   true,
}

func (h *Handler) handleMoods(w http.ResponseWriter, r *http.Request) {
	// Parse path: /api/moods/{mood}/playlist
	path := strings.TrimPrefix(r.URL.Path, "/api/moods/")
	parts := strings.Split(path, "/")

	if len(parts) < 2 || parts[1] != "playlist" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	mood := parts[0]

	// Validate mood is a known value
	if !validMoods[mood] {
		http.Error(w, "Unknown mood", http.StatusNotFound)
		return
	}

	instrumentalOnly := r.URL.Query().Get("instrumental") == "true"
	h.getPlaylist(w, mood, instrumentalOnly)
}

func (h *Handler) getPlaylist(w http.ResponseWriter, mood string, instrumentalOnly bool) {
	// Cache key for mood's playlist (instrumental gets separate cache entry)
	cacheKey := cache.PlaylistKey(mood)
	if instrumentalOnly {
		cacheKey += ":instrumental"
	}

	if cached, found := h.cache.Get(cacheKey); found {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=60")
		w.Header().Set("X-Cache", "HIT")
		if err := json.NewEncoder(w).Encode(cached); err != nil {
			log.Printf("Error encoding cached playlist: %v", err)
		}
		return
	}

	// Get shuffled playlist
	tracks, err := h.radio.GetPlaylist(mood, instrumentalOnly)
	if err != nil {
		log.Printf("Error fetching playlist: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// Return empty array instead of null if no tracks
	if tracks == nil {
		tracks = []*inventory.Track{}
	}

	// Resolve audio URLs for each track
	for _, track := range tracks {
		url, err := h.audioResolver.ResolveURL(track.FilePath)
		if err != nil {
			log.Printf("Warning: failed to resolve audio URL for track %d: %v", track.ID, err)
		}
		track.AudioURL = url
	}

	// Convert to slim playlist payload
	slim := toPlaylistTracks(tracks)

	// Cache the result
	if len(slim) > 0 {
		if err := h.cache.Set(cacheKey, slim); err != nil {
			log.Printf("Warning: failed to cache playlist: %v", err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=60")
	w.Header().Set("X-Cache", "MISS")
	if err := json.NewEncoder(w).Encode(slim); err != nil {
		log.Printf("Error encoding playlist: %v", err)
	}
}

func (h *Handler) handleTracks(w http.ResponseWriter, r *http.Request) {
	// Parse path: /api/tracks/{id}/play
	path := strings.TrimPrefix(r.URL.Path, "/api/tracks/")
	parts := strings.Split(path, "/")

	if len(parts) < 2 {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		http.Error(w, "Invalid track ID", http.StatusBadRequest)
		return
	}

	switch parts[1] {
	case "play":
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.recordPlay(w, r, id)
	default:
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

// validEventTypes are the allowed listen event types
var validEventTypes = map[string]bool{
	inventory.EventPlay:     true,
	inventory.EventSkip:     true,
	inventory.EventComplete: true,
}

func (h *Handler) recordPlay(w http.ResponseWriter, r *http.Request, trackID int64) {
	// Decode optional JSON body; empty body defaults to a play event
	var evt inventory.ListenEvent
	if r.Body != nil {
		body, err := io.ReadAll(io.LimitReader(r.Body, 1024))
		if err == nil && len(body) > 0 {
			// Ignore decode errors — treat as body-less play
			_ = json.Unmarshal(body, &evt)
		}
	}

	// Fill defaults
	evt.TrackID = trackID
	if evt.EventType == "" {
		evt.EventType = inventory.EventPlay
	}

	// Validate event type
	if !validEventTypes[evt.EventType] {
		http.Error(w, "invalid event type", http.StatusBadRequest)
		return
	}

	// Get track to find mood for radio state and listen event
	track, err := h.repo.GetByID(trackID)
	if err != nil {
		log.Printf("Warning: failed to get track %d for radio update: %v", trackID, err)
	} else if track != nil {
		if evt.Mood == "" {
			evt.Mood = track.Mood
		}
	}

	// Wrap DB writes in a transaction to prevent partial state
	tx, err := h.repo.BeginTx(r.Context())
	if err != nil {
		log.Printf("Error starting transaction for track %d: %v", trackID, err)
		http.Error(w, "failed to record play", http.StatusInternalServerError)
		return
	}
	defer func() { _ = tx.Rollback() }()

	// Only update play_stats for non-skip events
	if evt.EventType != inventory.EventSkip {
		if err := h.repo.UpdatePlayStatsTx(tx, trackID); err != nil {
			log.Printf("Error recording play for track %d: %v", trackID, err)
			http.Error(w, "failed to record play", http.StatusInternalServerError)
			return
		}
	}

	// Record listen event if we have a mood
	if evt.Mood != "" {
		if err := h.repo.RecordListenEventTx(tx, evt); err != nil {
			log.Printf("Error recording listen event for track %d: %v", trackID, err)
			http.Error(w, "failed to record play", http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Error committing transaction for track %d: %v", trackID, err)
		http.Error(w, "failed to record play", http.StatusInternalServerError)
		return
	}

	// Update in-memory state after successful commit
	if evt.EventType != inventory.EventSkip {
		metrics.Get().RecordPlay()
		if track != nil {
			h.radio.RecordPlay(track.Mood, trackID)
		}
	}

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("ok")); err != nil {
		log.Printf("Error writing response for track %d play: %v", trackID, err)
	}
}
