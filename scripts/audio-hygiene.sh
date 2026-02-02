#!/bin/bash
#
# Drift FM Audio Library Hygiene
# Checks and fixes: duplicates, orphans, naming, album art
#
# Usage:
#   ./scripts/audio-hygiene.sh check     - Report issues (no changes)
#   ./scripts/audio-hygiene.sh fix       - Fix all issues
#   ./scripts/audio-hygiene.sh rename    - Rename files to convention
#   ./scripts/audio-hygiene.sh dedupe    - Remove content duplicates
#   ./scripts/audio-hygiene.sh orphans   - Clean orphaned files
#   ./scripts/audio-hygiene.sh artwork   - Add album art (future)

set -e

# shellcheck source=lib/clean-lyrics.sh
. "$(dirname "$0")/lib/clean-lyrics.sh"

DB="data/inventory.db"
AUDIO_DIR="audio"

# Cross-platform MD5 (macOS uses md5, Linux uses md5sum)
md5_hash() {
    if command -v md5 &>/dev/null; then
        md5 -q "$1"
    else
        md5sum "$1" | cut -d' ' -f1
    fi
}

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

header() {
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
}

# Generate URL-friendly filename from title and track ID
# New structure: tracks/<prefix>/<title-slug>-<id>.mp3
generate_filename() {
    local title="$1"
    local track_id="$2"

    # Convert title to slug: lowercase, replace spaces with hyphens, remove special chars
    local title_slug=$(echo "$title" | tr '[:upper:]' '[:lower:]' | \
                 sed 's/[^a-z0-9 ]//g' | \
                 sed 's/  */ /g' | \
                 sed 's/ /-/g' | \
                 cut -c1-25 | \
                 sed 's/-$//')

    # Use track ID as hex for uniqueness (4 chars)
    local id_hex=$(printf "%04x" "$track_id")

    echo "${title_slug}-${id_hex}.mp3"
}

# Get prefix directory from track ID (last hex char for better distribution)
get_prefix() {
    local track_id="$1"
    local id_hex=$(printf "%04x" "$track_id")
    echo "${id_hex:3:1}"
}

# Check for content duplicates
check_duplicates() {
    echo -e "\n${YELLOW}Content Duplicates:${NC}"

    # Build hash list
    local hash_file=$(mktemp)
    while IFS= read -r f; do
        echo "$(md5_hash "$f") $f" >> "$hash_file"
    done < <(find "$AUDIO_DIR" -name "*.mp3" -type f)

    local dupes=$(cut -d' ' -f1 "$hash_file" | sort | uniq -c | sort -rn | awk '$1 > 1 {print $2}')

    if [ -z "$dupes" ]; then
        echo -e "  ${GREEN}No duplicates found${NC}"
        rm -f "$hash_file"
        return 0
    fi

    local count=0
    for hash in $dupes; do
        ((count++))
        echo -e "  ${RED}Duplicate set $count:${NC}"
        grep "^$hash " "$hash_file" | cut -d' ' -f2- | while read -r f; do
            echo "    $f"
        done
    done

    rm -f "$hash_file"
    return 1
}

# Check for orphaned files (in filesystem but not in DB)
check_orphans() {
    echo -e "\n${YELLOW}Orphaned Files (no DB record):${NC}"

    local orphans=""
    while IFS= read -r f; do
        if ! sqlite3 "$DB" "SELECT 1 FROM tracks WHERE file_path='$f' LIMIT 1;" | grep -q 1; then
            echo -e "  ${RED}$f${NC}"
            orphans="$orphans$f\n"
        fi
    done < <(find "$AUDIO_DIR" -name "*.mp3" -type f | sed "s|^$AUDIO_DIR/||")

    if [ -z "$orphans" ]; then
        echo -e "  ${GREEN}No orphans found${NC}"
    fi
}

# Check for missing files (in DB but not in filesystem)
check_missing() {
    echo -e "\n${YELLOW}Missing Files (DB record but no file):${NC}"

    local missing=""
    while IFS= read -r f; do
        if [ ! -f "$AUDIO_DIR/$f" ]; then
            echo -e "  ${RED}$f${NC}"
            missing="$missing$f\n"
        fi
    done < <(sqlite3 "$DB" "SELECT file_path FROM tracks WHERE status='approved';")

    if [ -z "$missing" ]; then
        echo -e "  ${GREEN}No missing files${NC}"
    fi
}

# Check naming conventions
check_naming() {
    echo -e "\n${YELLOW}Non-URL-Friendly Filenames:${NC}"

    local bad=""
    while IFS= read -r f; do
        local basename=$(basename "$f")
        # Check for: uppercase, spaces, parens, underscores
        if echo "$basename" | grep -qE '[A-Z]|[ ()]|_'; then
            echo -e "  ${RED}$basename${NC}"
            bad="$bad$basename\n"
        fi
    done < <(find "$AUDIO_DIR" -name "*.mp3" -type f)

    if [ -z "$bad" ]; then
        echo -e "  ${GREEN}All filenames are URL-friendly${NC}"
    fi
}

# Check album art
check_artwork() {
    echo -e "\n${YELLOW}Album Art Status:${NC}"

    local with_art=0
    local without_art=0

    for f in $(find "$AUDIO_DIR" -name "*.mp3" -type f | head -20); do
        if ffprobe -v quiet -show_streams "$f" 2>/dev/null | grep -q "codec_type=video"; then
            ((with_art++))
        else
            ((without_art++))
        fi
    done

    echo -e "  With art: ${GREEN}$with_art${NC}"
    echo -e "  Without art: ${YELLOW}$without_art${NC} (sample of 20)"
}

# Check lyrics quality/validity
check_lyrics() {
    echo -e "\n${YELLOW}Lyrics Quality Check:${NC}"

    local issues=0

    # Check 1: Instrumental tracks with lyrics (focus mood must be instrumental)
    echo -e "\n  ${BLUE}Checking instrumental tracks with lyrics...${NC}"
    local instrumental_with_lyrics=$(sqlite3 "$DB" "
        SELECT t.id, t.title, t.mood, LENGTH(t.lyrics) as len
        FROM tracks t
        WHERE t.has_vocals = 0
          AND t.lyrics IS NOT NULL
          AND LENGTH(t.lyrics) > 0
          AND t.status = 'approved';
    ")
    if [ -n "$instrumental_with_lyrics" ]; then
        echo -e "  ${RED}Instrumental tracks with lyrics (should not have lyrics):${NC}"
        echo "$instrumental_with_lyrics" | while IFS='|' read -r id title mood len; do
            echo -e "    ${RED}[ID:$id]${NC} $title ($mood) - $len chars"
            ((issues++)) || true
        done
    else
        echo -e "  ${GREEN}✓ No instrumental tracks with lyrics${NC}"
    fi

    # Check 2: Focus mood tracks with vocals (focus must be instrumental)
    echo -e "\n  ${BLUE}Checking focus mood for vocals...${NC}"
    local focus_with_vocals=$(sqlite3 "$DB" "
        SELECT t.id, t.title
        FROM tracks t
        WHERE t.mood = 'focus'
          AND t.has_vocals = 1
          AND t.status = 'approved';
    ")
    if [ -n "$focus_with_vocals" ]; then
        echo -e "  ${RED}Focus tracks with vocals (should be instrumental):${NC}"
        echo "$focus_with_vocals" | while IFS='|' read -r id title; do
            echo -e "    ${RED}[ID:$id]${NC} $title"
            ((issues++)) || true
        done
    else
        echo -e "  ${GREEN}✓ All focus tracks are instrumental${NC}"
    fi

    # Check 3: Lyrics containing structural tags that should be cleaned
    echo -e "\n  ${BLUE}Checking for uncleaned lyrics tags...${NC}"
    local suspicious=$(sqlite3 "$DB" "
        SELECT t.id, t.title,
            CASE
                WHEN LOWER(t.lyrics) LIKE '%[verse%' THEN 'contains [Verse tag'
                WHEN LOWER(t.lyrics) LIKE '%[chorus%' THEN 'contains [Chorus tag'
                WHEN LOWER(t.lyrics) LIKE '%[bridge%' THEN 'contains [Bridge tag'
                ELSE 'unknown'
            END as reason
        FROM tracks t
        WHERE t.status = 'approved'
          AND t.lyrics IS NOT NULL
          AND (
              LOWER(t.lyrics) LIKE '%[verse%'
              OR LOWER(t.lyrics) LIKE '%[chorus%'
              OR LOWER(t.lyrics) LIKE '%[bridge%'
          );
    ")
    if [ -n "$suspicious" ]; then
        echo -e "  ${RED}Tracks with uncleaned lyrics tags:${NC}"
        echo "$suspicious" | while IFS='|' read -r id title reason; do
            echo -e "    ${RED}[ID:$id]${NC} $title - $reason"
            ((issues++)) || true
        done
    else
        echo -e "  ${GREEN}✓ No uncleaned lyrics tags found${NC}"
    fi

    # Check 4: Very short lyrics (likely placeholder or error)
    echo -e "\n  ${BLUE}Checking for suspiciously short lyrics...${NC}"
    local short_lyrics=$(sqlite3 "$DB" "
        SELECT t.id, t.title, LENGTH(t.lyrics) as len
        FROM tracks t
        WHERE t.status = 'approved'
          AND t.has_vocals = 1
          AND t.lyrics IS NOT NULL
          AND LENGTH(t.lyrics) > 0
          AND LENGTH(t.lyrics) < 100;
    ")
    if [ -n "$short_lyrics" ]; then
        echo -e "  ${YELLOW}Vocal tracks with very short lyrics (<100 chars):${NC}"
        echo "$short_lyrics" | while IFS='|' read -r id title len; do
            echo -e "    ${YELLOW}[ID:$id]${NC} $title - $len chars"
            ((issues++)) || true
        done
    else
        echo -e "  ${GREEN}✓ No suspiciously short lyrics${NC}"
    fi

    # Check 5: Duplicate lyrics (same lyrics on different tracks)
    echo -e "\n  ${BLUE}Checking for duplicate lyrics...${NC}"
    local dupe_lyrics=$(sqlite3 "$DB" "
        SELECT GROUP_CONCAT(id, ', ') as ids,
               GROUP_CONCAT(title, ', ') as titles,
               COUNT(*) as cnt
        FROM tracks
        WHERE status = 'approved'
          AND lyrics IS NOT NULL
          AND LENGTH(lyrics) > 50
        GROUP BY lyrics
        HAVING COUNT(*) > 1;
    ")
    if [ -n "$dupe_lyrics" ]; then
        echo -e "  ${RED}Tracks with identical lyrics:${NC}"
        echo "$dupe_lyrics" | while IFS='|' read -r ids titles cnt; do
            echo -e "    ${RED}IDs: $ids${NC}"
            echo -e "    Titles: $titles"
            ((issues++)) || true
        done
    else
        echo -e "  ${GREEN}✓ No duplicate lyrics found${NC}"
    fi

    # Summary
    if [ "$issues" -eq 0 ]; then
        echo -e "\n  ${GREEN}All lyrics checks passed!${NC}"
    else
        echo -e "\n  ${RED}Found lyrics issues to review${NC}"
    fi
}

# Full check
cmd_check() {
    header "Audio Library Hygiene Check"

    echo -e "\n${BLUE}Library Stats:${NC}"
    echo "  Files in filesystem: $(find "$AUDIO_DIR" -name "*.mp3" -type f | wc -l | xargs)"
    echo "  Tracks in database: $(sqlite3 "$DB" "SELECT COUNT(*) FROM tracks WHERE status='approved';")"

    check_duplicates || true
    check_orphans || true
    check_missing || true
    check_naming || true
    check_artwork || true
    check_lyrics || true

    echo ""
}

# Remove duplicate files (keep the one in DB, or first one if none in DB)
cmd_dedupe() {
    header "Removing Content Duplicates"
    
    local removed=0
    
    find "$AUDIO_DIR" -name "*.mp3" -type f -exec md5 -q {} \; 2>/dev/null | \
        sort | uniq -d | while read hash; do
        
        # Find all files with this hash
        local files=()
        while IFS= read -r f; do
            files+=("$f")
        done < <(find "$AUDIO_DIR" -name "*.mp3" -type f -exec sh -c \
            'h=$(md5 -q "$1"); [ "$h" = "'"$hash"'" ] && echo "$1"' _ {} \;)
        
        if [ ${#files[@]} -lt 2 ]; then
            continue
        fi
        
        # Find which one is in DB (if any)
        local keep=""
        for f in "${files[@]}"; do
            local rel_path="${f#$AUDIO_DIR/}"
            if sqlite3 "$DB" "SELECT 1 FROM tracks WHERE file_path='$rel_path' LIMIT 1;" | grep -q 1; then
                keep="$f"
                break
            fi
        done
        
        # If none in DB, keep first
        if [ -z "$keep" ]; then
            keep="${files[0]}"
        fi
        
        # Remove others
        for f in "${files[@]}"; do
            if [ "$f" != "$keep" ]; then
                echo -e "  ${RED}Removing:${NC} $f"
                echo -e "  ${GREEN}Keeping:${NC} $keep"
                rm -f "$f"
                ((removed++))
            fi
        done
    done
    
    echo -e "\n${GREEN}Removed $removed duplicate file(s)${NC}"
}

# Remove orphaned files
cmd_orphans() {
    header "Removing Orphaned Files"
    
    local removed=0
    
    find "$AUDIO_DIR" -name "*.mp3" -type f | sed "s|^$AUDIO_DIR/||" | while read f; do
        if ! sqlite3 "$DB" "SELECT 1 FROM tracks WHERE file_path='$f' LIMIT 1;" | grep -q 1; then
            echo -e "  ${RED}Removing:${NC} $AUDIO_DIR/$f"
            rm -f "$AUDIO_DIR/$f"
            ((removed++)) || true
        fi
    done
    
    echo -e "\n${GREEN}Removed orphaned file(s)${NC}"
}

# Rename files to convention
cmd_rename() {
    header "Renaming Files to Convention"

    echo -e "${YELLOW}Pattern: tracks/<prefix>/<title-slug>-<id>.mp3${NC}\n"

    local renamed=0

    sqlite3 -separator '|' "$DB" "
        SELECT file_path, COALESCE(title, 'untitled'), id
        FROM tracks
        WHERE status='approved'
    " | while IFS='|' read -r file_path title track_id; do
        local old_file="$AUDIO_DIR/$file_path"

        if [ ! -f "$old_file" ]; then
            continue
        fi

        # Generate new filename using track ID for uniqueness
        local new_name=$(generate_filename "$title" "$track_id")
        local prefix=$(get_prefix "$track_id")
        local new_path="tracks/$prefix/$new_name"
        local new_file="$AUDIO_DIR/$new_path"

        # Skip if already correct
        if [ "$file_path" = "$new_path" ]; then
            continue
        fi

        echo -e "  ${YELLOW}$file_path${NC}"
        echo -e "  ${GREEN}→ $new_path${NC}"

        # Ensure target directory exists
        mkdir -p "$AUDIO_DIR/tracks/$prefix"

        # Rename file
        mv -f "$old_file" "$new_file"

        # Update DB
        sqlite3 "$DB" "UPDATE tracks SET file_path='$new_path' WHERE id=$track_id;"

        ((renamed++)) || true
        echo ""
    done

    echo -e "${GREEN}Renamed file(s) to URL-friendly convention${NC}"
}

# Fix missing DB records
cmd_fix_missing() {
    header "Fixing Missing Files"
    
    sqlite3 "$DB" "SELECT id, file_path FROM tracks WHERE status='approved';" | while IFS='|' read -r id path; do
        if [ ! -f "$AUDIO_DIR/$path" ]; then
            echo -e "  ${RED}Missing:${NC} $path (ID: $id)"
            echo -e "  ${YELLOW}Setting status to 'missing'${NC}"
            sqlite3 "$DB" "UPDATE tracks SET status='missing' WHERE id=$id;"
        fi
    done
    
    echo -e "\n${GREEN}Done${NC}"
}

# Clean lyrics: remove [Verse], [Chorus] tags and normalize whitespace
cmd_clean_lyrics() {
    header "Cleaning Lyrics"

    local cleaned=0

    sqlite3 "$DB" "SELECT id, title FROM tracks WHERE lyrics IS NOT NULL AND status='approved';" | \
    while IFS='|' read -r id title; do
        # Get current lyrics
        local lyrics=$(sqlite3 "$DB" "SELECT lyrics FROM tracks WHERE id=$id;")

        local cleaned_lyrics=$(echo "$lyrics" | clean_lyrics)

        # Check if changed
        if [ "$lyrics" != "$cleaned_lyrics" ]; then
            echo -e "  ${GREEN}Cleaned:${NC} $title (ID: $id)"
            # Escape for SQL
            local escaped=$(echo "$cleaned_lyrics" | sed "s/'/''/g")
            sqlite3 "$DB" "UPDATE tracks SET lyrics='$escaped' WHERE id=$id;"
            ((cleaned++)) || true
        fi
    done

    echo -e "\n${GREEN}Cleaned $cleaned track(s)${NC}"
}

# Full fix
cmd_fix() {
    cmd_dedupe
    echo ""
    cmd_orphans
    echo ""
    cmd_fix_missing
    echo ""
    cmd_rename
}

# Usage
usage() {
    echo "Drift FM Audio Library Hygiene"
    echo ""
    echo "Usage: $0 <command>"
    echo ""
    echo "Commands:"
    echo "  check     Report all issues (no changes)"
    echo "  fix       Fix all issues (dedupe, orphans, rename)"
    echo "  rename    Rename files to URL-friendly convention"
    echo "  dedupe    Remove content duplicates"
    echo "  orphans   Remove orphaned files"
}

case "${1:-}" in
    check)
        cmd_check
        ;;
    fix)
        cmd_fix
        ;;
    rename)
        cmd_rename
        ;;
    dedupe)
        cmd_dedupe
        ;;
    orphans)
        cmd_orphans
        ;;
    fix-missing)
        cmd_fix_missing
        ;;
    clean-lyrics)
        cmd_clean_lyrics
        ;;
    -h|--help|"")
        usage
        ;;
    *)
        echo "Unknown command: $1"
        usage
        exit 1
        ;;
esac
