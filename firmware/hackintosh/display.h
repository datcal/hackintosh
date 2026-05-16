#pragma once
#include <Arduino.h>

namespace display {

bool begin();
void blitFrame(const uint8_t* frame1024);
void setBrightness(uint8_t b);
void drawBootSplash();
void drawDisconnectSplash(uint32_t tickMs);
void drawBootloaderEnter();

}
