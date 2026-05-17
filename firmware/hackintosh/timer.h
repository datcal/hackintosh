#pragma once
#include <Arduino.h>

// Firmware-side 3-minute tea timer.
// Active only when the host is disconnected — when connected the host handles
// button A and renders the timer overlay in every framebuffer it sends.
//
// State machine (mirrors the Go tea package):
//   IDLE    -> tap()    -> RUNNING
//   RUNNING -> tap()    -> PAUSED
//   PAUSED  -> tap()    -> RUNNING
//   RUNNING -> tick(), elapsed >= BREW_MS -> DONE
//   DONE    -> tap()    -> IDLE (dismiss early)
//   DONE    -> tick(), now >= doneUntil  -> IDLE (auto-dismiss after 10 s)
//   any     -> reset()  -> IDLE

namespace timer {

constexpr uint32_t BREW_MS = 3UL * 60UL * 1000UL; // 3 minutes
constexpr uint32_t DONE_MS = 10UL * 1000UL;        // 10-second celebration

enum State { IDLE, RUNNING, PAUSED, DONE };

void tap();
void reset();
void tick(uint32_t nowMs);

State    getState();
uint32_t remainingMs(uint32_t nowMs); // ms left in current phase
float    progress(uint32_t nowMs);    // 0..1 brew fraction elapsed

} // namespace timer
