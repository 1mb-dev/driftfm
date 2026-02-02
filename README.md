# Drift FM

Mood radio you host yourself.

Drop in your mp3s, tag them by mood, hit play. Continuous, shuffled playback with weighted playlists. No accounts, no tracking, no frameworks.

**Stack:** Go + SQLite + vanilla JS

**Live demo:** [drift.1mb.dev](https://drift.1mb.dev)

---

## Quick Start

```bash
# Build
make build

# Import your music (interactive — assigns mood per track)
./scripts/import-tracks.sh /path/to/your/music

# Run
make run
# → http://localhost:8080
```

## How It Works

1. **You provide the music.** Drop mp3 files into `audio/tracks/` or use the import script.
2. **Tag by mood.** Each track gets a mood (focus, calm, energize, late_night) and optional metadata.
3. **Hit play.** The player serves continuous, weighted-shuffle playlists per mood. No gaps.

The player is a single-page vanilla JS app. No build step, no npm, no React. The backend is a single Go binary with SQLite. Deploy anywhere.

---

## Moods

| Mood | Vibe | Example Use |
|------|------|-------------|
| **focus** | Instrumental, ambient, post-rock | Deep work, coding |
| **calm** | Soft, gentle, meditative | Winding down, reading |
| **energize** | Upbeat, driving, anthemic | Morning, exercise |
| **late_night** | Chillwave, lo-fi, nocturnal | Late sessions, unwinding |

You can add custom moods by editing the configuration.

---

## Import Your Music

### Quick Import

```bash
# Interactive: walks through each file, you pick the mood
./scripts/import-tracks.sh /path/to/music/

# Batch: assign all files in a directory to one mood
./scripts/import-tracks.sh /path/to/focus-tracks/ --mood focus
```

### Manual Import

```bash
# Copy files to the tracks directory
cp my-track.mp3 audio/tracks/

# Import into database with metadata
sqlite3 data/inventory.db "INSERT INTO tracks (title, mood, file_path, energy, status) VALUES ('My Track', 'focus', 'tracks/my-track.mp3', 'medium', 'approved');"
```

### Track Requirements

- **Format:** MP3 (128-320 kbps)
- **Duration:** 1-10 minutes recommended
- **Naming:** Lowercase, hyphens, no spaces (e.g., `morning-coffee.mp3`)

---

## Configuration

Environment variables (or `.env` file):

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Server port |
| `DB_PATH` | `data/inventory.db` | SQLite database path |
| `AUDIO_DIR` | `audio/tracks` | Local audio directory |
| `AUDIO_STORE_TYPE` | `local` | `local` or `s3` for cloud storage |

For S3/R2 cloud storage, see [deploy/README.md](deploy/README.md).

---

## Development

```bash
make build          # Build binary
make run            # Run server (localhost:8080)
make test           # Run tests
make fmt            # Format code
make lint           # Lint check
make check          # Full quality gate (fmt, vet, lint, test)
```

---

## Deploy

### Bare Metal / VPS

```bash
make build-linux                    # Cross-compile for Linux
scp bin/server-linux user@host:/opt/driftfm/bin/server
# Set up systemd service (see deploy/driftfm.service)
```

### Docker

```bash
docker build -t driftfm .
docker run -p 8080:8080 -v ./audio:/app/audio -v ./data:/app/data driftfm
```

### With Reverse Proxy (Caddy)

See [deploy/README.md](deploy/README.md) for Caddy + TLS setup.

---

## API

| Endpoint | Description |
|----------|-------------|
| `GET /api/moods` | List moods with track counts |
| `GET /api/moods/:mood/playlist` | Shuffled playlist for mood |
| `POST /api/tracks/:id/play` | Record play event |
| `GET /health` | Health check |
| `GET /ready` | Readiness probe |

---

## Architecture

```
Browser ←→ Go server ←→ SQLite
              ↓
         Audio files (local or S3/R2)
```

- **Backend:** Single Go binary, net/http, no framework
- **Database:** SQLite with WAL mode (pure Go driver, no CGO)
- **Frontend:** Vanilla JS, CSS variables, Web Audio API
- **Audio:** Local filesystem or S3-compatible storage (R2, MinIO)

See [docs/architecture.md](docs/architecture.md) for details.

---

## License

MIT — see [LICENSE](LICENSE).
