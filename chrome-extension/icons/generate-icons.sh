#!/bin/bash
# Generates icon PNGs from the SVG source.
# Requires: rsvg-convert (brew install librsvg) or inkscape

SVG="icon.svg"

for size in 16 32 48 128; do
  if command -v rsvg-convert &>/dev/null; then
    rsvg-convert -w $size -h $size "$SVG" -o "icon${size}.png"
  elif command -v inkscape &>/dev/null; then
    inkscape --export-type=png --export-width=$size --export-filename="icon${size}.png" "$SVG"
  else
    echo "Install librsvg: brew install librsvg"
    exit 1
  fi
  echo "Generated icon${size}.png"
done
