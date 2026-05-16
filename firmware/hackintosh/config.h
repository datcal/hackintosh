#pragma once

// ===== Pin map (Seeed XIAO RP2040) =====
//
// The XIAO has two parallel pin namings:
//   - "Dx" labels printed on the silkscreen (what you see when wiring)
//   - "GPxx" raw RP2040 GPIO numbers (what the chip actually addresses)
//
// We use the D0..D10 macros from the arduino-pico Seeed variant so the
// source matches the board. The compiler resolves these to GPIO numbers:
//
//   D0 -> GP26   D1 -> GP27   D4 -> GP6   D5 -> GP7
//
// Wire your hardware to the pins labeled D0/D1/D4/D5 on the board.
constexpr uint8_t PIN_SDA      = D4;  // OLED SDA  -> silkscreen "D4" (GP6)
constexpr uint8_t PIN_SCL      = D5;  // OLED SCL  -> silkscreen "D5" (GP7)
constexpr uint8_t PIN_BTN_A    = D0;  // Button A  -> silkscreen "D0" (GP26)
constexpr uint8_t PIN_BTN_B    = D1;  // Button B  -> silkscreen "D1" (GP27)

// ===== OLED =====
constexpr uint8_t  OLED_W       = 128;
constexpr uint8_t  OLED_H       = 64;
constexpr uint8_t  OLED_ADDR    = 0x3C;
constexpr uint16_t FRAME_BYTES  = (uint16_t)(OLED_W * OLED_H / 8);  // 1024

// Set true if the OLED is mounted upside-down. Toggles SSD1306's SEGREMAP +
// COM-scan-direction registers so the chip rotates everything 180 degrees in
// hardware. No software framebuffer flipping needed — the host can keep
// drawing pixel (0,0) at the "logical" top-left.
constexpr bool OLED_FLIP_180    = true;

// ===== Serial transport =====
constexpr uint32_t SERIAL_BAUD          = 115200;  // ignored by USB-CDC, but harmless
constexpr uint8_t  PROTO_MAGIC_0        = 0xAA;
constexpr uint8_t  PROTO_MAGIC_1        = 0x55;
constexpr uint32_t DISCONNECT_TIMEOUT_MS = 3000;

// Inbound frame types (host -> MCU)
constexpr uint8_t TYPE_FRAME      = 0x01;
constexpr uint8_t TYPE_BRIGHTNESS = 0x02;
constexpr uint8_t TYPE_PING       = 0x03;

// Outbound frame types (MCU -> host)
constexpr uint8_t TYPE_BUTTON     = 0x81;
constexpr uint8_t TYPE_PONG       = 0x82;
constexpr uint8_t TYPE_LOG        = 0x83;

// Button events
constexpr uint8_t BTN_ID_A        = 0;
constexpr uint8_t BTN_ID_B        = 1;
constexpr uint8_t BTN_EVT_PRESS   = 0;
constexpr uint8_t BTN_EVT_RELEASE = 1;
constexpr uint8_t BTN_EVT_LONG    = 2;

constexpr uint16_t BTN_DEBOUNCE_MS  = 15;
constexpr uint16_t BTN_LONGPRESS_MS = 700;
