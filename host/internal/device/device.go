// Package device abstracts "where do frames go and where do button events come
// from" so the app loop can target either real hardware (serial) or a virtual
// device hosted in the browser (simulator).
package device

// ButtonEvent is the universal button input format consumed by the app.
type ButtonEvent struct {
	ID    byte // 0 = A, 1 = B
	Event byte // 0 = press, 1 = release, 2 = long-press
}

// Device is the contract every backend implements.
type Device interface {
	// SendFrame transmits a 1024-byte 1-bit framebuffer to the display.
	SendFrame(frame []byte) error
	// SendBrightness sets the OLED contrast (0..255). Simulator may ignore.
	SendBrightness(b byte) error
	// Buttons returns a channel of button events, closed when the device closes.
	Buttons() <-chan ButtonEvent
	// Close releases resources. Idempotent.
	Close() error
}

// Names for the protocol's button-event constants, mirrored from transport.
const (
	BtnA byte = 0
	BtnB byte = 1

	EvtPress     byte = 0
	EvtRelease   byte = 1
	EvtLongPress byte = 2
)
