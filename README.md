# Drift FM

[![CI](https://github.com/1mb-dev/driftfm/actions/workflows/ci.yml/badge.svg)](https://github.com/1mb-dev/driftfm/actions/workflows/ci.yml)

Mood radio you host yourself.

Drop in your mp3s, tag them by mood, hit play. Continuous shuffled playback per mood. No accounts, no tracking, no frameworks.

**Stack:** Go + SQLite + vanilla JS

---

## Quick Start

```bash
# 1. Initialize the database
make db-init

# 2. Import your music (interactive — pick mood per batch)
make import-batch ARGS="/path/to/your/music"

# 3. Run
make run
# → http://localhost:8080
```

---

## End-to-End Workflow

### 1. Set Up

```bash
git clone https://github.com/1mb-dev/driftfm.git
cd driftfm
make db-init
```

Prerequisites: Go 1.25+, SQLite3 CLI, ffmpeg/ffprobe (for audio import and normalization).

### 2. Add Content

**Option A — Batch import (recommended):**

```bash
# Import a directory of MP3s — prompts for mood selection
make import-batch ARGS="/path/to/focus-tracks"
```

**Option B — Single track with metadata:**

```bash
make import FILE=song.mp3 MOOD=focus
```

The import script (`scripts/import-track.sh`) supports additional flags:

```bash
# Full metadata example
./scripts/import-track.sh song.mp3 \
  --mood calm \
  --title "Ocean Waves" \
  --artist "Your Name" \
  --energy low \
  --bpm 72 \
  --intensity 8 \
  --time evening
```

Run `./scripts/import-track.sh --help` for all options.

### 3. Normalize Audio (Optional)

Normalize loudness across your library for consistent playback:

```bash
# Single file
make normalize FILE=audio/tracks/a/ocean-waves-001a.mp3

# Batch (all files in a directory)
./scripts/normalize-batch.sh audio/tracks/
```

### 4. Run the Server

```bash
make run
# → http://localhost:8080
```

Or build a binary:

```bash
make build
./bin/server
```

### 5. Update the Database

After adding new tracks or modifying the schema:

```bash
# Run pending migrations
make db-migrate

# Re-initialize from scratch (destructive)
make db-init
```

---

## Moods

| Mood | Vibe | Example Use |
|------|------|-------------|
| **focus** | Instrumental, ambient, post-rock | Deep work, coding |
| **calm** | Soft, gentle, meditative | Winding down, reading |
| **energize** | Upbeat, driving, anthemic | Morning, exercise |
| **late_night** | Chillwave, lo-fi, nocturnal | Late sessions, unwinding |

---

## Make Targets

```
Development:
  make build          Build the server binary
  make run            Run the server (localhost:8080)
  make dev            Run with hot reload (requires air)
  make test           Run all tests
  make clean          Remove build artifacts

Code Quality:
  make check          Full quality gate (fmt, vet, lint, test)
  make fmt            Format code
  make vet            Run go vet
  make lint           Run linter (requires golangci-lint)

Setup & Database:
  make setup          Create data/ and audio/ directories
  make db-init        Initialize SQLite database
  make db-migrate     Run pending migrations

Audio:
  make import FILE=<path> MOOD=<mood>   Import single track
  make import-batch ARGS=<dir>          Import directory of tracks
  make normalize FILE=<path>            Normalize audio file
```

---

## Configuration

Environment variables override `config.yaml` values (see `.env.example`):

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Server port |
| `DB_PATH` | `data/inventory.db` | SQLite database path |
| `AUDIO_STORE_LOCAL_PATH` | `audio` | Local audio directory |

---

## Deploy

### Bare Metal / VPS

```bash
# Build for Linux
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/server ./cmd/server

# Copy to your server:
scp -r bin/server web/ config.yaml scripts/ user@host:/opt/driftfm/

# On the server:
mkdir -p /opt/driftfm/data /opt/driftfm/audio/tracks
sqlite3 /opt/driftfm/data/inventory.db < /opt/driftfm/scripts/migrations/schema.sql
PORT=8080 /opt/driftfm/server
```

For production, put a reverse proxy (Caddy, nginx) in front for TLS and set up a systemd unit for process management.

---

## API

| Endpoint | Description |
|----------|-------------|
| `GET /api/moods` | List moods with track counts |
| `GET /api/moods/:mood/playlist` | Shuffled playlist for mood |
| `POST /api/tracks/:id/play` | Record listen event |
| `GET /health` | Health check |
| `GET /ready` | Readiness probe |

---

## Architecture

```
Browser ←→ Go server ←→ SQLite
              ↓
         audio/tracks/
```

- **Backend:** Single Go binary, net/http, no framework
- **Database:** SQLite with WAL mode (pure Go driver, no CGO)
- **Frontend:** Vanilla JS, CSS variables, Web Audio API
- **Audio:** Local filesystem (`audio/tracks/`)

---

## License

MIT — see [LICENSE](LICENSE).
