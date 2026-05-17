#include "timer.h"

namespace timer {

static State    s_state      = IDLE;
static uint32_t s_startedAt  = 0;  // millis() when current RUNNING phase began
static uint32_t s_elapsed    = 0;  // ms accumulated across pause cycles
static uint32_t s_doneUntil  = 0;  // millis() when DONE celebration ends

// ---- public API ----

void tap() {
    const uint32_t now = millis();
    switch (s_state) {
    case IDLE:
        s_state     = RUNNING;
        s_startedAt = now;
        s_elapsed   = 0;
        break;
    case RUNNING:
        s_elapsed  += now - s_startedAt;
        s_state     = PAUSED;
        break;
    case PAUSED:
        s_startedAt = now;
        s_state     = RUNNING;
        break;
    case DONE:
        s_state   = IDLE;
        s_elapsed = 0;
        break;
    }
}

void reset() {
    s_state   = IDLE;
    s_elapsed = 0;
}

void tick(uint32_t nowMs) {
    if (s_state == RUNNING) {
        const uint32_t total = s_elapsed + (nowMs - s_startedAt);
        if (total >= BREW_MS) {
            s_state     = DONE;
            s_doneUntil = nowMs + DONE_MS;
            s_elapsed   = BREW_MS;
        }
    } else if (s_state == DONE) {
        if (nowMs >= s_doneUntil) {
            s_state   = IDLE;
            s_elapsed = 0;
        }
    }
}

State getState() { return s_state; }

uint32_t remainingMs(uint32_t nowMs) {
    switch (s_state) {
    case RUNNING: {
        const uint32_t used = s_elapsed + (nowMs - s_startedAt);
        return (used >= BREW_MS) ? 0 : (BREW_MS - used);
    }
    case PAUSED:
        return (s_elapsed >= BREW_MS) ? 0 : (BREW_MS - s_elapsed);
    case DONE:
        return (nowMs >= s_doneUntil) ? 0 : (s_doneUntil - nowMs);
    default:
        return BREW_MS;
    }
}

float progress(uint32_t nowMs) {
    switch (s_state) {
    case RUNNING: {
        const uint32_t used = s_elapsed + (nowMs - s_startedAt);
        return (used >= BREW_MS) ? 1.0f : (float)used / (float)BREW_MS;
    }
    case PAUSED:
        return (s_elapsed >= BREW_MS) ? 1.0f : (float)s_elapsed / (float)BREW_MS;
    default:
        return 0.0f;
    }
}

} // namespace timer
