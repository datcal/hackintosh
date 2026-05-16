// Package anim provides time-based animation primitives used across all
// screens: critically-damped springs for data smoothing, easings for one-shot
// transitions, periodic loops for ambient motion, and a particle system.
package anim

import (
	"math"
	"math/rand/v2"
)

// Spring smooths a scalar value toward a target with mass/damping physics.
// Typical use: each frame, set Target to the latest sample, then call Step(dt)
// — Value glides naturally toward Target without snapping.
type Spring struct {
	Value, Target  float64
	Velocity       float64
	Stiffness      float64 // spring constant (k); higher = snappier
	Damping        float64 // damping coefficient (c); higher = less oscillation
}

// NewSpring returns a spring tuned for "settles in ~0.5s with slight overshoot".
func NewSpring(initial float64) *Spring {
	return &Spring{Value: initial, Target: initial, Stiffness: 120, Damping: 14}
}

// Step advances the spring by dt seconds.
func (s *Spring) Step(dt float64) {
	a := -s.Stiffness*(s.Value-s.Target) - s.Damping*s.Velocity
	s.Velocity += a * dt
	s.Value += s.Velocity * dt
}

// SetSnap forces value and velocity. Use on construction or to clear motion.
func (s *Spring) SetSnap(v float64) {
	s.Value, s.Target, s.Velocity = v, v, 0
}

// Ease functions: input t is normalized 0..1.
func EaseOutCubic(t float64) float64 {
	t = clamp01(t)
	u := 1 - t
	return 1 - u*u*u
}

func EaseInOutSine(t float64) float64 {
	t = clamp01(t)
	return 0.5 - 0.5*math.Cos(math.Pi*t)
}

func EaseInOutCubic(t float64) float64 {
	t = clamp01(t)
	if t < 0.5 {
		return 4 * t * t * t
	}
	u := -2*t + 2
	return 1 - u*u*u/2
}

// Loop returns a phase in [0, 1) given a wall-clock time and a period.
func Loop(tSeconds, periodSeconds float64) float64 {
	if periodSeconds <= 0 {
		return 0
	}
	p := math.Mod(tSeconds, periodSeconds) / periodSeconds
	if p < 0 {
		p += 1
	}
	return p
}

// Wobble returns a deterministic-noise-like value in [-1, 1] using sums of
// out-of-phase sinusoids. Each element should use a unique `seed`.
func Wobble(tSeconds float64, freq float64, seed int) float64 {
	s := float64(seed)
	return 0.5*math.Sin(2*math.Pi*freq*tSeconds+s*0.731) +
		0.5*math.Sin(2*math.Pi*freq*1.71*tSeconds+s*1.913)
}

// Marquee scrolls a string-width through a window. Call Tick(dt) each frame,
// then Offset() to get the current x offset.
type Marquee struct {
	speed   float64 // pixels per second
	pauseS  float64 // pause at start of each cycle, seconds
	width   int     // pixel width of the text
	window  int     // pixel width of the visible area
	gap     int     // gap between repetitions
	t       float64
}

func NewMarquee(textWidth, windowWidth int) *Marquee {
	return &Marquee{
		speed:  18,
		pauseS: 0.8,
		width:  textWidth,
		window: windowWidth,
		gap:    16,
	}
}

func (m *Marquee) SetText(width int) { m.width = width; m.t = 0 }

// Tick advances time. Pass dt in seconds.
func (m *Marquee) Tick(dt float64) { m.t += dt }

// Offset returns the negative-or-zero x offset at which the text should be
// drawn this frame. Returns 0 if the text fits in the window.
func (m *Marquee) Offset() int {
	if m.width <= m.window {
		return 0
	}
	cycleDist := float64(m.width + m.gap)
	cycleTime := m.pauseS + cycleDist/m.speed
	phase := math.Mod(m.t, cycleTime)
	if phase < m.pauseS {
		return 0
	}
	return -int((phase - m.pauseS) * m.speed)
}

// Particle is a tiny physics body used for rain, snow, sparks.
type Particle struct {
	X, Y     float64
	VX, VY   float64
	Life, Max float64 // current life and max life, in seconds; 0 means infinite
	Active   bool
}

// Step advances the particle by dt.
func (p *Particle) Step(dt float64) {
	if !p.Active {
		return
	}
	p.X += p.VX * dt
	p.Y += p.VY * dt
	if p.Max > 0 {
		p.Life -= dt
		if p.Life <= 0 {
			p.Active = false
		}
	}
}

// ParticleSystem is a fixed-size pool of particles with a spawn function.
type ParticleSystem struct {
	P     []Particle
	Spawn func(p *Particle, r *rand.Rand)
	rand  *rand.Rand
}

func NewParticleSystem(count int, seed uint64, spawn func(p *Particle, r *rand.Rand)) *ParticleSystem {
	ps := &ParticleSystem{
		P:     make([]Particle, count),
		Spawn: spawn,
		rand:  rand.New(rand.NewPCG(seed, seed*2654435761)),
	}
	for i := range ps.P {
		ps.Spawn(&ps.P[i], ps.rand)
		ps.P[i].Active = true
	}
	return ps
}

// Step advances all particles. Particles that go inactive are auto-respawned.
func (ps *ParticleSystem) Step(dt float64) {
	for i := range ps.P {
		ps.P[i].Step(dt)
		if !ps.P[i].Active {
			ps.Spawn(&ps.P[i], ps.rand)
			ps.P[i].Active = true
		}
	}
}

// Random returns the system's RNG, useful for spawn closures and ad-hoc jitter.
func (ps *ParticleSystem) Random() *rand.Rand { return ps.rand }

func clamp01(t float64) float64 {
	if t < 0 {
		return 0
	}
	if t > 1 {
		return 1
	}
	return t
}

// Lerp interpolates a..b by t.
func Lerp(a, b, t float64) float64 { return a + (b-a)*t }
