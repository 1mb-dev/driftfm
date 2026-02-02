#!/bin/bash
#
# Normalize audio file for Drift FM
# Usage: ./scripts/normalize.sh <input_file> [output_dir]
#
# Uses EBU R128 loudness normalization
# Target: -16 LUFS (streaming standard)

set -e

INPUT_FILE="$1"
OUTPUT_DIR="${2:-audio/focus}"

if [ -z "$INPUT_FILE" ]; then
    echo "Usage: $0 <input_file> [output_dir]"
    echo ""
    echo "Normalizes audio to -16 LUFS for consistent playback."
    exit 1
fi

if [ ! -f "$INPUT_FILE" ]; then
    echo "Error: File not found: $INPUT_FILE"
    exit 1
fi

# Check ffmpeg
if ! command -v ffmpeg &> /dev/null; then
    echo "Error: ffmpeg is required. Install with: brew install ffmpeg"
    exit 1
fi

mkdir -p "$OUTPUT_DIR"

# Generate output filename
BASENAME=$(basename "$INPUT_FILE" | sed 's/\.[^.]*$//')
TIMESTAMP=$(date +%s)
OUTPUT_FILE="$OUTPUT_DIR/${BASENAME}_${TIMESTAMP}.mp3"

echo "Processing: $INPUT_FILE"
echo "Output: $OUTPUT_FILE"

# Two-pass loudness normalization
echo "Pass 1: Analyzing loudness..."
LOUDNESS_JSON=$(ffmpeg -hide_banner -i "$INPUT_FILE" \
    -af loudnorm=I=-16:TP=-1.5:LRA=11:print_format=json \
    -f null - 2>&1 | grep -A 20 '"input_i"' | head -15)

# Extract measured values
INPUT_I=$(echo "$LOUDNESS_JSON" | grep '"input_i"' | sed 's/.*: "\([^"]*\)".*/\1/')
INPUT_TP=$(echo "$LOUDNESS_JSON" | grep '"input_tp"' | sed 's/.*: "\([^"]*\)".*/\1/')
INPUT_LRA=$(echo "$LOUDNESS_JSON" | grep '"input_lra"' | sed 's/.*: "\([^"]*\)".*/\1/')
INPUT_THRESH=$(echo "$LOUDNESS_JSON" | grep '"input_thresh"' | sed 's/.*: "\([^"]*\)".*/\1/')

if [ -z "$INPUT_I" ]; then
    echo "Warning: Could not analyze loudness, using single-pass normalization"
    ffmpeg -hide_banner -y -i "$INPUT_FILE" \
        -af "loudnorm=I=-16:TP=-1.5:LRA=11" \
        -codec:a libmp3lame -b:a 192k \
        "$OUTPUT_FILE"
else
    echo "Measured loudness: ${INPUT_I} LUFS"
    echo "Pass 2: Normalizing..."
    ffmpeg -hide_banner -y -i "$INPUT_FILE" \
        -af "loudnorm=I=-16:TP=-1.5:LRA=11:measured_I=${INPUT_I}:measured_TP=${INPUT_TP}:measured_LRA=${INPUT_LRA}:measured_thresh=${INPUT_THRESH}:linear=true" \
        -codec:a libmp3lame -b:a 192k \
        "$OUTPUT_FILE"
fi

# Get duration
DURATION=$(ffprobe -v quiet -show_entries format=duration -of csv=p=0 "$OUTPUT_FILE" | cut -d'.' -f1)

echo ""
echo "Done: $OUTPUT_FILE"
echo "Duration: ${DURATION}s"
echo ""
echo "To import: make import FILE=$OUTPUT_FILE MOOD=focus"
