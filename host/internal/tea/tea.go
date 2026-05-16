// Package tea implements a 3-minute tea brewing timer driven by button A.
//
// State machine:
//
//	Idle    -> tap A       -> Running (3 min countdown)
//	Running -> tap A       -> Paused (keeps remaining time)
//	Paused  -> tap A       -> Running (resumes)
//	Running -> elapsed=0   -> Done    (10 sec celebration overlay)
//	Done    -> elapsed=0   -> Idle    (auto-return)
//	Done    -> tap A       -> Idle    (dismiss celebration early)
//	any     -> long-press  -> Idle    (reset)
package tea

import (
	"sync"
	"time"
)

// State is the timer's current phase.
type State int

const (
	Idle State = iota
	Running
	Paused
	Done // 10-second celebration after the timer expires
)

const (
	// BrewDuration is how long one tap of A counts down. Adjust to taste —
	// some teas want 2 min, herbal blends 5-7 min, etc.
	BrewDuration = 3 * time.Minute

	// DoneDuration is how long the "TEA!" celebration overlay stays up
	// before the timer auto-returns to Idle.
	DoneDuration = 10 * time.Second
)

// Timer is safe for concurrent reads from the render loop and writes from the
// input handler.
type Timer struct {
	mu        sync.Mutex
	state     State
	startedAt time.Time     // wall-clock start of the current Running phase
	pausedAt  time.Time     // wall-clock when paused (during Paused state)
	elapsed   time.Duration // accumulated brew elapsed across pause cycles
	doneUntil time.Time     // wall-clock end of the Done celebration window
}

// New returns a timer in the Idle state.
func New() *Timer { return &Timer{state: Idle} }

// Snapshot is a point-in-time view of the timer for rendering.
type Snapshot struct {
	State     State
	Elapsed   time.Duration
	Total     time.Duration
	Remaining time.Duration
	Progress  float64 // 0..1 — 0 = just started, 1 = done
}

// Snapshot returns the current rendering state. Safe to call from the render
// loop. May internally transition Running → Done or Done → Idle when their
// timers expire, since those transitions are time-driven not input-driven.
func (t *Timer) Snapshot() Snapshot {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := time.Now()

	switch t.state {
	case Idle:
		return Snapshot{State: Idle, Total: BrewDuration}
	case Paused:
		return Snapshot{
			State:     Paused,
			Elapsed:   t.elapsed,
			Total:     BrewDuration,
			Remaining: BrewDuration - t.elapsed,
			Progress:  float64(t.elapsed) / float64(BrewDuration),
		}
	case Running:
		e := t.elapsed + now.Sub(t.startedAt)
		if e >= BrewDuration {
			t.state = Done
			t.doneUntil = now.Add(DoneDuration)
			return Snapshot{State: Done, Total: DoneDuration, Remaining: DoneDuration}
		}
		return Snapshot{
			State:     Running,
			Elapsed:   e,
			Total:     BrewDuration,
			Remaining: BrewDuration - e,
			Progress:  float64(e) / float64(BrewDuration),
		}
	case Done:
		rem := t.doneUntil.Sub(now)
		if rem <= 0 {
			t.state = Idle
			t.elapsed = 0
			return Snapshot{State: Idle, Total: BrewDuration}
		}
		return Snapshot{
			State:     Done,
			Remaining: rem,
			Total:     DoneDuration,
			Progress:  1 - float64(rem)/float64(DoneDuration),
		}
	}
	return Snapshot{State: Idle, Total: BrewDuration}
}

// Tap handles a short press of button A.
func (t *Timer) Tap() {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := time.Now()
	switch t.state {
	case Idle:
		t.state = Running
		t.startedAt = now
		t.elapsed = 0
	case Running:
		t.state = Paused
		t.elapsed += now.Sub(t.startedAt)
		t.pausedAt = now
	case Paused:
		t.state = Running
		t.startedAt = now
	case Done:
		// Dismiss the celebration overlay early.
		t.state = Idle
		t.elapsed = 0
	}
}

// Reset handles a long press of button A (cancel the brew).
func (t *Timer) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.state = Idle
	t.elapsed = 0
}

// Active reports whether the timer should be drawn as an overlay
// (anything other than Idle).
func (s Snapshot) Active() bool { return s.State != Idle }
