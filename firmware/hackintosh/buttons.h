#pragma once
#include <Arduino.h>

namespace buttons {

void begin();
// Polls both buttons; emits transport-level events for state changes + long-press.
void poll();

}
