package device

import (
	"context"
	"log"

	"github.com/datcal/hackintosh/host/internal/transport"
)

// Serial wraps the USB-CDC transport.Client and adapts its protocol events to
// the device interface used by the app.
type Serial struct {
	cli *transport.Client
	out chan ButtonEvent
}

// OpenSerial dials the named port and starts the inbound read loop.
func OpenSerial(ctx context.Context, portName string) (*Serial, error) {
	cli, err := transport.Open(portName)
	if err != nil {
		return nil, err
	}
	s := &Serial{cli: cli, out: make(chan ButtonEvent, 16)}
	go s.runReader(ctx)
	return s, nil
}

func (s *Serial) runReader(ctx context.Context) {
	defer close(s.out)
	// transport.Client.Run blocks until the port errors or ctx ends.
	go func() {
		if err := s.cli.Run(ctx); err != nil && err != context.Canceled {
			log.Printf("serial: read loop ended: %v", err)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-s.cli.Events():
			if !ok {
				return
			}
			switch ev.Type {
			case transport.TypeButton:
				if len(ev.Payload) != 2 {
					continue
				}
				select {
				case s.out <- ButtonEvent{ID: ev.Payload[0], Event: ev.Payload[1]}:
				default:
					// drop if app isn't consuming fast enough
				}
			case transport.TypeLog:
				log.Printf("MCU: %s", string(ev.Payload))
			}
		}
	}
}

func (s *Serial) SendFrame(frame []byte) error    { return s.cli.SendFrame(frame) }
func (s *Serial) SendBrightness(b byte) error     { return s.cli.SendBrightness(b) }
func (s *Serial) Buttons() <-chan ButtonEvent     { return s.out }
func (s *Serial) Close() error                    { return s.cli.Close() }
