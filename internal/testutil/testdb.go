// Package testutil provides shared test helpers for database setup.
package testutil

// SchemaDDL is the canonical test schema matching the production database.
// Used by test helpers across packages to avoid DDL duplication.
const SchemaDDL = `
	CREATE TABLE tracks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		file_path TEXT NOT NULL UNIQUE,
		title TEXT,
		artist TEXT DEFAULT 'Drift FM',
		mood TEXT NOT NULL DEFAULT 'focus',
		energy TEXT NOT NULL DEFAULT 'low',
		tempo_bpm INTEGER,
		has_vocals INTEGER NOT NULL DEFAULT 0,
		musical_key TEXT,
		intensity INTEGER DEFAULT 5,
		time_affinity TEXT DEFAULT 'any',
		lyrics TEXT,
		duration_seconds INTEGER NOT NULL,
		status TEXT NOT NULL DEFAULT 'approved',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE play_stats (
		file_path TEXT PRIMARY KEY NOT NULL REFERENCES tracks(file_path) ON DELETE CASCADE,
		play_count INTEGER NOT NULL DEFAULT 0,
		last_played_at DATETIME,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE listen_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		track_id INTEGER NOT NULL REFERENCES tracks(id),
		mood TEXT NOT NULL,
		event_type TEXT NOT NULL CHECK (event_type IN ('play', 'skip', 'complete')),
		listen_seconds INTEGER NOT NULL DEFAULT 0,
		playlist_position INTEGER,
		created_at DATETIME NOT NULL DEFAULT (datetime('now'))
	);
`
