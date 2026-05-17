#include "display.h"
#include "config.h"
#include "splash_data.h"

#include <Wire.h>
#include <Adafruit_GFX.h>
#include <Adafruit_SSD1306.h>

namespace display {

static Adafruit_SSD1306 oled(OLED_W, OLED_H, &Wire, -1);

bool begin() {
    Wire.setSDA(PIN_SDA);
    Wire.setSCL(PIN_SCL);
    Wire.begin();
    Wire.setClock(400000);

    if (!oled.begin(SSD1306_SWITCHCAPVCC, OLED_ADDR)) return false;

    // Hardware rotation: when the OLED is physically mounted upside-down,
    // override the two register bits that control horizontal + vertical scan
    // direction. Adafruit's default init sends 0xA1 (SEGREMAP|1) and 0xC8
    // (COMSCANDEC); we send the opposite pair to flip 180 degrees.
    if (OLED_FLIP_180) {
        oled.ssd1306_command(SSD1306_SEGREMAP | 0x0);  // 0xA0 — flip columns
        oled.ssd1306_command(SSD1306_COMSCANINC);      // 0xC0 — flip rows
    }

    oled.clearDisplay();
    oled.display();
    return true;
}

void blitFrame(const uint8_t* frame1024) {
    memcpy(oled.getBuffer(), frame1024, FRAME_BYTES);
    oled.display();
}

void setBrightness(uint8_t b) {
    oled.ssd1306_command(SSD1306_SETCONTRAST);
    oled.ssd1306_command(b);
}

void drawBootSplash() {
    oled.clearDisplay();
    oled.setTextSize(1);
    oled.setTextColor(SSD1306_WHITE);
    oled.setCursor(28, 20);
    oled.print("Hackintosh");
    oled.setCursor(20, 40);
    oled.print("waiting for host");
    oled.display();
}

void drawBootloaderEnter() {
    oled.clearDisplay();
    // Big "BOOTLOADER" centered, with a bordered frame so it's unmistakable.
    oled.drawRect(0, 0, OLED_W, OLED_H, SSD1306_WHITE);
    oled.drawRect(2, 2, OLED_W - 4, OLED_H - 4, SSD1306_WHITE);
    oled.setTextSize(2);
    oled.setTextColor(SSD1306_WHITE);
    oled.setCursor(8, 12);
    oled.print("BOOTLOAD");
    oled.setTextSize(1);
    oled.setCursor(18, 36);
    oled.print("ready to flash");
    oled.setCursor(30, 50);
    oled.print("plug + drop");
    oled.display();
}

void drawDisconnectSplash(uint32_t tickMs) {
    oled.clearDisplay();

    const uint8_t phase   = (tickMs / 5000) % 4;
    const uint8_t itemIdx = (tickMs / 20000) % min(NUM_POKEMON, NUM_GERMAN);

    if (phase == 0) {
        // Phase 0: original spinner + "searching..."
        const int cx = 28, cy = 32, r = 14;
        oled.drawCircle(cx, cy, r,     SSD1306_WHITE);
        oled.drawCircle(cx, cy, r / 2, SSD1306_WHITE);
        const float angle = (tickMs / 1000.0f) * 2.0f * 3.14159f;
        oled.drawLine(cx, cy,
                      cx + (int)(cos(angle) * r),
                      cy + (int)(sin(angle) * r),
                      SSD1306_WHITE);
        oled.setTextSize(1);
        oled.setTextColor(SSD1306_WHITE);
        oled.setCursor(54, 20);
        oled.print("Hackintosh");
        const uint8_t dotPhase = (tickMs / 500) % 4;
        oled.setCursor(54, 34);
        oled.print("searching");
        oled.setCursor(54, 44);
        for (uint8_t i = 0; i < dotPhase; i++) oled.print(".");

    } else if (phase == 1) {
        // Phase 1: today's Pokemon name (big) + types (small)
        const SplashPokemon& p = POKEMON[itemIdx % NUM_POKEMON];
        const uint8_t nlen = strlen(p.name);
        oled.setTextSize(2);
        oled.setTextColor(SSD1306_WHITE);
        oled.setCursor(max(0, (OLED_W - (int)nlen * 12) / 2), 8);
        oled.print(p.name);
        oled.setTextSize(1);
        oled.setCursor(max(0, (OLED_W - (int)strlen(p.type1) * 6) / 2), 36);
        oled.print(p.type1);
        if (p.type2[0] != '\0') {
            oled.setCursor(max(0, (OLED_W - (int)strlen(p.type2) * 6) / 2), 50);
            oled.print(p.type2);
        }

    } else if (phase == 2) {
        // Phase 2: "Wort des Tages" header + German word big and centered
        const SplashGerman& g = GERMAN[itemIdx % NUM_GERMAN];
        oled.setTextSize(1);
        oled.setTextColor(SSD1306_WHITE);
        oled.setCursor(20, 4);
        oled.print("Wort des Tages");
        oled.drawLine(0, 14, OLED_W - 1, 14, SSD1306_WHITE);
        oled.setTextSize(2);
        oled.setCursor(max(0, (OLED_W - (int)strlen(g.german) * 12) / 2), 22);
        oled.print(g.german);

    } else {
        // Phase 3: German word small at top + "EN:" + English meaning big
        const SplashGerman& g = GERMAN[itemIdx % NUM_GERMAN];
        oled.setTextSize(1);
        oled.setTextColor(SSD1306_WHITE);
        oled.setCursor(4, 4);
        oled.print(g.german);
        oled.drawLine(0, 14, OLED_W - 1, 14, SSD1306_WHITE);
        oled.setCursor(4, 20);
        oled.print("EN:");
        oled.setTextSize(2);
        oled.setCursor(max(0, (OLED_W - (int)strlen(g.english) * 12) / 2), 32);
        oled.print(g.english);
    }

    oled.display();
}

void drawTimerDisplay(timer::State state, uint32_t remainingMs, float progress, uint32_t tickMs) {
    oled.clearDisplay();

    if (state == timer::DONE) {
        // "TEA!" celebration with three animated dots.
        oled.setTextSize(3);
        oled.setTextColor(SSD1306_WHITE);
        oled.setCursor(22, 10);
        oled.print("TEA!");

        // Three dots bounce on a 600 ms cycle, offset by 200 ms each.
        for (uint8_t i = 0; i < 3; i++) {
            const uint32_t phase = (tickMs + i * 200) % 600;
            const int yOff = (phase < 300) ? -(int)(phase / 75) : -(int)((600 - phase) / 75);
            oled.fillCircle(48 + i * 16, 48 + yOff, 3, SSD1306_WHITE);
        }

    } else {
        // Running or Paused: show MM:SS countdown + progress bar.
        const uint32_t totalSec = remainingMs / 1000;
        const uint8_t  mm       = totalSec / 60;
        const uint8_t  ss       = totalSec % 60;

        // Header
        oled.setTextSize(1);
        oled.setTextColor(SSD1306_WHITE);
        oled.setCursor(34, 2);
        oled.print("TEA TIMER");

        oled.drawLine(0, 11, OLED_W - 1, 11, SSD1306_WHITE);

        // Big MM:SS centered (textSize 3 = 18 px wide per char, "MM:SS" = 90 px)
        char buf[6];
        snprintf(buf, sizeof(buf), "%02u:%02u", mm, ss);
        oled.setTextSize(3);
        oled.setCursor((OLED_W - 90) / 2, 16);
        oled.print(buf);

        // "PAUSED" badge when paused
        if (state == timer::PAUSED) {
            oled.setTextSize(1);
            oled.setCursor(44, 42);
            oled.print("PAUSED");
        }

        // Progress bar (x=2..125, y=54..58, height=4)
        oled.drawRect(2, 54, OLED_W - 4, 5, SSD1306_WHITE);
        const int barW = (int)((OLED_W - 6) * progress);
        if (barW > 0) {
            oled.fillRect(3, 55, barW, 3, SSD1306_WHITE);
        }
    }

    oled.display();
}

}
