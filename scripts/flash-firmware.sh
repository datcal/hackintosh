#!/usr/bin/env bash
# Compile + auto-upload the firmware via the 1200-baud reset trick.
#
# Uploads while the board is running normally — no need to double-tap RESET
# or unplug/replug. See the PowerShell version's comment for the mechanism.
#
# Usage:
#   ./scripts/flash-firmware.sh                       # auto-detect port
#   ./scripts/flash-firmware.sh /dev/cu.usbmodem1101  # explicit port
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
sketch="$root/firmware/hackintosh"
fqbn="rp2040:rp2040:seeed_xiao_rp2040"

port="${1:-}"

if [[ -z "$port" ]]; then
    echo "Scanning for connected RP2040 board..."
    # arduino-cli emits JSON with the matched FQBN; jq isn't always installed
    # on macOS by default, so we use a Python one-liner instead.
    port=$(arduino-cli board list --format json | python3 -c "
import json, sys
data = json.load(sys.stdin)
ports = data.get('detected_ports', data) if isinstance(data, dict) else data
for p in ports:
    addr = p.get('port', {}).get('address') or p.get('address')
    for mb in (p.get('matching_boards') or []):
        if mb.get('fqbn') == '$fqbn':
            print(addr)
            sys.exit(0)
" 2>/dev/null || true)
fi

if [[ -z "$port" ]]; then
    cat <<'EOF'

Could not auto-detect the XIAO RP2040. Try:
    arduino-cli board list
Then re-run with the right port:
    ./scripts/flash-firmware.sh /dev/cu.usbmodemXXXX

Or fall back to manual flashing (double-tap RESET + drag .uf2):
    ./scripts/build-firmware.sh
    # Then drag dist/hackintosh.uf2 onto the RPI-RP2 drive.
EOF
    exit 1
fi

echo "Flashing via $port (will reboot the board automatically)..."
echo

arduino-cli compile \
    --fqbn "$fqbn" \
    --upload \
    --port "$port" \
    "$sketch"

echo
echo "Done — firmware running. Test with:  ./scripts/dev-host.sh"
