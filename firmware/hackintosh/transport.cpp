#include "transport.h"

namespace transport {

// ---------- CRC16/CCITT-FALSE (poly 0x1021, init 0xFFFF) ----------
static uint16_t crc16(const uint8_t* data, size_t len) {
    uint16_t crc = 0xFFFF;
    for (size_t i = 0; i < len; i++) {
        crc ^= (uint16_t)data[i] << 8;
        for (uint8_t b = 0; b < 8; b++) {
            crc = (crc & 0x8000) ? (uint16_t)((crc << 1) ^ 0x1021) : (uint16_t)(crc << 1);
        }
    }
    return crc;
}

// ---------- Inbound parser state machine ----------
enum class RxState : uint8_t { WaitMagic0, WaitMagic1, Type, LenLo, LenHi, Payload, CrcLo, CrcHi };

static RxState   rxState         = RxState::WaitMagic0;
static uint8_t   rxType          = 0;
static uint16_t  rxLen           = 0;
static uint16_t  rxIdx           = 0;
static uint16_t  rxCrcRecv       = 0;
// rxBuf holds [type byte][payload bytes...] so it must be 1 larger than the
// maximum payload size. With FRAME_BYTES = 1024, max payload is 1024 and we
// need 1025 bytes total. Bug: previously sized as [FRAME_BYTES], which
// overflowed on every FRAME message and corrupted adjacent globals.
static uint8_t   rxBuf[FRAME_BYTES + 1];
static uint8_t   frameBuf[FRAME_BYTES];
static bool      frameReady      = false;
static uint32_t  lastFrameMs     = 0;

static void resync() { rxState = RxState::WaitMagic0; }

static void dispatch() {
    const uint16_t crcCalc = crc16(rxBuf, rxLen + 1);  // include type byte
    if (crcCalc != rxCrcRecv) { resync(); return; }

    switch (rxType) {
        case TYPE_FRAME:
            if (rxLen == FRAME_BYTES) {
                memcpy(frameBuf, rxBuf + 1, FRAME_BYTES);
                frameReady   = true;
                lastFrameMs  = millis();
            }
            break;
        case TYPE_BRIGHTNESS:
            // handled in main loop; expose via consumer (we just log it for now)
            // future: set a flag + value
            break;
        case TYPE_PING:
            sendPong();
            break;
        default:
            break;
    }
    resync();
}

void poll() {
    while (Serial.available() > 0) {
        const int c = Serial.read();
        if (c < 0) break;
        const uint8_t b = (uint8_t)c;

        switch (rxState) {
            case RxState::WaitMagic0:
                if (b == PROTO_MAGIC_0) rxState = RxState::WaitMagic1;
                break;
            case RxState::WaitMagic1:
                rxState = (b == PROTO_MAGIC_1) ? RxState::Type : RxState::WaitMagic0;
                break;
            case RxState::Type:
                rxType    = b;
                rxBuf[0]  = b;   // type included in CRC
                rxState   = RxState::LenLo;
                break;
            case RxState::LenLo:
                rxLen   = b;
                rxState = RxState::LenHi;
                break;
            case RxState::LenHi:
                rxLen  |= (uint16_t)b << 8;
                rxIdx   = 0;
                if (rxLen > FRAME_BYTES) { resync(); break; }
                rxState = (rxLen == 0) ? RxState::CrcLo : RxState::Payload;
                break;
            case RxState::Payload:
                rxBuf[1 + rxIdx++] = b;
                if (rxIdx >= rxLen) rxState = RxState::CrcLo;
                break;
            case RxState::CrcLo:
                rxCrcRecv = b;
                rxState   = RxState::CrcHi;
                break;
            case RxState::CrcHi:
                rxCrcRecv |= (uint16_t)b << 8;
                dispatch();
                break;
        }
    }
}

// ---------- Outbound ----------
static void sendFrame(uint8_t type, const uint8_t* payload, uint16_t len) {
    uint8_t hdr[5];
    hdr[0] = PROTO_MAGIC_0;
    hdr[1] = PROTO_MAGIC_1;
    hdr[2] = type;
    hdr[3] = (uint8_t)(len & 0xFF);
    hdr[4] = (uint8_t)(len >> 8);
    Serial.write(hdr, 5);
    if (len > 0) Serial.write(payload, len);

    // CRC over type + payload
    uint16_t crc = 0xFFFF;
    {
        const uint8_t t = type;
        crc ^= (uint16_t)t << 8;
        for (uint8_t b = 0; b < 8; b++)
            crc = (crc & 0x8000) ? (uint16_t)((crc << 1) ^ 0x1021) : (uint16_t)(crc << 1);
    }
    for (uint16_t i = 0; i < len; i++) {
        crc ^= (uint16_t)payload[i] << 8;
        for (uint8_t b = 0; b < 8; b++)
            crc = (crc & 0x8000) ? (uint16_t)((crc << 1) ^ 0x1021) : (uint16_t)(crc << 1);
    }
    uint8_t crcBytes[2] = { (uint8_t)(crc & 0xFF), (uint8_t)(crc >> 8) };
    Serial.write(crcBytes, 2);
    Serial.flush();
}

void sendButton(uint8_t id, uint8_t evt) {
    uint8_t payload[2] = { id, evt };
    sendFrame(TYPE_BUTTON, payload, 2);
}

void sendPong()              { sendFrame(TYPE_PONG, nullptr, 0); }
void sendLog(const char* m)  { sendFrame(TYPE_LOG, (const uint8_t*)m, (uint16_t)strlen(m)); }

// ---------- State ----------
bool isConnected() {
    return lastFrameMs != 0 && (millis() - lastFrameMs) < DISCONNECT_TIMEOUT_MS;
}

const uint8_t* lastFrame() { return frameBuf; }

bool consumeNewFrame() {
    if (!frameReady) return false;
    frameReady = false;
    return true;
}

}
