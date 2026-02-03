# Quickstart

Get Drift FM running with your own music in a few minutes.

---

## Prerequisites

| Tool | Version | Check | Install (macOS) |
|------|---------|-------|-----------------|
| Go | 1.25+ | `go version` | `brew install go` |
| SQLite3 | any | `sqlite3 --version` | `brew install sqlite` |
| ffmpeg | any | `ffmpeg -version` | `brew install ffmpeg` |

ffmpeg is only needed for audio import (duration detection) and normalization. The server itself has no ffmpeg dependency.

---

## 1. Clone and Initialize

```bash
git clone https://github.com/1mb-dev/driftfm.git
cd driftfm
make db-init
```

This creates:
- `data/inventory.db` — SQLite database with the schema
- `audio/tracks/` — directory where imported audio files are stored

---

## 2. Prepare Your Music

Gather mp3 files and organize them by mood. Drift FM has 4 moods:

| Mood | What it's for | Notes |
|------|---------------|-------|
| `focus` | Deep work, concentration | **Instrumental only** — vocal tracks are filtered out |
| `calm` | Relaxation, winding down | Ambient, lo-fi, acoustic |
| `energize` | Workouts, upbeat tasks | Higher energy, faster tempo |
| `late_night` | Late sessions, atmosphere | Downtempo, atmospheric |

A good starting point is 5-10 tracks per mood. More is better — the shuffle algorithm avoids repeats based on your library size.

### Audio format

- **MP3 only** — other formats are not supported
- Any bitrate works, but 128-320 kbps is typical
- No minimum or maximum duration

### Optional: normalize volume

For consistent playback volume across tracks, normalize before importing:

```bash
# Single file — outputs normalized copy alongside original
make normalize FILE=~/music/track.mp3

# Batch — normalize all mp3s in a directory
./scripts/normalize-batch.sh ~/music/focus-tracks/
```

This applies EBU R128 loudness normalization targeting -16 LUFS (streaming standard). The original files are not modified — normalized copies are created with a `-norm` suffix.

---

## 3. Import Tracks

### Option A: Batch import (recommended)

Import an entire directory at once. The script prompts you to pick a mood.

```bash
make import-batch ARGS="~/music/focus-tracks"
```

```
Found 8 MP3 file(s) to import.

Select a mood:
  1) focus
  2) calm
  3) energize
  4) late_night

Mood [1-4]: 1

Importing 8 file(s) as 'focus'...
```

Run this once per mood directory:

```bash
make import-batch ARGS="~/music/focus-tracks"
make import-batch ARGS="~/music/calm-tracks"
make import-batch ARGS="~/music/energize-tracks"
make import-batch ARGS="~/music/late-night-tracks"
```

### Option B: Single track with metadata

For more control over individual tracks:

```bash
make import FILE=~/music/song.mp3 MOOD=calm
```

Or use the script directly for full metadata:

```bash
./scripts/import-track.sh ~/music/song.mp3 \
  --mood calm \
  --title "Ocean Waves" \
  --artist "Your Name" \
  --energy low \
  --bpm 72 \
  --intensity 8 \
  --time evening
```

### Vocals and lyrics

The import script auto-detects vocal tracks from a companion `.txt` file:

```
~/music/
  ocean-waves.mp3              ← no .txt → instrumental
  marmalade.mp3                ← has .txt → vocal
  marmalade.txt                ← lyrics (or empty for vocals without lyrics)
```

The `.txt` file can contain lyrics (plain text, one line per lyric, blank lines between stanzas) or be empty — both mark the track as vocal.

With these files in place, batch import works with no extra flags:

```bash
make import-batch ARGS="~/music/my-tracks"
```

**Important:** Focus mood enforces instrumental-only. Tracks with vocals won't appear in focus playlists.

Available flags:

| Flag | Default | Description |
|------|---------|-------------|
| `--mood` | *(required)* | focus, calm, energize, or late_night |
| `--title` | derived from filename | Display title |
| `--artist` | Drift FM | Artist name |
| `--energy` | low | low, medium, or high |
| `--bpm` | — | Tempo in BPM |
| `--key` | — | Musical key (C, Am, Dm, etc.) |
| `--vocals` | no | Flag: track has vocals |
| `--lyrics` | — | Path to a lyrics text file |
| `--intensity` | 5 | 1-10 scale (1=light, 10=deep) |
| `--time` | any | morning, afternoon, evening, night, or any |
| `--dry-run` | — | Preview without importing |

### Preview before importing

```bash
./scripts/import-track.sh song.mp3 --mood focus --dry-run
```

### What import does

1. Reads duration via ffprobe
2. Copies the file to `audio/tracks/<prefix>/<slug>-<id>.mp3` (content-addressed path)
3. Inserts a row into the `tracks` table with metadata
4. Reports the inventory count for that mood

Your original files are not modified or moved.

---

## 4. Run the Server

```bash
make run
```

Open [http://localhost:8080](http://localhost:8080) in your browser. Pick a mood and hit play.

### With hot reload (development)

```bash
# Install air first (one-time)
go install github.com/air-verse/air@latest

make dev
```

---

## 5. Verify

Check that everything is working:

```bash
# Health check
curl -s localhost:8080/health

# List moods with track counts
curl -s localhost:8080/api/moods | python3 -m json.tool

# Get a playlist for a mood
curl -s localhost:8080/api/moods/focus/playlist | python3 -m json.tool
```

Or run the automated smoke test:

```bash
make smoke
```

---

## File Layout After Setup

```
driftfm/
├── data/
│   └── inventory.db          # SQLite database (gitignored)
├── audio/
│   └── tracks/               # Imported audio files (gitignored)
│       ├── 0/
│       │   └── ocean-waves-0001.mp3
│       ├── 1/
│       │   └── deep-focus-0011.mp3
│       └── ...
├── config.yaml               # Server config (port, paths)
└── bin/
    └── server                # Compiled binary (after make build)
```

Audio files are organized into subdirectories by the last hex digit of the track ID. This is automatic — you don't need to manage it.

---

## Library Maintenance

### Check library health

```bash
./scripts/audio-hygiene.sh check
```

Reports: duplicates, orphaned files (in filesystem but not DB), missing files (in DB but not filesystem), naming issues.

### Fix issues

```bash
./scripts/audio-hygiene.sh fix
```

### Run database migrations

After pulling updates that include schema changes:

```bash
make db-migrate
```

---

## Configuration

The server reads `config.yaml` by default. Override any value with environment variables using the `DRIFT_` prefix:

```bash
# Change port
DRIFT_SERVER_PORT=3000 make run

# Change database path
DRIFT_DATABASE_PATH=/var/lib/driftfm/inventory.db make run

# Change audio directory
DRIFT_AUDIO_LOCAL_PATH=/mnt/music make run
```

See `config.yaml` for all available options.

---

## Troubleshooting

**"Database not found"** — Run `make db-init` first.

**"ffprobe: command not found"** — Install ffmpeg: `brew install ffmpeg` (macOS) or `apt install ffmpeg` (Linux).

**No tracks showing up** — Check that you imported with `status='approved'` (this is the default). Verify with:
```bash
sqlite3 data/inventory.db "SELECT mood, COUNT(*) FROM tracks WHERE status='approved' GROUP BY mood;"
```

**Focus mood shows no tracks** — Focus enforces instrumental-only. If you imported tracks with `--vocals`, they won't appear in focus playlists. Re-import without the `--vocals` flag, or use a different mood.

**Port already in use** — Change the port: `DRIFT_SERVER_PORT=3001 make run`
