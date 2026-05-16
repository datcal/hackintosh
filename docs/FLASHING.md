# Flashing the firmware

How to compile and upload the Arduino firmware to your XIAO RP2040.

The RP2040 uses **UF2 (USB Flashing Format)**, which means you don't need
`avrdude`, `esptool`, or any traditional uploader — you just **copy a file
onto a USB drive that appears when you put the board into bootloader mode**.

Three ways to do it, in order of how nice they are once your firmware is
working:

1. **Hold A+B on the device for 3 seconds, then auto-copy from Go** —
   the cleanest path. Your application buttons trigger the reboot, the Go
   tool watches for the bootloader drive and copies the new firmware over.
   Zero physical RESET/BOOT button presses, fully scripted, works on
   Windows + macOS + Linux. See "Option 0" below.
2. **`arduino-cli` 1200-baud auto-upload** — works if your firmware is
   running and responsive. Compile + upload in one script.
3. **Manual double-tap RESET** — required for the very first flash (before
   any firmware exists) or if your current firmware is unresponsive.

## One-time toolchain setup

The Go host doesn't need anything beyond Go itself, but the firmware needs
the Arduino CLI plus the RP2040 board package and two display libraries.

```powershell
# 1. Install arduino-cli
winget install ArduinoSA.CLI
# (on macOS:  brew install arduino-cli)

# 2. Add the community RP2040 board package (Earle Philhower's core)
arduino-cli core update-index --additional-urls https://github.com/earlephilhower/arduino-pico/releases/download/global/package_rp2040_index.json
arduino-cli core install rp2040:rp2040 --additional-urls https://github.com/earlephilhower/arduino-pico/releases/download/global/package_rp2040_index.json

# 3. Install the OLED libraries
arduino-cli lib install "Adafruit GFX Library" "Adafruit SSD1306"
```

To verify the install succeeded:

```powershell
arduino-cli core list
# Should list:  rp2040:rp2040  <version>  arduino-pico

arduino-cli board listall xiao
# Should list:  Seeed XIAO RP2040  rp2040:rp2040:seeed_xiao_rp2040
```

The RP2040 board package is ~150 MB (it bundles a full GCC cross-compiler
for ARM Cortex-M0+). This is a one-time download.

## Build the .uf2

From the project root:

```powershell
.\scripts\build-firmware.ps1
```

On macOS / Linux:

```bash
./scripts/build-firmware.sh
```

This invokes `arduino-cli compile` with the right FQBN (`rp2040:rp2040:seeed_xiao_rp2040`),
compiles all the `.ino` and `.cpp` files under `firmware/hackintosh/`,
and writes `dist/hackintosh.uf2`.

Typical output size: 30–80 KB.

## Option 0 — A+B hold (3 sec) + Go auto-copy (recommended)

This is the smoothest workflow once the device is alive. The XIAO's tiny
on-board RESET and BOOT buttons are never touched.

### How it works

1. The firmware watches both application buttons (A on D0, B on D1). When
   it sees them held simultaneously for **3 seconds**, it briefly draws a
   "BOOTLOAD ready to flash" panel on the OLED, then calls
   `rp2040.rebootToBootloader()` — a soft reset into the mask-ROM bootloader.
2. The `RPI-RP2` mass-storage drive appears on your computer within ~500ms.
3. The Go host's `--flash` mode polls for that drive cross-platform
   (checking for the `INFO_UF2.TXT` marker file the bootloader always
   writes) and copies your new `dist\hackintosh.uf2` onto it.
4. The bootloader detects the .uf2 write, flashes it to internal flash, and
   reboots into the new firmware. Total elapsed: ~5-8 seconds.

### Step by step

**Terminal 1** — build the new firmware:

```powershell
.\scripts\build-firmware.ps1
```

**Terminal 2** — start the auto-flash watcher:

```powershell
.\scripts\dev-host.ps1 --flash dist\hackintosh.uf2
```

On macOS / Linux:

```bash
./scripts/dev-host.sh --flash dist/hackintosh.uf2
```

You'll see:

```text
flash: waiting for RPI-RP2 drive — hold A+B on the device for 3 seconds...
```

**On the device**: press and hold A and B together. After 3 seconds the
OLED briefly shows a bordered "BOOTLOAD ready to flash" panel — that's
your visual confirmation the combo registered. You can release the buttons
now; the board has already triggered the reboot.

The Go tool will then print:

```text
flash: found D:\, copying dist\hackintosh.uf2 (78420 bytes)...
flash: done — the board will reboot into the new firmware momentarily.
```

That's it. No physical RESET or BOOT button presses. By default the
watcher waits up to 60 seconds for the drive (configurable with
`--flash-timeout=N`).

### What if A+B doesn't work?

The A+B combo runs inside `buttons.cpp::poll()`, which is part of your
firmware's main loop. If your firmware is **crashing or unresponsive**,
that loop never runs and the combo never fires. In that case, fall back to
Option 1 (still software-driven, via arduino-cli's 1200-baud trick) or
Option 2 (manual double-tap RESET).

The hierarchy of escalating recovery:

| Firmware state | Use |
|---|---|
| Healthy + responsive | Option 0 (A+B for 3s + `--flash`) |
| Responds to serial but A+B handler broken | Option 1 (`.\scripts\flash-firmware.ps1`) |
| Crashing, no USB-CDC | Option 2 (double-tap RESET, drag .uf2 manually) |
| Bricked completely | Option 2 with the BOOT-button-hold variant |

## Option 1 — Automatic (compile + upload, no buttons)

Once you've successfully flashed the board at least once and the firmware
is running normally, you can use the one-shot script:

```powershell
.\scripts\flash-firmware.ps1
```

On macOS / Linux:

```bash
./scripts/flash-firmware.sh
```

This compiles the firmware AND uploads it via the **1200-baud reset trick**
(see "How it works" below). No button presses on the board are required.
The whole sequence takes ~5 seconds:

```text
Scanning for connected RP2040 board...
Flashing via COM5 (will reboot the board automatically)...

Sketch uses 88340 bytes (4%) of program storage space...
Performing 1200-bps touch reset on serial port COM5
Waiting for upload port...
Upload port found on COM5
Loading into Flash: [==============================] 100%
The device was reset

Done — firmware running. Test with: .\scripts\dev-host.ps1
```

If the script can't auto-detect your port, run `arduino-cli board list` to
see the right name and pass it explicitly:

```powershell
.\scripts\flash-firmware.ps1 -Port COM5
```

```bash
./scripts/flash-firmware.sh /dev/cu.usbmodem1101
```

### How the 1200-baud reset trick works

When `arduino-cli` opens the USB-CDC serial port at exactly **1200 baud**,
the arduino-pico core in your running firmware treats this as a magic
"reboot into bootloader" signal:

1. arduino-cli opens `COM5` at 1200 bps, briefly.
2. Your running firmware's USB-CDC handler sees the unusual baud rate, stores
   a magic value in SRAM, and triggers a watchdog reset.
3. The RP2040's mask ROM bootloader reboots, sees the magic value, and
   stays in BOOTSEL mode instead of launching the firmware.
4. The `RPI-RP2` drive appears.
5. arduino-cli copies the new .uf2 onto the drive.
6. The board reboots into the new firmware.

This is the same trick used by Arduino Leonardo / Pro Micro / Teensy and
many other boards — it lets you flash without ever touching the physical
RESET button. The catch: it only works if your *currently running* firmware
is responsive enough to receive the 1200-baud signal. If you've bricked
the board with crashing firmware, the host-side script will time out
waiting for the `RPI-RP2` drive to appear; fall back to Option 2 below.

## Option 2 — Manual upload (the first time, or when stuck)

Plug the XIAO into your computer with a **USB-C data cable**. Then choose
one of two methods:

### Method A — double-tap RESET (normal path)

1. With the XIAO plugged in, **double-tap the small RESET button** on the
   board (next to the USB connector).
2. A USB drive named `RPI-RP2` appears in your file explorer.
3. **Drag `dist\hackintosh.uf2` onto that drive.**
4. The drive disappears and the board reboots — your firmware is running.

### Method B — hold BOOT during plug-in (rescue path)

Use this when Method A doesn't work (e.g., if the previously-flashed
firmware crashes immediately and never gets to the point where a double-tap
can trigger the bootloader):

1. Unplug the XIAO.
2. **Hold the BOOT button** while plugging the USB cable back in. Keep it
   held for a second after it's plugged in.
3. The `RPI-RP2` drive appears.
4. Drag `dist\hackintosh.uf2` onto it.

## Verify the firmware is alive

```powershell
# List ports — your XIAO should show up as a COM... or /dev/cu.usbmodem... entry:
.\scripts\dev-host.ps1 --list-ports

# Test the USB-CDC link with PING/PONG and live button events:
.\scripts\dev-host.ps1 --probe
```

You should see something like:

```
auto-picked port: COM5
2026/05/16 17:09:35 probe: waiting for events. Press Ctrl-C to exit.
2026/05/16 17:09:35 PONG  (RTT 0s)
2026/05/16 17:09:37 PONG  (RTT 0s)
2026/05/16 17:09:42 BUTTON A press
2026/05/16 17:09:42 BUTTON A release
```

"RTT 0s" is normal — USB-CDC round-trip is in the hundreds of nanoseconds,
which my display rounds down to 0. The OLED will show the animated radar
"searching for host..." splash because `--probe` doesn't send any
framebuffers — that's the firmware's correct fallback behavior.

To see the actual UI:

```powershell
.\scripts\dev-host.ps1
```

(No flags. This runs the full app with all data sources active.)

## Troubleshooting

| Symptom | Fix |
|---|---|
| `arduino-cli: command not found` | Toolchain setup didn't finish — re-run the winget/brew install and restart your shell. |
| `Adafruit_SSD1306.h: No such file` | `arduino-cli lib install "Adafruit SSD1306"`. |
| Double-tap RESET does nothing | Use Method B — the existing firmware may be unresponsive. |
| `RPI-RP2` drive doesn't appear | Try a different USB cable. Many USB-C cables are charge-only. |
| OLED stays black after flash | Check wiring: SDA on D4, SCL on D5, OLED VCC on 3V3 (not 5V). Confirm the address — some breakouts ship at `0x3D` instead of `0x3C`; change `OLED_ADDR` in `firmware/hackintosh/config.h`. |
| `--probe` shows no PONG | The host might have grabbed the wrong serial port. Try `--list-ports`, identify the XIAO's actual port name, and use `--probe --port=COM6` (or whatever). |
| Display is upside-down | Set `OLED_FLIP_180 = true` in `firmware/hackintosh/config.h` and re-flash. |
| Garbled pixels or partial screens | If you've changed the protocol, host and firmware versions may be mismatched — rebuild both and re-flash. |
| Compile error about a missing function or symbol | Make sure you rebuilt `dist/hackintosh.uf2` after editing the firmware. The build script doesn't auto-watch files. |

## How UF2 works (educational background)

The RP2040 has a tiny **mask-ROM bootloader** built into the silicon. It
runs the moment the chip powers on and decides whether to launch your
firmware or expose the chip as a USB mass-storage device.

The decision is made in <500 ms after power-on:

- If `RUN` (the BOOT pin) is held LOW during reset → enter bootloader mode
  (exposes `RPI-RP2` drive).
- Otherwise → launch firmware at flash address `0x10000000`.

The "double-tap RESET" trick works because the RP2040 also watches for a
**second reset within ~500 ms of the first one**, and re-enters the
bootloader if it sees one. This is purely a hardware feature — it works
even if your firmware crashes immediately, because the second RESET pulse
bypasses everything.

UF2 itself is a tiny file format invented by Microsoft for the BBC micro:bit.
Each .uf2 file is a series of 512-byte blocks; each block contains 256 bytes
of payload plus metadata saying *where in flash* it should land. When you
drag a .uf2 onto the `RPI-RP2` drive, the bootloader reads each block,
copies its payload to the indicated flash address, and reboots when done.
There's no protocol — you really are just copying a file.
