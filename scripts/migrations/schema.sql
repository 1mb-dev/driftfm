-- Drift FM - Inventory Schema
-- Track metadata storage

-- Migration tracking
CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT PRIMARY KEY,
    applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Mark this schema as including all migrations
INSERT OR IGNORE INTO schema_migrations (version) VALUES ('004_play_stats');
INSERT OR IGNORE INTO schema_migrations (version) VALUES ('005_listen_events');

CREATE TABLE IF NOT EXISTS tracks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    -- File reference (URL-friendly auto-generated filename)
    file_path TEXT NOT NULL UNIQUE,

    -- Display metadata
    title TEXT,                                        -- Human-readable display name
    artist TEXT DEFAULT 'Drift FM',                   -- Artist/creator name

    -- Classification
    mood TEXT NOT NULL DEFAULT 'focus',               -- Primary mood
    energy TEXT NOT NULL DEFAULT 'low',               -- low, medium, high
    tempo_bpm INTEGER,                                -- Beats per minute
    has_vocals INTEGER NOT NULL DEFAULT 0,            -- 0=instrumental, 1=has vocals
    musical_key TEXT,                                 -- Musical key (C, Am, Dm, etc.)

    -- Moodlet discovery
    intensity INTEGER DEFAULT 5                       -- 1-10: 1=very light, 10=very deep
        CHECK (intensity >= 1 AND intensity <= 10),
    time_affinity TEXT DEFAULT 'any'                  -- Best time to play
        CHECK (time_affinity IN ('morning', 'afternoon', 'evening', 'night', 'any')),

    -- Content
    lyrics TEXT,                                      -- Full lyrics for vocal tracks

    -- Audio properties
    duration_seconds INTEGER NOT NULL,

    -- Status workflow: pending -> approved -> (played) -> expired
    status TEXT NOT NULL DEFAULT 'approved',

    -- Timestamps
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_tracks_mood ON tracks(mood);
CREATE INDEX IF NOT EXISTS idx_tracks_status ON tracks(status);
CREATE INDEX IF NOT EXISTS idx_tracks_mood_status ON tracks(mood, status);
CREATE INDEX IF NOT EXISTS idx_tracks_intensity ON tracks(intensity);

-- Runtime play data (separated from content data for safe reimports)
CREATE TABLE IF NOT EXISTS play_stats (
    file_path TEXT PRIMARY KEY NOT NULL REFERENCES tracks(file_path) ON DELETE CASCADE,
    play_count INTEGER NOT NULL DEFAULT 0,
    last_played_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Listen events: write-only engagement data (play/skip/complete).
-- Used as a signal log for future playlist tuning. Not queried at runtime.
CREATE TABLE IF NOT EXISTS listen_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    track_id INTEGER NOT NULL REFERENCES tracks(id),
    mood TEXT NOT NULL,
    event_type TEXT NOT NULL CHECK (event_type IN ('play', 'skip', 'complete')),
    listen_seconds INTEGER NOT NULL DEFAULT 0,
    playlist_position INTEGER,
    created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_listen_events_track ON listen_events(track_id, event_type);
CREATE INDEX IF NOT EXISTS idx_listen_events_mood ON listen_events(mood, created_at);
CREATE INDEX IF NOT EXISTS idx_listen_events_created ON listen_events(created_at);
