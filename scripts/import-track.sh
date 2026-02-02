#!/bin/bash
#
# Import a track into Drift FM inventory with full metadata
# Usage: ./scripts/import-track.sh <file> --mood <mood> [options]
#
# Options:
#   --mood <mood>         Primary mood (focus, calm, late_night, energize) [required]
#   --title <title>       Display title (default: derived from filename)
#   --artist <artist>     Artist name (default: Drift FM)
#   --energy <energy>     Energy level: low, medium, high (default: low)
#   --bpm <bpm>           Tempo in BPM
#   --key <key>           Musical key (e.g., C, Am, Dm)
#   --vocals              Track has vocals (default: no vocals)
#   --lyrics <file>       Path to lyrics file
#   --intensity <1-10>    Moodlet intensity (1=light, 10=deep, default: 5)
#   --time <affinity>     Time affinity: morning, afternoon, evening, night, any (default: any)
#   --dry-run             Show what would be imported without doing it
#
# Examples:
#   ./scripts/import-track.sh song.mp3 --mood calm --title "Ocean Waves"
#   ./scripts/import-track.sh song.mp3 --mood energize --bpm 120 --energy high --vocals
#   ./scripts/import-track.sh song.mp3 --mood late_night --lyrics lyrics.txt --intensity 8

set -e

# shellcheck source=lib/clean-lyrics.sh
. "$(dirname "$0")/lib/clean-lyrics.sh"
# shellcheck source=lib/moods.sh
. "$(dirname "$0")/lib/moods.sh"

# Parse arguments
INPUT_FILE=""
MOOD=""
TITLE=""
ARTIST="Drift FM"
ENERGY="low"
BPM=""
KEY=""
HAS_VOCALS=0
LYRICS_FILE=""
INTENSITY=5
TIME_AFFINITY="any"
DRY_RUN=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --mood) MOOD="$2"; shift 2 ;;
        --title) TITLE="$2"; shift 2 ;;
        --artist) ARTIST="$2"; shift 2 ;;
        --energy) ENERGY="$2"; shift 2 ;;
        --bpm) BPM="$2"; shift 2 ;;
        --key) KEY="$2"; shift 2 ;;
        --vocals) HAS_VOCALS=1; shift ;;
        --lyrics) LYRICS_FILE="$2"; shift 2 ;;
        --intensity) INTENSITY="$2"; shift 2 ;;
        --time) TIME_AFFINITY="$2"; shift 2 ;;
        --dry-run) DRY_RUN=true; shift ;;
        -h|--help)
            head -30 "$0" | tail -n +2 | sed 's/^# //' | sed 's/^#//'
            exit 0
            ;;
        *)
            if [[ -z "$INPUT_FILE" && -f "$1" ]]; then
                INPUT_FILE="$1"
            else
                echo "Unknown option or file not found: $1"
                exit 1
            fi
            shift
            ;;
    esac
done

DB="data/inventory.db"

# Validate required arguments
if [[ -z "$INPUT_FILE" ]]; then
    echo "Error: Input file required"
    echo "Usage: $0 <file> --mood <mood> [options]"
    exit 1
fi

if [[ ! -f "$INPUT_FILE" ]]; then
    echo "Error: File not found: $INPUT_FILE"
    exit 1
fi

if [[ -z "$MOOD" ]]; then
    echo "Error: --mood is required"
    exit 1
fi

if ! validate_mood "$MOOD"; then
    echo "Error: Invalid mood '$MOOD'. Must be: focus, calm, late_night, energize"
    exit 1
fi

if [[ ! "$ENERGY" =~ ^(low|medium|high)$ ]]; then
    echo "Error: Invalid energy. Must be: low, medium, high"
    exit 1
fi

if [[ ! "$TIME_AFFINITY" =~ ^(morning|afternoon|evening|night|any)$ ]]; then
    echo "Error: Invalid time affinity. Must be: morning, afternoon, evening, night, any"
    exit 1
fi

if [[ "$INTENSITY" -lt 1 || "$INTENSITY" -gt 10 ]]; then
    echo "Error: Intensity must be 1-10"
    exit 1
fi

if [[ ! -f "$DB" ]]; then
    echo "Error: Database not found. Run 'make db-init' first."
    exit 1
fi

# File will be named after we know the track ID
# Format: audio/tracks/<prefix>/<title-slug>-<id-hex>.mp3
FILE_EXT="${INPUT_FILE##*.}"
TEMP_DIR="audio/tracks/tmp"
TEMP_FILENAME="importing_$(openssl rand -hex 4).${FILE_EXT}"
TEMP_PATH="${TEMP_DIR}/${TEMP_FILENAME}"

# Derive title from filename if not provided
if [[ -z "$TITLE" ]]; then
    BASENAME=$(basename "$INPUT_FILE")
    # Remove extension and clean up
    TITLE="${BASENAME%.*}"
    # Remove common suffixes like (1), _timestamp, etc.
    TITLE=$(echo "$TITLE" | sed -E 's/ \([0-9]+\)$//' | sed -E 's/_[0-9]{10,}$//')
    # Replace underscores/hyphens with spaces
    TITLE=$(echo "$TITLE" | tr '_-' '  ' | sed 's/  */ /g')
fi

# Get duration
DURATION=$(ffprobe -v quiet -show_entries format=duration -of csv=p=0 "$INPUT_FILE" | cut -d'.' -f1)

# Read and clean lyrics if file provided
LYRICS=""
if [[ -n "$LYRICS_FILE" && -f "$LYRICS_FILE" ]]; then
    LYRICS=$(cat "$LYRICS_FILE" | clean_lyrics)
    HAS_VOCALS=1  # Assume vocals if lyrics provided
fi

# Display what we're importing
echo "═══════════════════════════════════════════════════════════════"
echo "Track Import"
echo "═══════════════════════════════════════════════════════════════"
echo ""
echo "  Source:     $INPUT_FILE"
echo "  Dest:       audio/tracks/<prefix>/<title>-<id>.mp3"
echo "  Duration:   ${DURATION}s ($(( DURATION / 60 )):$(printf '%02d' $(( DURATION % 60 ))))"
echo ""
echo "  Title:      $TITLE"
echo "  Artist:     $ARTIST"
echo "  Mood:       $MOOD"
echo "  Energy:     $ENERGY"
echo "  Intensity:  $INTENSITY/10"
echo "  Time:       $TIME_AFFINITY"
if [[ -n "$BPM" ]]; then
    echo "  BPM:        $BPM"
fi
if [[ -n "$KEY" ]]; then
    echo "  Key:        $KEY"
fi
if [[ $HAS_VOCALS -eq 1 ]]; then
    echo "  Vocals:     yes"
fi
echo ""

if $DRY_RUN; then
    echo "[DRY RUN] Would import track. Use without --dry-run to proceed."
    exit 0
fi

# Copy file to temp location first
mkdir -p "$TEMP_DIR"
cp "$INPUT_FILE" "$TEMP_PATH"

# Build SQL for insert (with temp path initially)
SQL="INSERT INTO tracks (
    file_path, title, artist, mood, energy, tempo_bpm, has_vocals,
    musical_key, intensity, time_affinity, lyrics, duration_seconds,
    status
) VALUES (
    'tracks/tmp/${TEMP_FILENAME}',
    '$(echo "$TITLE" | sed "s/'/''/g")',
    '$(echo "$ARTIST" | sed "s/'/''/g")',
    '$MOOD',
    '$ENERGY',
    $([ -n "$BPM" ] && echo "$BPM" || echo "NULL"),
    $HAS_VOCALS,
    $([ -n "$KEY" ] && echo "'$KEY'" || echo "NULL"),
    $INTENSITY,
    '$TIME_AFFINITY',
    $([ -n "$LYRICS" ] && echo "'$(echo "$LYRICS" | sed "s/'/''/g")'" || echo "NULL"),
    $DURATION,
    'approved'
);"

# Execute insert and get ID
NEW_ID=$(sqlite3 "$DB" "$SQL SELECT last_insert_rowid();")

if [[ -z "$NEW_ID" || "$NEW_ID" == "0" ]]; then
    echo "Error: Failed to import track"
    rm -f "$TEMP_PATH"
    exit 1
fi

# Generate final filename using track ID
# Pattern: audio/tracks/<prefix>/<title-slug>-<id-hex>.mp3
TITLE_SLUG=$(echo "$TITLE" | tr '[:upper:]' '[:lower:]' | \
             sed 's/[^a-z0-9 ]//g' | \
             sed 's/  */ /g' | \
             sed 's/ /-/g' | \
             cut -c1-25 | \
             sed 's/-$//')
ID_HEX=$(printf "%04x" "$NEW_ID")
PREFIX="${ID_HEX:3:1}"
FINAL_FILENAME="${TITLE_SLUG}-${ID_HEX}.${FILE_EXT}"
FINAL_DIR="audio/tracks/${PREFIX}"
FINAL_PATH="${FINAL_DIR}/${FINAL_FILENAME}"
DB_PATH="tracks/${PREFIX}/${FINAL_FILENAME}"

# Move file to final content-addressed location
mkdir -p "$FINAL_DIR"
mv -f "$TEMP_PATH" "$FINAL_PATH"

# Update DB with final path
sqlite3 "$DB" "UPDATE tracks SET file_path='$DB_PATH' WHERE id=$NEW_ID;"

echo "✓ Track imported with ID: $NEW_ID"
echo "✓ File: $DB_PATH"
echo "✓ Mood: $MOOD"

echo ""
echo "═══════════════════════════════════════════════════════════════"

# Show inventory stats
echo ""
echo "Inventory for $MOOD:"
sqlite3 "$DB" "SELECT COUNT(*) || ' tracks, ' || COALESCE(SUM(duration_seconds)/60, 0) || ' minutes' FROM tracks WHERE mood='$MOOD' AND status='approved';"
