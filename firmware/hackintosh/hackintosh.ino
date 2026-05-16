// hackintosh firmware
//
// Receives 128x64 1-bit framebuffers over USB-CDC at ~30 FPS and blits them
// to the SSD1306. Sends button events back to the host. Shows a self-drawn
// disconnect splash if the host stops sending frames for >3 s.

#include <Arduino.h>
#include "config.h"
#include "display.h"
#include "transport.h"
#include "buttons.h"

static bool displayReady = false;

void setup() {
    Serial.begin(SERIAL_BAUD);
    displayReady = display::begin();
    if (displayReady) display::drawBootSplash();
    buttons::begin();
}

void loop() {
    transport::poll();
    buttons::poll();

    if (!displayReady) return;

    if (transport::consumeNewFrame()) {
        display::blitFrame(transport::lastFrame());
        return;
    }

    // No fresh frame this iteration. If we're still inside the connected
    // window, just leave the last frame on the OLED. If we've lost the host,
    // animate the disconnect splash at ~5 fps so the screen never feels dead.
    if (!transport::isConnected()) {
        static uint32_t lastSplashMs = 0;
        const uint32_t now = millis();
        if (now - lastSplashMs >= 200) {
            lastSplashMs = now;
            display::drawDisconnectSplash(now);
        }
    }
}
