package device

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

// Simulator hosts an HTTP server that streams framebuffers to a browser-based
// virtual OLED via Server-Sent Events and accepts button presses via POST.
//
// Endpoints:
//
//	GET  /            -> the HTML viewer
//	GET  /frames      -> SSE stream of base64-encoded 1024-byte frames
//	POST /button      -> {"id":0|1, "event":"press"|"release"|"long"}
//	GET  /health      -> liveness probe
type Simulator struct {
	addr string

	mu          sync.Mutex
	current     []byte
	subscribers map[chan []byte]struct{}

	buttons chan ButtonEvent

	server *http.Server
}

// NewSimulator binds the server to addr (e.g. ":8080" or "127.0.0.1:8080")
// and starts serving immediately. Call Close() to shut down.
func NewSimulator(addr string) (*Simulator, error) {
	s := &Simulator{
		addr:        addr,
		subscribers: make(map[chan []byte]struct{}),
		buttons:     make(chan ButtonEvent, 16),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/frames", s.handleFrames)
	mux.HandleFunc("/button", s.handleButton)
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	})
	s.server = &http.Server{Addr: addr, Handler: mux}

	ln, err := listenWithLog(addr)
	if err != nil {
		return nil, err
	}
	go func() {
		if err := s.server.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Printf("simulator: server: %v", err)
		}
	}()
	return s, nil
}

// SendFrame caches the latest framebuffer and pushes it to all subscribers.
// Slow subscribers drop frames rather than block.
func (s *Simulator) SendFrame(frame []byte) error {
	frameCopy := make([]byte, len(frame))
	copy(frameCopy, frame)

	s.mu.Lock()
	s.current = frameCopy
	subs := make([]chan []byte, 0, len(s.subscribers))
	for ch := range s.subscribers {
		subs = append(subs, ch)
	}
	s.mu.Unlock()

	for _, ch := range subs {
		select {
		case ch <- frameCopy:
		default:
			// drop
		}
	}
	return nil
}

// SendBrightness is a no-op for the simulator.
func (s *Simulator) SendBrightness(_ byte) error { return nil }

// Buttons returns the virtual button event stream.
func (s *Simulator) Buttons() <-chan ButtonEvent { return s.buttons }

// Close stops the HTTP server and unblocks readers.
func (s *Simulator) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := s.server.Shutdown(ctx)
	s.mu.Lock()
	for ch := range s.subscribers {
		close(ch)
		delete(s.subscribers, ch)
	}
	s.mu.Unlock()
	close(s.buttons)
	return err
}

func (s *Simulator) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, viewerHTML)
}

func (s *Simulator) handleFrames(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", 500)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher.Flush()

	ch := make(chan []byte, 4)
	s.mu.Lock()
	s.subscribers[ch] = struct{}{}
	current := s.current
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.subscribers, ch)
		s.mu.Unlock()
	}()

	send := func(frame []byte) bool {
		_, err := fmt.Fprintf(w, "data: %s\n\n", base64.StdEncoding.EncodeToString(frame))
		if err != nil {
			return false
		}
		flusher.Flush()
		return true
	}

	if current != nil {
		if !send(current) {
			return
		}
	}

	for {
		select {
		case <-r.Context().Done():
			return
		case frame, ok := <-ch:
			if !ok {
				return
			}
			if !send(frame) {
				return
			}
		}
	}
}

func (s *Simulator) handleButton(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", 405)
		return
	}
	var msg struct {
		ID    int    `json:"id"`
		Event string `json:"event"`
	}
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	var evt byte
	switch msg.Event {
	case "press":
		evt = EvtPress
	case "release":
		evt = EvtRelease
	case "long":
		evt = EvtLongPress
	default:
		http.Error(w, "unknown event", 400)
		return
	}
	id := byte(msg.ID)
	if id != BtnA && id != BtnB {
		http.Error(w, "id must be 0 or 1", 400)
		return
	}
	select {
	case s.buttons <- ButtonEvent{ID: id, Event: evt}:
		w.WriteHeader(204)
	default:
		http.Error(w, "queue full", 503)
	}
}

func listenWithLog(addr string) (lnNetListener, error) {
	ln, err := netListen("tcp", addr)
	if err != nil {
		return nil, err
	}
	log.Printf("simulator: listening on http://%s", ln.Addr())
	return ln, nil
}
