# Build the XIAO RP2040 firmware via arduino-cli.
# Outputs dist\hackintosh.uf2.
#
# One-time setup:
#   arduino-cli core update-index --additional-urls https://github.com/earlephilhower/arduino-pico/releases/download/global/package_rp2040_index.json
#   arduino-cli core install rp2040:rp2040 --additional-urls https://github.com/earlephilhower/arduino-pico/releases/download/global/package_rp2040_index.json
#   arduino-cli lib install "Adafruit GFX Library" "Adafruit SSD1306"
$ErrorActionPreference = "Stop"

$root   = Resolve-Path "$PSScriptRoot\.."
$sketch = Join-Path $root "firmware\hackintosh"
$out    = Join-Path $root "dist"
New-Item -ItemType Directory -Force -Path $out | Out-Null

$build = Join-Path $out "firmware-build"
arduino-cli compile `
    --fqbn rp2040:rp2040:seeed_xiao_rp2040 `
    --output-dir $build `
    $sketch

Copy-Item -Force (Join-Path $build "hackintosh.ino.uf2") (Join-Path $out "hackintosh.uf2")
Write-Host ""
Write-Host "Built $(Join-Path $out 'hackintosh.uf2')"
Write-Host "Flash:  double-tap RESET on the XIAO, then drag hackintosh.uf2 onto the RPI-RP2 drive."
