# Hardware

Wiring, pin map, and physical assembly for Hackintosh.

## Parts list

| Part | Notes |
|---|---|
| Seeed XIAO RP2040 | The microcontroller. RP2040 chip, USB-C, no WiFi/BT. |
| 0.96" SSD1306 OLED, 128x64, I2C | Adafruit / generic. I2C address `0x3C`. |
| 2 momentary push buttons | 4-pin tactile style works (only 2 of the 4 pins are needed). |
| USB-C cable | Must be a *data* cable — many charge-only cables won't enumerate. |
| Optional: 2 x ~10kΩ resistors | NOT needed when using `INPUT_PULLUP` in firmware. |

## Pin assignments

The XIAO RP2040 has **silkscreen labels** (`D0`–`D10`) which are different from
the internal **GPIO numbers** (`GP0`–`GP29`). When wiring, use the silkscreen
labels — that's what's printed on the board next to each pin.

| Function | Silkscreen | Internal GPIO | Notes |
|---|---|---|---|
| OLED SDA | **D4** | GP6 | I2C data |
| OLED SCL | **D5** | GP7 | I2C clock |
| Button A | **D0** | GP26 | INPUT_PULLUP — buttons short to GND |
| Button B | **D1** | GP27 | INPUT_PULLUP — buttons short to GND |
| OLED VCC | **3V3** | — | 3.3 V supply |
| OLED GND | **GND** | — | shared with button GNDs |

## XIAO RP2040 pinout reference

Looking at the board with the **USB-C connector facing up**:

```
            +-------------+
            |    USB-C    |
            +--+-------+--+
        D0 -- *|       |* -- 5V
        D1 -- *|       |* -- GND
        D2 -- *|       |* -- 3V3
        D3 -- *|  XIAO |* -- D10
        D4 -- *| RP2040|* -- D9
        D5 -- *|       |* -- D8
              |       |* -- D7
              +-------+* -- D6
```

## Circuit diagram

```
                                 SSD1306 OLED 128x64
                                 +----+----+----+----+
                                 |GND |VCC |SCL |SDA |
                                 +-+--+-+--+-+--+-+--+
                                   |    |    |    |
                                   |    |    |    |
                       +-----------+    |    |    +---- to D4 (SDA)
                       |        +-------+    +---- to D5 (SCL)
                       |        |    +---- to 3V3
                       |        |    |
                       |        |    |
                      GND      3V3   |  (already routed via the OLED rail)
                       (shared GND rail with both buttons)


                       +----------------------+
                       |  XIAO RP2040         |
                       |  (USB-C on top)      |
                       |                      |
                  D0 --|                      |-- 5V
                  D1 --|                      |-- GND (shared GND rail)
                  D2 --|                      |-- 3V3 (powers OLED)
                  D3 --|                      |-- D10
                  D4 --| (OLED SDA)           |-- D9
                  D5 --| (OLED SCL)           |-- D8
                       |                      |
                       +----------------------+
                        ^                  ^
                        |                  |
                       D0                 D1
                        |                  |
            +----- BUTTON A -----+   +----- BUTTON B -----+
            |                    |   |                    |
            |  PIN 1 *    * PIN 2|   |  PIN 1 *    * PIN 2|
            |          X         |   |          X         |
            |  PIN 3 *    * PIN 4|   |  PIN 3 *    * PIN 4|
            +----+-----------+---+   +----+-----------+---+
                 |           |            |           |
              (to D0)     (to GND)     (to D1)     (to GND)
                 |           |            |           |
                 +-----------+------------+-----------+
                             |
                             v
                            GND (any GND pin on the XIAO)
```

## 4-pin tactile button anatomy

The 4 pins of a typical tactile button only correspond to **2 electrical
contacts**. Pins on the same side of the body are internally bridged with a
metal bar at all times — even when the button isn't pressed. Pressing the
cap closes a contact between the two sides.

```
          +---------------------+
          |                     |
   PIN 1 *|@           @        |* PIN 2
          |                     |
          |        [cap]        |
          |                     |
   PIN 3 *|@           @        |* PIN 4
          |                     |
          +---------------------+

  Always connected:
    PIN 1 --- PIN 3       (one internal side bar)
    PIN 2 --- PIN 4       (the other internal side bar)

  When pressed, the gap closes — now all four are connected.
```

**Rule of thumb:** pick any pin on the LEFT side and any pin on the RIGHT
side. Those are your "switch terminals." Diagonal (PIN 1 + PIN 4) is the
safest choice because they're guaranteed to be on opposite sides.

If your multimeter shows all 4 pins connected when pressed, your button
is healthy — that's expected behavior.

## How the active-LOW input works

The firmware configures Button A and Button B as `INPUT_PULLUP`. This
activates the RP2040's built-in ~50kΩ pull-up resistor on each GPIO, so:

```
Button NOT pressed:                Button PRESSED:

  3V3 --[ ~50kΩ pull-up ]--+         3V3 --[ ~50kΩ pull-up ]--+
                           |                                  |
                          GPIO  (reads HIGH ~ 3.3V)           GPIO  (reads LOW ~ 0V)
                           |                                  |
                          (open switch)                      (closed switch)
                           |                                  |
                          GND                                GND
```

In firmware (see `buttons.cpp`), `digitalRead(pin) == LOW` means "pressed."

## Why you don't need the external resistors

If your buttons came bundled with separate resistors, they were intended
for the **external pull-up** wiring pattern: GPIO -> 10kΩ -> 3V3, plus GPIO ->
button -> GND. With `INPUT_PULLUP` in firmware, the resistor is internal —
the external one becomes redundant.

You can wire the externals too if you want — it doesn't hurt, just makes the
input slightly stiffer against electrical noise. To use the externals
instead of the internal pull-up, change `INPUT_PULLUP` to `INPUT` in
`firmware/hackintosh/buttons.cpp:begin()`.

## Display orientation (mounting it upside-down)

If you physically mount the OLED upside-down, set this in
`firmware/hackintosh/config.h`:

```cpp
constexpr bool OLED_FLIP_180 = true;
```

This sends two override commands to the SSD1306 during init (`SEGREMAP` +
`COMSCANINC`) that rotate the display 180° in hardware. The host doesn't
need to know — every screen, splash, and animation comes out the right way
up. Set it back to `false` if you flip the mounting.

## Power budget

| Component | Typical | Max |
|---|---|---|
| RP2040 idle | ~15 mA | ~50 mA |
| SSD1306 OLED, all pixels off | ~5 mA | — |
| SSD1306 OLED, all pixels on | — | ~25 mA |
| Total at typical usage | ~25 mA | ~75 mA |

All well under USB's 500 mA budget. No need for an external supply.
