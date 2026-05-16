#include "display.h"
#include "config.h"

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

    const int cx = 28, cy = 32;
    const int r  = 14;
    oled.drawCircle(cx, cy, r,     SSD1306_WHITE);
    oled.drawCircle(cx, cy, r / 2, SSD1306_WHITE);

    const float angle = (tickMs / 1000.0f) * 2.0f * 3.14159f;
    const int ex = cx + (int)(cos(angle) * r);
    const int ey = cy + (int)(sin(angle) * r);
    oled.drawLine(cx, cy, ex, ey, SSD1306_WHITE);

    oled.setTextSize(1);
    oled.setTextColor(SSD1306_WHITE);
    oled.setCursor(54, 20);
    oled.print("Hackintosh");

    const uint8_t dotPhase = (tickMs / 500) % 4;
    oled.setCursor(54, 34);
    oled.print("searching");
    oled.setCursor(54, 44);
    for (uint8_t i = 0; i < dotPhase; i++) oled.print(".");

    oled.display();
}

}
