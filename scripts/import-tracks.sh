#!/usr/bin/env bash
set -euo pipefail

# import-tracks.sh — Interactive batch import of MP3 files into Drift FM
#
# Usage:
#   ./scripts/import-tracks.sh <directory-or-file> [...]
#   ./scripts/import-tracks.sh ~/music/ambient/
#   ./scripts/import-tracks.sh track1.mp3 track2.mp3
#
# Prompts for mood selection, then imports each MP3 via import-track.sh.

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
. "$SCRIPT_DIR/lib/moods.sh"

usage() {
    echo "Usage: $0 <directory-or-file> [...]"
    echo ""
    echo "Import MP3 files into Drift FM inventory."
    echo "Accepts directories (scanned for .mp3 files) or individual files."
    echo ""
    echo "Examples:"
    echo "  $0 ~/music/ambient/"
    echo "  $0 track1.mp3 track2.mp3"
    exit 1
}

if [ $# -eq 0 ]; then
    usage
fi

# Collect all MP3 files from arguments
FILES=()
for arg in "$@"; do
    if [ -d "$arg" ]; then
        while IFS= read -r -d '' f; do
            FILES+=("$f")
        done < <(find "$arg" -maxdepth 1 -name '*.mp3' -print0 | sort -z)
    elif [ -f "$arg" ]; then
        FILES+=("$arg")
    else
        echo "Warning: '$arg' is not a file or directory, skipping."
    fi
done

if [ ${#FILES[@]} -eq 0 ]; then
    echo "No MP3 files found."
    exit 1
fi

echo "Found ${#FILES[@]} MP3 file(s) to import."
echo ""

# Prompt for mood
echo "Select a mood:"
echo "  1) focus"
echo "  2) calm"
echo "  3) energize"
echo "  4) late_night"
echo ""
printf "Mood [1-4]: "
read -r CHOICE

case "$CHOICE" in
    1) MOOD="focus" ;;
    2) MOOD="calm" ;;
    3) MOOD="energize" ;;
    4) MOOD="late_night" ;;
    *)
        echo "Invalid choice. Aborting."
        exit 1
        ;;
esac

echo ""
echo "Importing ${#FILES[@]} file(s) as '$MOOD'..."
echo ""

# Import each file
SUCCESS=0
FAILED=0
for f in "${FILES[@]}"; do
    echo "--- $(basename "$f")"
    if "$SCRIPT_DIR/import-track.sh" "$f" --mood "$MOOD"; then
        SUCCESS=$((SUCCESS + 1))
    else
        FAILED=$((FAILED + 1))
        echo "  FAILED"
    fi
done

echo ""
echo "=== Import Summary ==="
echo "  Mood:      $MOOD"
echo "  Imported:  $SUCCESS"
if [ "$FAILED" -gt 0 ]; then
    echo "  Failed:    $FAILED"
fi
echo "  Total:     ${#FILES[@]}"
