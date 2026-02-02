#!/bin/bash
#
# Batch normalize audio files for Drift FM
# Usage: ./scripts/normalize-batch.sh <input_dir> [output_dir]

set -e

INPUT_DIR="$1"
OUTPUT_DIR="${2:-audio/focus}"

if [ -z "$INPUT_DIR" ]; then
    echo "Usage: $0 <input_dir> [output_dir]"
    exit 1
fi

if [ ! -d "$INPUT_DIR" ]; then
    echo "Error: Directory not found: $INPUT_DIR"
    exit 1
fi

echo "Batch normalizing from: $INPUT_DIR"
echo "Output directory: $OUTPUT_DIR"
echo ""

COUNT=0
FAILED=0

for FILE in "$INPUT_DIR"/*.{mp3,wav,flac,m4a,ogg} 2>/dev/null; do
    if [ -f "$FILE" ]; then
        echo "--- Processing: $(basename "$FILE") ---"
        if ./scripts/normalize.sh "$FILE" "$OUTPUT_DIR"; then
            COUNT=$((COUNT + 1))
        else
            FAILED=$((FAILED + 1))
            echo "Warning: Failed to process $FILE"
        fi
        echo ""
    fi
done

echo "================================"
echo "Processed: $COUNT files"
if [ $FAILED -gt 0 ]; then
    echo "Failed: $FAILED files"
fi
