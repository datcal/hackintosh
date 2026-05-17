# Hackintosh

A USB-tethered OLED companion device — Tinytosh-style retro-Mac chrome with live
animations — for showing clock, weather, air quality, currency rates, system
metrics, and now-playing media from your host PC.

## Hardware

- Seeed XIAO RP2040
- 0.96" SSD1306 128x64 I2C OLED at `0x3C`
- 2 momentary push buttons

For full wiring details, pin map, circuit diagram, and 4-pin button anatomy,
see **[docs/HARDWARE.md](docs/HARDWARE.md)**.

## Layout

```
firmware/   Arduino C++ for the XIAO RP2040 — receives 1024-byte framebuffers
            over USB-CDC and blits them; sends button events back.
host/       Go application — fetches data, renders 1-bit 128x64 framebuffers at
            30 FPS, streams them over the serial port.
scripts/    Build scripts (Windows .ps1, macOS .sh) for host + firmware + dev.
dist/       Build outputs (gitignored).
```

## Quick start

**Build the host:**
- Windows: `.\scripts\build-host.ps1`
- macOS:   `./scripts/build-host.sh`

**Build + flash the firmware in one shot** (recommended once your board is alive):
- Windows: `.\scripts\flash-firmware.ps1`
- macOS:   `./scripts/flash-firmware.sh`

This compiles + uploads via the 1200-baud auto-reset trick — no button presses on the board.

**First flash** (or recovery when firmware is unresponsive):
- Build only: `.\scripts\build-firmware.ps1` / `./scripts/build-firmware.sh`
- Then **double-tap the XIAO's RESET** and drag `dist/hackintosh.uf2` onto the `RPI-RP2` drive.

For one-time `arduino-cli` setup and troubleshooting, see **[docs/FLASHING.md](docs/FLASHING.md)**.

**Run without hardware (browser simulator):**
- Windows: `.\scripts\dev-host.ps1 --simulate=:8080`
- macOS:   `./scripts/dev-host.sh --simulate=:8080`

Open <http://localhost:8080> in a browser. You'll see a styled XIAO+OLED device
with the full UI streaming at 30 FPS via Server-Sent Events. The two on-screen
buttons (and keys `A`/`1`, `B`/`2`) feed presses back to the app exactly like
the real hardware. Long-press = hold for 700 ms.

The same flag works on the production binary:
```
dist\hackintosh.exe --simulate=:8080
dist/hackintosh     --simulate=:8080
```

**Probe a connected MCU:** `.\scripts\dev-host.ps1 --probe`

**Flags:** `--no-net` skips weather + currency · `--no-hw` skips system monitor
· `--no-media` skips the now-playing source · `--port=COM5` picks a specific
serial port.

## Install as a startup app

To have the host run automatically when you log in, with a menu-bar / tray
icon instead of a terminal window:

1. Build the host:
   - Windows: `./scripts/build-host.ps1`
   - macOS:   `./scripts/build-host.sh`
   - Linux:   `cd host && go build -o ../dist/hackintosh ./cmd/hackintosh`
2. Run the installer:

   ```
   ./dist/hackintosh install
   ```

   This copies the binary to a per-user location and registers it for
   autostart. No admin/sudo required.
3. The next time you log in, an icon appears in your system tray (Windows)
   or menu bar (macOS) or app tray (Linux). Right-click for: Open Simulator
   (only enabled in simulator mode), Restart, Quit.
4. Re-launch from your Start Menu / Applications folder / app menu after
   quitting, just like any other installed app.

To remove everything the installer added:

```bash
./dist/hackintosh uninstall
```

After making code changes, rebuild and run `./dist/hackintosh install`
again. It is idempotent and overwrites the previous binary.

### Linux note: system trays in GNOME

GNOME removed built-in tray support, so on stock GNOME the icon will not
appear without an extension like AppIndicator or TopIconsPlus. The host
still runs and connects to the device; only the menu is hidden.

### macOS note: Gatekeeper

The first run on macOS Catalina+ may trigger a Gatekeeper warning
("Hackintosh cannot be opened"). Right-click the app in Finder, choose Open,
or run `xattr -d com.apple.quarantine ~/Applications/Hackintosh.app` once.

## Controls

- **Button A** — Tea timer: tap to start a 3-minute countdown, tap again to pause/resume, long-press to reset. "TEA!" appears on the OLED when it expires.
- **Button B** — Cycle to the next screen (Clock → Weather → AQ → Currency → System → Media)
- **A + B together for 3 seconds** — Reboot into bootloader for over-the-USB firmware updates. The OLED briefly shows "BOOTLOAD ready to flash". Pair with `.\scripts\dev-host.ps1 --flash dist\hackintosh.uf2` for fully scripted updates.

## Documentation

- [docs/HARDWARE.md](docs/HARDWARE.md) — wiring, pin map, circuit diagram, button anatomy
- [docs/FLASHING.md](docs/FLASHING.md) — toolchain setup, firmware build, UF2 upload, troubleshooting
