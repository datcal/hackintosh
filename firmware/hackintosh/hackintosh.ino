// hackintosh firmware
//
// Receives 128x64 1-bit framebuffers over USB-CDC at ~30 FPS and blits them
// to the SSD1306. Sends button events back to the host. Shows a self-drawn
// disconnect splash if the host stops sending frames for >3 s.
//
// When disconnected, button A drives a native 3-minute tea timer that
// renders directly to the OLED without needing the host.

#include <Arduino.h>
#include "config.h"
#include "display.h"
#include "transport.h"
#include "buttons.h"
#include "timer.h"

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

    const uint32_t now = millis();

    // Consume button A events every loop so they don't queue up.
    // When disconnected, route them to the firmware timer.
    // When connected, the host handles the tea timer via framebuffer.
    const bool aPressEvent = buttons::consumeAPress();
    const bool aLongEvent  = buttons::consumeALong();
    if (!transport::isConnected()) {
        if (aPressEvent) timer::tap();
        if (aLongEvent)  timer::reset();
    }

    timer::tick(now);

    if (transport::consumeNewFrame()) {
        display::blitFrame(transport::lastFrame());
        return;
    }

    // No fresh frame this iteration. If we're still inside the connected
    // window, just leave the last frame on the OLED. If we've lost the host,
    // update the display at ~5 fps so the screen never feels dead.
    if (!transport::isConnected()) {
        static uint32_t lastDrawMs = 0;
        if (now - lastDrawMs >= 200) {
            lastDrawMs = now;
            if (timer::getState() != timer::IDLE) {
                display::drawTimerDisplay(
                    timer::getState(),
                    timer::remainingMs(now),
                    timer::progress(now),
                    now
                );
            } else {
                display::drawDisconnectSplash(now);
            }
        }
    }
}
