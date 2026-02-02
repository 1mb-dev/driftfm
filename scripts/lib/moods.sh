# Shared mood definitions for Drift FM import pipeline
# Source this file: . scripts/lib/moods.sh
#
# Single source of truth for mood values. Update MOOD_LIST when adding
# a new mood — all scripts that source this file pick it up automatically.

MOOD_LIST="focus calm energize late_night"

# Build sed alternation pattern from MOOD_LIST: (focus|calm|energize|late_night)
_MOOD_ALT=$(echo "$MOOD_LIST" | tr ' ' '|')

# Strip mood prefix from filename
# Usage: CLEAN=$(strip_mood_prefix "$FILENAME")
# Example: strip_mood_prefix "calm-drifting-home-0030.mp3" → "drifting-home-0030.mp3"
strip_mood_prefix() {
    echo "$1" | sed -E "s/^(${_MOOD_ALT})-//"
}

# Validate mood value (canonical form: underscore, not hyphen)
# Usage: validate_mood "$MOOD" || exit 1
validate_mood() {
    case "$1" in
        focus|calm|late_night|energize) return 0 ;;
        *) return 1 ;;
    esac
}
