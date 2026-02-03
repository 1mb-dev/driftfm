package inventory

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// Repository handles track storage operations
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new inventory repository
func NewRepository(dbPath string) (*Repository, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// WAL mode allows concurrent reads during writes
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("failed to set WAL mode: %w", err)
	}
	// Wait up to 5s for write lock instead of failing immediately
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		return nil, fmt.Errorf("failed to set busy timeout: %w", err)
	}

	// SQLite supports one writer at a time; constrain the pool accordingly
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	return &Repository{db: db}, nil
}

// Close closes the database connection
func (r *Repository) Close() error {
	return r.db.Close()
}

// Ping checks database connectivity (for readiness probes)
func (r *Repository) Ping() error {
	return r.db.Ping()
}

// trackColumns is the standard column list for track queries.
// Play data comes from play_stats via LEFT JOIN (see trackFrom).
const trackColumns = `t.id, t.file_path, t.title, t.artist, t.mood, t.energy, t.tempo_bpm, t.has_vocals,
	t.musical_key, t.intensity, t.time_affinity, t.lyrics, t.duration_seconds,
	t.status, COALESCE(ps.play_count, 0), ps.last_played_at, t.created_at`

const trackFrom = `FROM tracks t LEFT JOIN play_stats ps ON t.file_path = ps.file_path`

// scanTrackRow scans a row into a scanTrack struct
func scanTrackRow(row interface{ Scan(...any) error }) (*scanTrack, error) {
	var st scanTrack
	err := row.Scan(
		&st.ID,
		&st.FilePath,
		&st.Title,
		&st.Artist,
		&st.Mood,
		&st.Energy,
		&st.TempoBPM,
		&st.HasVocals,
		&st.MusicalKey,
		&st.Intensity,
		&st.TimeAffinity,
		&st.Lyrics,
		&st.DurationSeconds,
		&st.Status,
		&st.PlayCount,
		&st.LastPlayedAt,
		&st.CreatedAt,
	)
	return &st, err
}

// GetByID retrieves a track by ID
func (r *Repository) GetByID(id int64) (*Track, error) {
	query := fmt.Sprintf(`SELECT %s %s WHERE t.id = ?`, trackColumns, trackFrom)

	st, err := scanTrackRow(r.db.QueryRow(query, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get track: %w", err)
	}

	return st.toTrack(), nil
}

// GetByMood retrieves all approved tracks for a mood.
// If instrumentalOnly is true, only tracks with has_vocals=0 are returned.
func (r *Repository) GetByMood(mood string, instrumentalOnly bool) ([]*Track, error) {
	where := "WHERE t.mood = ? AND t.status = ?"
	args := []any{mood, StatusApproved}
	if instrumentalOnly {
		where += " AND t.has_vocals = 0"
	}

	query := fmt.Sprintf(`
		SELECT %s %s
		%s
		ORDER BY COALESCE(ps.play_count, 0) ASC, ps.last_played_at ASC NULLS FIRST
	`, trackColumns, trackFrom, where)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query tracks: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var tracks []*Track
	for rows.Next() {
		st, err := scanTrackRow(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan track: %w", err)
		}
		tracks = append(tracks, st.toTrack())
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed iterating tracks: %w", err)
	}

	return tracks, nil
}

// UpdatePlayStats increments play count in the play_stats table.
// Uses a single INSERT...SELECT to atomically resolve file_path and UPSERT.
func (r *Repository) UpdatePlayStats(id int64) error {
	query := `
		INSERT INTO play_stats (file_path, play_count, last_played_at)
		SELECT file_path, 1, ?
		FROM tracks WHERE id = ?
		ON CONFLICT(file_path) DO UPDATE SET
			play_count = play_count + 1,
			last_played_at = excluded.last_played_at
	`
	result, err := r.db.Exec(query, time.Now().UTC().Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("failed to update play stats: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("failed to find track: id %d", id)
	}

	return nil
}

// BeginTx starts a new database transaction
func (r *Repository) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return r.db.BeginTx(ctx, nil)
}

// UpdatePlayStatsTx increments play count within an existing transaction
func (r *Repository) UpdatePlayStatsTx(tx *sql.Tx, id int64) error {
	query := `
		INSERT INTO play_stats (file_path, play_count, last_played_at)
		SELECT file_path, 1, ?
		FROM tracks WHERE id = ?
		ON CONFLICT(file_path) DO UPDATE SET
			play_count = play_count + 1,
			last_played_at = excluded.last_played_at
	`
	result, err := tx.Exec(query, time.Now().UTC().Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("failed to update play stats: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("failed to find track: id %d", id)
	}

	return nil
}

// RecordListenEventTx inserts a listen event within an existing transaction
func (r *Repository) RecordListenEventTx(tx *sql.Tx, evt ListenEvent) error {
	query := `
		INSERT INTO listen_events (track_id, mood, event_type, listen_seconds, playlist_position)
		VALUES (?, ?, ?, ?, ?)
	`
	_, err := tx.Exec(query, evt.TrackID, evt.Mood, evt.EventType, evt.ListenSeconds, evt.PlaylistPosition)
	if err != nil {
		return fmt.Errorf("failed to record listen event: %w", err)
	}
	return nil
}

// MoodStats holds aggregated stats for a mood
type MoodStats struct {
	Mood         string
	TrackCount   int
	TotalSeconds int
}

// GetMoodStats returns track count and total duration per mood
func (r *Repository) GetMoodStats() ([]MoodStats, error) {
	query := `
		SELECT mood, COUNT(*) as track_count, COALESCE(SUM(duration_seconds), 0) as total_seconds
		FROM tracks
		WHERE status = ?
		GROUP BY mood
		ORDER BY mood
	`

	rows, err := r.db.Query(query, StatusApproved)
	if err != nil {
		return nil, fmt.Errorf("failed to query mood stats: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var stats []MoodStats
	for rows.Next() {
		var s MoodStats
		if err := rows.Scan(&s.Mood, &s.TrackCount, &s.TotalSeconds); err != nil {
			return nil, fmt.Errorf("failed to scan mood stats: %w", err)
		}
		stats = append(stats, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed iterating mood stats: %w", err)
	}

	return stats, nil
}

