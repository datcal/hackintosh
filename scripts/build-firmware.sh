#!/usr/bin/env bash
# Build the XIAO RP2040 firmware via arduino-cli.
# Outputs a .uf2 file to dist/hackintosh.uf2.
#
# One-time setup (both OSes):
#   arduino-cli core update-index --additional-urls https://github.com/earlephilhower/arduino-pico/releases/download/global/package_rp2040_index.json
#   arduino-cli core install rp2040:rp2040 --additional-urls https://github.com/earlephilhower/arduino-pico/releases/download/global/package_rp2040_index.json
#   arduino-cli lib install "Adafruit GFX Library" "Adafruit SSD1306"
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
sketch="$root/firmware/hackintosh"
out="$root/dist"
mkdir -p "$out"

arduino-cli compile \
    --fqbn rp2040:rp2040:seeed_xiao_rp2040 \
    --output-dir "$out/firmware-build" \
    "$sketch"

cp "$out/firmware-build/hackintosh.ino.uf2" "$out/hackintosh.uf2"
echo
echo "Built $out/hackintosh.uf2"
echo "Flash:  double-tap the XIAO's RESET button, then drag hackintosh.uf2 onto the RPI-RP2 drive."
