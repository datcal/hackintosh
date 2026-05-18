#include "buttons.h"
#include "transport.h"
#include "display.h"
#include "config.h"

namespace buttons {

// Holding BOTH A and B for this long triggers a reboot into BOOTSEL so
// you can drop a new .uf2 onto the RPI-RP2 drive without ever touching
// the tiny on-board RESET / BOOT buttons.
constexpr uint32_t COMBO_REBOOT_MS = 3000;

struct Btn {
    uint8_t  pin;
    uint8_t  id;
    bool     rawDown;          // current debounced state
    bool     pressedSent;      // whether we already sent the PRESS event for the current down-stroke
    bool     longSent;         // whether we already sent the LONG event
    uint32_t lastEdgeMs;       // last time the raw input changed (for debounce)
    uint32_t pressStartMs;     // time the current press began
    int      lastRawRead;      // last unfiltered digitalRead value
};

static Btn btns[2];

// 0 = not currently in the A+B combo; otherwise the millis() at which the
// combo started. Reset whenever either button is released.
static uint32_t comboSince    = 0;
static bool     pendingAPress = false;
static bool     pendingALong  = false;

static void onPress(Btn& b) {
    b.pressedSent  = true;
    b.longSent     = false;
    b.pressStartMs = millis();
    if (b.id == BTN_ID_A) pendingAPress = true;
    transport::sendButton(b.id, BTN_EVT_PRESS);
}

static void onRelease(Btn& b) {
    transport::sendButton(b.id, BTN_EVT_RELEASE);
    b.pressedSent = false;
    b.longSent    = false;
}

void begin() {
    btns[0] = { PIN_BTN_A, BTN_ID_A, false, false, false, 0, 0, HIGH };
    btns[1] = { PIN_BTN_B, BTN_ID_B, false, false, false, 0, 0, HIGH };
    pinMode(PIN_BTN_A, INPUT_PULLUP);
    pinMode(PIN_BTN_B, INPUT_PULLUP);
}

void poll() {
    const uint32_t now = millis();
    for (auto& b : btns) {
        const int raw = digitalRead(b.pin);

        if (raw != b.lastRawRead) {
            b.lastRawRead = raw;
            b.lastEdgeMs  = now;
        }

        if ((now - b.lastEdgeMs) >= BTN_DEBOUNCE_MS) {
            const bool nowDown = (raw == LOW);  // INPUT_PULLUP: LOW means pressed
            if (nowDown != b.rawDown) {
                b.rawDown = nowDown;
                if (nowDown) onPress(b);
                else         onRelease(b);
            }
        }

        if (b.rawDown && b.pressedSent && !b.longSent &&
            (now - b.pressStartMs) >= BTN_LONGPRESS_MS) {
            b.longSent = true;
            if (b.id == BTN_ID_A) pendingALong = true;
            transport::sendButton(b.id, BTN_EVT_LONG);
        }
    }

    // --- A+B combo: hold both for 3 seconds to reboot into BOOTSEL ---
    const bool both = btns[0].rawDown && btns[1].rawDown;
    if (both) {
        if (comboSince == 0) {
            comboSince = now;
        } else if ((now - comboSince) >= COMBO_REBOOT_MS) {
            // Confirm visually so the user knows the hold registered, then
            // reboot to bootloader. The "BOOTLOAD" message stays on the OLED
            // because the bootloader doesn't reinitialize I2C — it's visible
            // right up until the new firmware loads and takes over.
            transport::sendLog("entering bootloader (A+B held 3s)");
            display::drawBootloaderEnter();
            delay(600);  // brief visual confirmation + USB-CDC flush
            rp2040.rebootToBootloader();
        }
    } else {
        comboSince = 0;
    }
}

bool consumeAPress() { bool v = pendingAPress; pendingAPress = false; return v; }
bool consumeALong()  { bool v = pendingALong;  pendingALong  = false; return v; }

}
