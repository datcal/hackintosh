#pragma once
#include <Arduino.h>

namespace buttons {

void begin();
// Polls both buttons; emits transport-level events for state changes + long-press.
void poll();

// Firmware-local event consumers for button A.
// Each returns true exactly once per event and then clears itself, so the
// main loop can claim the event without interfering with transport sends.
bool consumeAPress();
bool consumeALong();

}
