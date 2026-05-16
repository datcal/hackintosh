// Package transport speaks the framed binary protocol shared with the MCU.
//
// Frame layout (both directions):
//
//	[0xAA 0x55] [type: u8] [len: u16 LE] [payload: len bytes] [crc16 LE]
//
// CRC16/CCITT-FALSE (poly 0x1021, init 0xFFFF) over type + payload.
package transport

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"go.bug.st/serial"
)

const (
	Magic0 byte = 0xAA
	Magic1 byte = 0x55

	TypeFrame      byte = 0x01
	TypeBrightness byte = 0x02
	TypePing       byte = 0x03

	TypeButton byte = 0x81
	TypePong   byte = 0x82
	TypeLog    byte = 0x83

	FrameBytes = 1024 // 128 * 64 / 8

	ButtonIDA      byte = 0
	ButtonIDB      byte = 1
	ButtonPress    byte = 0
	ButtonRelease  byte = 1
	ButtonLongPress byte = 2
)

// Event represents one inbound message from the MCU.
type Event struct {
	Type    byte
	Payload []byte
}

// Client owns a single serial port and runs the read loop.
type Client struct {
	port serial.Port

	mu       sync.Mutex
	closed   bool

	events chan Event
}

// Open dials the given serial port. Baud is ignored for USB-CDC but the
// underlying library still wants a value.
func Open(portName string) (*Client, error) {
	port, err := serial.Open(portName, &serial.Mode{BaudRate: 115200})
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", portName, err)
	}
	if err := port.SetReadTimeout(100 * time.Millisecond); err != nil {
		_ = port.Close()
		return nil, fmt.Errorf("set read timeout: %w", err)
	}
	return &Client{port: port, events: make(chan Event, 32)}, nil
}

// List returns the available serial ports for use by the caller's port-picker.
func List() ([]string, error) { return serial.GetPortsList() }

// Events returns a channel of inbound messages from the MCU. Closed when the
// client is closed or the port errors out.
func (c *Client) Events() <-chan Event { return c.events }

// Run pumps the read loop until ctx is cancelled or the port closes. Safe to
// call exactly once per Client; subsequent calls return immediately.
func (c *Client) Run(ctx context.Context) error {
	defer close(c.events)

	hdr := make([]byte, 5)
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// 1) Find magic. Read one byte at a time until we sync.
		if !c.findMagic(ctx) {
			return c.closedErr()
		}

		// 2) Header: type + len LE
		if _, err := io.ReadFull(c.port, hdr[2:5]); err != nil {
			if isTimeout(err) { continue }
			return err
		}
		msgType := hdr[2]
		msgLen  := binary.LittleEndian.Uint16(hdr[3:5])
		if msgLen > FrameBytes {
			continue // garbage; resync
		}

		// 3) Payload + CRC
		buf := make([]byte, int(msgLen)+2)
		if _, err := io.ReadFull(c.port, buf); err != nil {
			if isTimeout(err) { continue }
			return err
		}
		recvCRC := binary.LittleEndian.Uint16(buf[len(buf)-2:])
		payload := buf[:len(buf)-2]

		// 4) Validate CRC over type + payload
		expect := crc16(append([]byte{msgType}, payload...))
		if expect != recvCRC {
			continue
		}

		select {
		case c.events <- Event{Type: msgType, Payload: append([]byte(nil), payload...)}:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (c *Client) findMagic(ctx context.Context) bool {
	one := make([]byte, 1)
	state := 0
	for {
		if ctx.Err() != nil { return false }
		n, err := c.port.Read(one)
		if err != nil {
			if isTimeout(err) { continue }
			return false
		}
		if n == 0 { continue }
		switch state {
		case 0:
			if one[0] == Magic0 { state = 1 }
		case 1:
			if one[0] == Magic1 { return true }
			if one[0] != Magic0 { state = 0 }
		}
	}
}

// SendFrame ships a 1024-byte framebuffer to the MCU.
func (c *Client) SendFrame(frame []byte) error {
	if len(frame) != FrameBytes {
		return fmt.Errorf("frame must be %d bytes, got %d", FrameBytes, len(frame))
	}
	return c.write(TypeFrame, frame)
}

// SendBrightness sets the OLED contrast (0-255).
func (c *Client) SendBrightness(b byte) error {
	return c.write(TypeBrightness, []byte{b})
}

// Ping sends a PING; the MCU will reply with a PONG event.
func (c *Client) Ping() error { return c.write(TypePing, nil) }

func (c *Client) write(msgType byte, payload []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return errors.New("transport closed")
	}
	out := make([]byte, 5+len(payload)+2)
	out[0] = Magic0
	out[1] = Magic1
	out[2] = msgType
	binary.LittleEndian.PutUint16(out[3:5], uint16(len(payload)))
	copy(out[5:], payload)
	crc := crc16(append([]byte{msgType}, payload...))
	binary.LittleEndian.PutUint16(out[5+len(payload):], crc)
	_, err := c.port.Write(out)
	return err
}

// Close releases the port. Safe to call multiple times.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed { return nil }
	c.closed = true
	return c.port.Close()
}

func (c *Client) closedErr() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed { return io.EOF }
	return nil
}

// ---- helpers ----

func crc16(data []byte) uint16 {
	crc := uint16(0xFFFF)
	for _, b := range data {
		crc ^= uint16(b) << 8
		for i := 0; i < 8; i++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}

func isTimeout(err error) bool {
	type timeouter interface{ Timeout() bool }
	var t timeouter
	if errors.As(err, &t) {
		return t.Timeout()
	}
	// go.bug.st/serial returns no bytes + nil on read timeout, but on Windows
	// some configurations can surface as os.ErrDeadlineExceeded or io.EOF on
	// short reads. We're defensive here.
	return errors.Is(err, io.ErrUnexpectedEOF)
}
