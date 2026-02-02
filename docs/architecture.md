# Architecture

Drift FM is a mood-based internet radio you host yourself. Pick a mood, get continuous music. No algorithms, no accounts, no tracking.

---

## Design Principles

1. **Simple by default.** Single binary, SQLite, vanilla JS. No build step.
2. **Bring your own music.** The platform is the player + shuffle engine. You supply content.
3. **Mood-first.** Every track belongs to a mood. Playlists are per-mood, weighted shuffle.
4. **No frameworks.** Go stdlib for HTTP, vanilla JS for frontend, SQLite for storage.
5. **Deploy anywhere.** Runs on a $5 VPS, a Raspberry Pi, or a container.

---

## System Overview

```
┌─────────────┐     ┌──────────────┐     ┌──────────┐
│   Browser   │────▶│  Go Server   │────▶│  SQLite  │
│  (vanilla   │◀────│  (net/http)  │     │  (WAL)   │
│   JS/CSS)   │     │              │     └──────────┘
└─────────────┘     │  :8080       │
                    │              │────▶ Audio files
                    └──────────────┘     (local or S3)
```

### Request Flow

1. Browser loads `/` → serves `web/index.html` (single page)
2. JS calls `GET /api/moods` → returns mood list with track counts
3. User picks mood → JS calls `GET /api/moods/:mood/playlist`
4. Server generates weighted-shuffle playlist from SQLite
5. JS plays tracks sequentially via `<audio>` element
6. Audio served from local filesystem or S3-presigned URLs

---

## Backend Packages

```
cmd/server/          Entry point, wiring
internal/
├── api/             HTTP handlers, routing, middleware
├── audio/           Audio file serving (local + S3 adapters)
├── cache/           In-memory cache with TTL
├── config/          Environment-based configuration
├── inventory/       SQLite track management, queries
├── metrics/         Runtime and application metrics
└── radio/           Playlist generation, weighted shuffle algorithm
```

### Key Design Decisions

**SQLite over Postgres/MySQL:** A music library of thousands of tracks fits comfortably in SQLite. WAL mode handles concurrent reads. No external dependencies to manage.

**Pure Go SQLite (modernc.org/sqlite):** No CGO required. Cross-compiles cleanly to any platform. Slightly slower than CGO sqlite3 but the workload is tiny.

**Weighted shuffle:** Tracks aren't purely random. The shuffle algorithm weights by:
- Track freshness (newer tracks get a small boost)
- Play history (recently played tracks are deprioritized)
- Energy matching (requested energy level influences selection)

**Content-addressed paths:** Audio files live at `audio/tracks/<prefix>/<slug>-<hex-id>.mp3`. The hex prefix distributes files across subdirectories for filesystem performance.

**No SPA framework:** The player is ~500 lines of vanilla JS. CSS variables handle theming. No build step, no node_modules, no bundler.

---

## Data Model

### tracks

| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER | Primary key |
| title | TEXT | Track title |
| artist | TEXT | Artist name |
| mood | TEXT | Primary mood (focus, calm, etc.) |
| file_path | TEXT | Path relative to audio root |
| duration_seconds | INTEGER | Track length |
| energy | TEXT | low / medium / high |
| intensity | INTEGER | 1-10 scale |
| tempo_bpm | INTEGER | Beats per minute |
| has_vocals | BOOLEAN | Instrumental flag |
| lyrics | TEXT | Display lyrics (cleaned) |
| status | TEXT | approved / pending / rejected |
| variant_group | TEXT | Links variations of same track |

### track_moods

| Column | Type | Description |
|--------|------|-------------|
| track_id | INTEGER | FK to tracks |
| mood | TEXT | Mood name |
| weight | REAL | 0.0-1.0, strength of association |

### play_stats

| Column | Type | Description |
|--------|------|-------------|
| track_id | INTEGER | FK to tracks |
| play_count | INTEGER | Total plays |
| skip_count | INTEGER | Total skips |
| last_played_at | DATETIME | Last play timestamp |

### listen_events

| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER | Primary key |
| track_id | INTEGER | FK to tracks |
| mood | TEXT | Mood during play |
| event_type | TEXT | play / skip / complete |
| listen_seconds | REAL | Duration listened |
| playlist_position | INTEGER | Position in playlist |
| created_at | DATETIME | Event timestamp |

---

## Frontend Architecture

```
web/
├── index.html       Single page, no routing
├── css/
│   └── style.css    CSS variables for theming, responsive
├── js/
│   ├── app.js       Main application logic
│   ├── player.js    Audio playback engine
│   ├── api.js       Server communication
│   └── ui.js        DOM manipulation
├── icons/           PWA icons
├── manifest.json    PWA manifest
└── sw.js            Service worker (offline support)
```

**Player engine:** Uses HTML5 `<audio>` element with preloading. When current track reaches 80% completion, the next track starts preloading to eliminate gaps between tracks.

**Mood selection:** Mood grid with visual indicators. Selecting a mood fetches a fresh playlist and begins playback immediately.

**No tracking:** No analytics, no cookies, no third-party scripts. Listen events are stored locally in SQLite for playlist optimization only.

---

## Audio Storage

### Local Mode (default)

Audio files live on the same server:

```
audio/tracks/
├── 0/
│   ├── morning-coffee-0010.mp3
│   └── deep-current-04a0.mp3
├── 1/
│   ├── quiet-studio-0041.mp3
│   └── typewriter-04a1.mp3
...
```

Files are served directly by the Go server with appropriate cache headers.

### S3 Mode (cloud)

Set `AUDIO_STORE_TYPE=s3` with S3-compatible credentials. Audio URLs become presigned S3 URLs with short expiry. Works with AWS S3, Cloudflare R2, MinIO, etc.

---

## Adding Custom Moods

Moods are derived from the `tracks.mood` column and `track_moods` table. To add a new mood:

1. Import tracks with the new mood name
2. The API automatically includes it in `/api/moods`
3. The frontend dynamically renders mood buttons

No code changes needed.

---

## Performance Characteristics

- **Startup:** < 1 second (single binary + SQLite)
- **Memory:** ~20-30 MB for a library of 200 tracks
- **Playlist generation:** < 5ms for weighted shuffle of 50 tracks
- **Concurrent users:** SQLite WAL handles dozens of readers comfortably
- **Storage:** Bottleneck is audio files, not the application

For high-traffic deployments, put a CDN (Cloudflare, etc.) in front of audio files and use S3 mode.
