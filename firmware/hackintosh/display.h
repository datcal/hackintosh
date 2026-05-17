#pragma once
#include <Arduino.h>
#include "timer.h"

namespace display {

bool begin();
void blitFrame(const uint8_t* frame1024);
void setBrightness(uint8_t b);
void drawBootSplash();
void drawDisconnectSplash(uint32_t tickMs);
void drawBootloaderEnter();
void drawTimerDisplay(timer::State state, uint32_t remainingMs, float progress, uint32_t tickMs);

}
