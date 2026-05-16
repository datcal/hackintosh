# Compile + auto-upload the firmware via the 1200-baud reset trick.
#
# Uploads while the board is running normally - no need to double-tap RESET
# or unplug/replug. The arduino-pico core's USB-CDC firmware watches for
# `arduino-cli` opening the port at exactly 1200 bps as a magic handshake
# and reboots into BOOTSEL mode, then the CLI copies the new .uf2 over.
#
# Usage:
#   .\scripts\flash-firmware.ps1                   # auto-detect port
#   .\scripts\flash-firmware.ps1 -Port COM5        # explicit port
#
# Fallback: if the board isn't running responsive firmware (e.g. you bricked
# it with a bad upload), this script will fail. Use scripts\build-firmware.ps1
# instead and flash manually with the double-tap RESET method (see docs/FLASHING.md).
param([string]$Port = "")

$ErrorActionPreference = "Stop"

$root   = Resolve-Path "$PSScriptRoot\.."
$sketch = Join-Path $root "firmware\hackintosh"
$fqbn   = "rp2040:rp2040:seeed_xiao_rp2040"

# --- Auto-detect port via arduino-cli board list ---
if ($Port -eq "") {
    Write-Host "Scanning for connected RP2040 board..."
    $json = arduino-cli board list --format json
    $boards = $json | ConvertFrom-Json
    # The JSON shape varies by arduino-cli version; flatten safely:
    $ports = @()
    if ($boards.detected_ports) {
        $ports = $boards.detected_ports
    } else {
        $ports = $boards
    }
    foreach ($p in $ports) {
        $addr = if ($p.port) { $p.port.address } else { $p.address }
        $matched = $false
        if ($p.matching_boards) {
            foreach ($mb in $p.matching_boards) {
                if ($mb.fqbn -eq $fqbn) { $matched = $true; break }
            }
        }
        if ($matched) {
            $Port = $addr
            break
        }
    }
}

if ($Port -eq "") {
    Write-Host ""
    Write-Host "Could not auto-detect the XIAO RP2040." -ForegroundColor Yellow
    Write-Host "Try:  arduino-cli board list"
    Write-Host "Then re-run with the right port:  .\scripts\flash-firmware.ps1 -Port COM5"
    Write-Host ""
    Write-Host "Or fall back to manual flashing (double-tap RESET + drag .uf2):"
    Write-Host "  .\scripts\build-firmware.ps1"
    Write-Host "  Then drag dist\hackintosh.uf2 to the RPI-RP2 drive."
    exit 1
}

Write-Host "Flashing via $Port (will reboot the board automatically)..."
Write-Host ""

arduino-cli compile `
    --fqbn $fqbn `
    --upload `
    --port $Port `
    $sketch

if ($LASTEXITCODE -ne 0) {
    Write-Host ""
    Write-Host "Upload failed." -ForegroundColor Red
    Write-Host "If the board is unresponsive, use the manual method:"
    Write-Host "  1. Double-tap RESET on the XIAO"
    Write-Host "  2. Drag dist\hackintosh.uf2 to the RPI-RP2 drive"
    exit $LASTEXITCODE
}

Write-Host ""
Write-Host "Done - firmware running. Test with:  .\scripts\dev-host.ps1"
