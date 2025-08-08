#!/bin/bash
set -euo pipefail

FRAMES_DIR="frames"
GIF_OUT="orbit.gif"

# Create output directory
rm -rf "$FRAMES_DIR"
mkdir -p "$FRAMES_DIR"

# Render frames
for lon in $(seq 0 0.5 360); do
    lon10=$(printf "%.0f" "$(echo "$lon * 10" | bc)")
    printf -v lonpad "%04d" "$lon10"
    outfile="$FRAMES_DIR/frame_$lonpad.png"

    echo "Rendering longitude $lon → $outfile"

    go run main.go \
      -lat 0.0 \
      -lon "$lon" \
      -alt 12000.0 \
      -fov 60.0 \
      -tilt 0.0 \
      -size 768 \
      -supersample 2 \
      -day assets/world.200408.tif \
      -night assets/night.tif \
      -clouds assets/cloud.2001210.tif \
      -out "$outfile"
done

# Make GIF using ImageMagick
echo "Creating GIF..."
convert -delay 4 -loop 0 "$FRAMES_DIR"/frame_*.png "$GIF_OUT"
echo "Done → $GIF_OUT"
