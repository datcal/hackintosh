#pragma once
#include <Arduino.h>
#include "config.h"

namespace transport {

// Called by the main loop with whatever bytes Serial has buffered.
// Reassembles complete frames and dispatches via the callbacks below.
void poll();

// Send outbound frames to the host.
void sendButton(uint8_t id, uint8_t evt);
void sendPong();
void sendLog(const char* msg);

// True if we have received a valid FRAME within the last DISCONNECT_TIMEOUT_MS.
bool isConnected();

// Most recent received frame (1024 bytes); valid only after onFrameReady() in the main loop.
const uint8_t* lastFrame();
bool consumeNewFrame();  // returns true if a fresh frame is pending; resets the flag

}
