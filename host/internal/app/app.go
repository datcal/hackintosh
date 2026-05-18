// Package app drives the 30 FPS render loop, dispatches button input to the
// screen cycler and the tea timer, and ships frames to whichever device
// the caller wired in (serial or simulator).
package app

import (
	"context"
	"log"
	"time"

	"github.com/datcal/hackintosh/host/internal/device"
	"github.com/datcal/hackintosh/host/internal/tea"
	"github.com/datcal/hackintosh/host/internal/render"
	"github.com/datcal/hackintosh/host/internal/render/screens"
	"github.com/datcal/hackintosh/host/internal/store"
)

const (
	FrameRate           = 30   // frames per second
	AutoRotateInterval  = 8.0  // seconds per screen during auto-rotation
	ResumeAfterIdleSecs = 45.0 // seconds of no buttons before rotation resumes
)

// App owns the render loop. Construct with New(), then Run(ctx) blocks until
// ctx is canceled or the device closes.
type App struct {
	dev    device.Device
	store  *store.Store
	timer  *tea.Timer
	scrn   []screens.Screen
	idx    int

	chrome      *render.ChromeState
	transition  *transitionState

	idleSecs   float64 // seconds since the last button event
	rotateSecs float64 // seconds accumulated toward the next auto-rotation
}

type transitionState struct {
	active   bool
	t        float64
	from, to int
}

func New(dev device.Device, st *store.Store, timer *tea.Timer) *App {
	return &App{
		dev:    dev,
		store:  st,
		timer:  timer,
		scrn:   screens.All(),
		chrome: render.NewChromeState(),
		transition: &transitionState{},
		// Start past the idle threshold so auto-rotation kicks in on boot
		// instead of waiting 45s after the first frame.
		idleSecs: ResumeAfterIdleSecs,
	}
}

// Run blocks until ctx is canceled. Pumps frames at FrameRate and forwards
// inputs from the device.
func (a *App) Run(ctx context.Context) error {
	ticker := time.NewTicker(time.Second / FrameRate)
	defer ticker.Stop()

	last := time.Now()
	curFrame := render.New()
	outBuf := render.New()
	inBuf := render.New()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-a.dev.Buttons():
			if !ok {
				return nil
			}
			a.handleButton(ev)
		case now := <-ticker.C:
			dt := now.Sub(last).Seconds()
			if dt > 0.2 {
				dt = 0.2
			}
			last = now

			// Tick chrome + all screens. (Tick every screen, not just the
			// active one, so when you cycle to another the EQ meter / clouds
			// are already in flight.)
			a.chrome.Tick(dt)
			for _, s := range a.scrn {
				s.Tick(dt)
			}

			a.idleSecs += dt
			if a.idleSecs >= ResumeAfterIdleSecs {
				if !a.transition.active {
					a.rotateSecs += dt
					if a.rotateSecs >= AutoRotateInterval {
						a.startTransition()
						a.rotateSecs = 0
					}
				}
			} else {
				a.rotateSecs = 0
			}

			snap := a.store.Snapshot()

			if a.transition.active {
				a.transition.t += dt / render.TransitionDuration
				if a.transition.t >= 1 {
					a.transition.active = false
					a.idx = a.transition.to
				}
				// Render both screens into out/in buffers.
				outBuf.Clear()
				render.DrawChrome(outBuf, a.chrome, a.scrn[a.transition.from].Name(),
					a.timer.Snapshot())
				a.scrn[a.transition.from].Render(outBuf, snap, now)

				inBuf.Clear()
				render.DrawChrome(inBuf, a.chrome, a.scrn[a.transition.to].Name(),
					a.timer.Snapshot())
				a.scrn[a.transition.to].Render(inBuf, snap, now)

				curFrame.Clear()
				render.CompositeSlideUp(curFrame, outBuf, inBuf, a.transition.t)
				render.DrawTimerStrip(curFrame, a.timer.Snapshot(), float64(now.UnixNano())/1e9)
				render.DrawTimerDone(curFrame, a.timer.Snapshot(), float64(now.UnixNano())/1e9)
			} else {
				curFrame.Clear()
				render.DrawChrome(curFrame, a.chrome, a.scrn[a.idx].Name(), a.timer.Snapshot())
				a.scrn[a.idx].Render(curFrame, snap, now)
				render.DrawTimerStrip(curFrame, a.timer.Snapshot(), float64(now.UnixNano())/1e9)
				render.DrawTimerDone(curFrame, a.timer.Snapshot(), float64(now.UnixNano())/1e9)
			}

			if err := a.dev.SendFrame(curFrame.Bytes()); err != nil {
				log.Printf("app: send frame: %v", err)
			}
		}
	}
}

func (a *App) handleButton(ev device.ButtonEvent) {
	a.idleSecs = 0
	a.rotateSecs = 0
	switch ev.ID {
	case device.BtnA:
		switch ev.Event {
		case device.EvtPress:
			a.timer.Tap()
		case device.EvtLongPress:
			a.timer.Reset()
		}
	case device.BtnB:
		if ev.Event == device.EvtPress {
			a.startTransition()
		}
	}
}

func (a *App) startTransition() {
	if a.transition.active {
		return
	}
	a.transition.active = true
	a.transition.t = 0
	a.transition.from = a.idx
	a.transition.to = (a.idx + 1) % len(a.scrn)
}
