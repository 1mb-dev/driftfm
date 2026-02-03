package inventory

import (
	"database/sql"
	"time"
)

// Track represents an audio track in the inventory
type Track struct {
	ID       int64  `json:"id"`
	FilePath string `json:"file_path"`

	// AudioURL is the resolved playable URL (computed at runtime, not stored)
	AudioURL string `json:"audio_url,omitempty"`

	// Display metadata
	Title  *string `json:"title,omitempty"`
	Artist *string `json:"artist,omitempty"`

	// Classification
	Mood       string  `json:"mood"`
	Energy     string  `json:"energy"`
	TempoBPM   *int    `json:"tempo_bpm,omitempty"`
	HasVocals  bool    `json:"has_vocals"`
	MusicalKey *string `json:"musical_key,omitempty"`

	// Moodlet discovery
	Intensity    *int    `json:"intensity,omitempty"`     // 1-10: 1=light, 10=deep
	TimeAffinity *string `json:"time_affinity,omitempty"` // morning, afternoon, evening, night, any

	// Content
	Lyrics *string `json:"lyrics,omitempty"`

	// Audio properties
	DurationSeconds int `json:"duration_seconds"`

	// Status and tracking
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`

	// Play stats (sourced from play_stats table via LEFT JOIN, not from tracks)
	PlayCount    int        `json:"play_count"`
	LastPlayedAt *time.Time `json:"last_played_at,omitempty"`
}

// scanTrack is a helper for scanning track rows
type scanTrack struct {
	ID              int64
	FilePath        string
	Title           sql.NullString
	Artist          sql.NullString
	Mood            string
	Energy          string
	TempoBPM        sql.NullInt64
	HasVocals       int
	MusicalKey      sql.NullString
	Intensity       sql.NullInt64
	TimeAffinity    sql.NullString
	Lyrics          sql.NullString
	DurationSeconds int
	Status          string
	PlayCount       int
	LastPlayedAt    sql.NullTime
	CreatedAt       time.Time
}

func (s *scanTrack) toTrack() *Track {
	t := &Track{
		ID:              s.ID,
		FilePath:        s.FilePath,
		Mood:            s.Mood,
		Energy:          s.Energy,
		HasVocals:       s.HasVocals == 1,
		DurationSeconds: s.DurationSeconds,
		Status:          s.Status,
		PlayCount:       s.PlayCount,
		CreatedAt:       s.CreatedAt,
	}
	if s.Title.Valid {
		t.Title = &s.Title.String
	}
	if s.Artist.Valid {
		t.Artist = &s.Artist.String
	}
	if s.TempoBPM.Valid {
		v := int(s.TempoBPM.Int64)
		t.TempoBPM = &v
	}
	if s.MusicalKey.Valid {
		t.MusicalKey = &s.MusicalKey.String
	}
	if s.Intensity.Valid {
		v := int(s.Intensity.Int64)
		t.Intensity = &v
	}
	if s.TimeAffinity.Valid {
		t.TimeAffinity = &s.TimeAffinity.String
	}
	if s.Lyrics.Valid {
		t.Lyrics = &s.Lyrics.String
	}
	if s.LastPlayedAt.Valid {
		t.LastPlayedAt = &s.LastPlayedAt.Time
	}
	return t
}

// Status constants
const (
	StatusApproved = "approved"
)

// ListenEvent represents a single listen engagement event
type ListenEvent struct {
	TrackID          int64  `json:"track_id"`
	Mood             string `json:"mood"`
	EventType        string `json:"event"`
	ListenSeconds    int    `json:"listen_seconds"`
	PlaylistPosition *int   `json:"position,omitempty"`
}

// Listen event type constants
const (
	EventPlay     = "play"
	EventSkip     = "skip"
	EventComplete = "complete"
)
