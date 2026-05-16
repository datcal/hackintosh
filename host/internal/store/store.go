// Package store aggregates data from all sources behind a single thread-safe
// snapshot type. Sources publish via setters; the render loop reads a
// `Snapshot()` once per frame and uses it without holding any lock.
package store

import (
	"sync"
	"time"
)

// Weather state.
type Weather struct {
	Valid      bool
	TempC      float64
	FeelsLikeC float64
	WindKMH    float64
	Condition  Condition
	UpdatedAt  time.Time
	LocationName string
}

// Condition captures broad weather categories so screens can pick an animation.
type Condition int

const (
	CondUnknown Condition = iota
	CondSunny
	CondCloudy
	CondRain
	CondSnow
	CondThunder
	CondFog
)

func (c Condition) String() string {
	switch c {
	case CondSunny:   return "Sunny"
	case CondCloudy:  return "Cloudy"
	case CondRain:    return "Rain"
	case CondSnow:    return "Snow"
	case CondThunder: return "Thunder"
	case CondFog:     return "Fog"
	}
	return "—"
}

// AirQuality state.
type AirQuality struct {
	Valid     bool
	AQI       int    // EU AQI 1..5 (we keep raw integer)
	PM25      float64
	PM10      float64
	UpdatedAt time.Time
}

// AQI level label.
func (a AirQuality) Label() string {
	switch a.AQI {
	case 1:
		return "Good"
	case 2:
		return "Fair"
	case 3:
		return "Moderate"
	case 4:
		return "Poor"
	case 5:
		return "V.Poor"
	}
	return "—"
}

// Currency state.
type Currency struct {
	Valid     bool
	EURTRY    float64
	USDTRY    float64
	EURTRYPct float64 // 24h % change
	USDTRYPct float64
	UpdatedAt time.Time
	// SparkEUR/USD hold the last hour of samples (most recent last). Bounded
	// to ~60 entries; renderer treats whatever's there as the data window.
	SparkEUR []float64
	SparkUSD []float64
}

// Hardware monitor state.
type HW struct {
	Valid     bool
	CPUPct    float64
	RAMPct    float64
	DiskPct   float64
	NetUpKBs  float64
	NetDownKBs float64
	UpdatedAt time.Time
	Uptime    time.Duration
}

// Media state.
type Media struct {
	Valid    bool      // false = idle / no playback at all
	Title    string
	Artist   string
	Playing  bool      // true if playback is active (vs paused)
	Position time.Duration
	Length   time.Duration
	UpdatedAt time.Time
}

// Store is the central, thread-safe snapshot container.
type Store struct {
	mu      sync.RWMutex
	weather Weather
	aq      AirQuality
	cur     Currency
	hw      HW
	media   Media
}

func New() *Store { return &Store{} }

// Snapshot is what the render loop reads each frame.
type Snapshot struct {
	Weather Weather
	AQ      AirQuality
	Currency Currency
	HW      HW
	Media   Media
}

// Snapshot returns a copy safe for concurrent reading without locking.
func (s *Store) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return Snapshot{
		Weather:  s.weather,
		AQ:       s.aq,
		Currency: s.cur,
		HW:       s.hw,
		Media:    s.media,
	}
}

func (s *Store) SetWeather(w Weather)   { s.mu.Lock(); s.weather = w; s.mu.Unlock() }
func (s *Store) SetAirQuality(a AirQuality) { s.mu.Lock(); s.aq = a; s.mu.Unlock() }
func (s *Store) SetCurrency(c Currency) { s.mu.Lock(); s.cur = c; s.mu.Unlock() }
func (s *Store) SetHW(h HW)             { s.mu.Lock(); s.hw = h; s.mu.Unlock() }
func (s *Store) SetMedia(m Media)       { s.mu.Lock(); s.media = m; s.mu.Unlock() }
