# Shared lyrics cleaning for Drift FM import pipeline
# Source this file: . scripts/lib/clean-lyrics.sh

clean_lyrics() {
    # Remove any line that is only a structural tag like [Verse 1], [Final Chorus], etc.
    sed -E '/^[[:space:]]*\[.*\][[:space:]]*$/d' | \
    # Remove leading whitespace from each line
    sed 's/^[[:space:]]*//' | \
    # Collapse 3+ consecutive blank lines to 1 (preserve verse spacing)
    sed -E '/^$/N;/^\n$/N;/^\n\n$/d'
}
