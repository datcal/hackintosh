// Package sources contains the data fetchers — small workers that periodically
// hit external APIs / OS facilities and push results into the store.
package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/datcal/hackintosh/host/internal/store"
)

// WeatherWorker fetches current weather + air quality from Open-Meteo,
// using ipapi.co for IP-based geolocation on first run.
type WeatherWorker struct {
	S      *store.Store
	Period time.Duration // how often to refetch; default 10 min
	Client *http.Client

	lat, lon  float64
	locName   string
	resolved  bool
}

// Run blocks until ctx is canceled, refreshing on each tick.
func (w *WeatherWorker) Run(ctx context.Context) {
	if w.Period == 0 {
		w.Period = 10 * time.Minute
	}
	if w.Client == nil {
		w.Client = &http.Client{Timeout: 8 * time.Second}
	}
	t := time.NewTicker(w.Period)
	defer t.Stop()

	w.refresh(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			w.refresh(ctx)
		}
	}
}

func (w *WeatherWorker) refresh(ctx context.Context) {
	if !w.resolved {
		if err := w.resolveLocation(ctx); err != nil {
			log.Printf("weather: all geo providers failed (%v); falling back to Istanbul. "+
				"To override, edit lat/lon in sources/weather.go.", err)
			w.lat, w.lon, w.locName = 41.0082, 28.9784, "Istanbul"
			w.resolved = true
		} else {
			log.Printf("weather: located in %s (%.2f, %.2f)", w.locName, w.lat, w.lon)
		}
	}
	if err := w.fetchWeather(ctx); err != nil {
		log.Printf("weather: fetch failed: %v", err)
	}
	if err := w.fetchAirQuality(ctx); err != nil {
		log.Printf("air-quality: fetch failed: %v", err)
	}
}

// resolveLocation tries multiple free geo-IP providers in sequence and uses
// the first one that returns a non-zero lat/lon. Each provider has different
// free-tier rate limits, IP-range coverage, and reliability — cascading
// covers individual outages and rate-limiting that would otherwise make the
// weather screen permanently fall back to a hard-coded default.
func (w *WeatherWorker) resolveLocation(ctx context.Context) error {
	providers := []struct {
		name string
		fn   func(context.Context) (lat, lon float64, city string, err error)
	}{
		{"ipwho.is", w.geoIPWhoIS},     // generous free tier, HTTPS, primary
		{"ipapi.co", w.geoIPAPICO},     // popular but rate-limited, fallback
		{"freeipapi.com", w.geoFreeIPAPI}, // independent CDN, second fallback
	}
	var errs []string
	for _, p := range providers {
		lat, lon, city, err := p.fn(ctx)
		if err == nil && (lat != 0 || lon != 0) {
			w.lat, w.lon, w.locName = lat, lon, city
			w.resolved = true
			return nil
		}
		if err != nil {
			errs = append(errs, p.name+": "+err.Error())
		} else {
			errs = append(errs, p.name+": empty location")
		}
	}
	return fmt.Errorf("%v", errs)
}

// geoIPWhoIS — https://ipwho.is — returns {"latitude":..,"longitude":..,"city":..,"success":true}
func (w *WeatherWorker) geoIPWhoIS(ctx context.Context) (float64, float64, string, error) {
	var data struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		City      string  `json:"city"`
		Success   bool    `json:"success"`
		Message   string  `json:"message"`
	}
	if err := w.fetchJSON(ctx, "https://ipwho.is/", &data); err != nil {
		return 0, 0, "", err
	}
	if !data.Success {
		msg := data.Message
		if msg == "" {
			msg = "provider reported failure"
		}
		return 0, 0, "", fmt.Errorf("%s", msg)
	}
	return data.Latitude, data.Longitude, data.City, nil
}

// geoIPAPICO — https://ipapi.co/json/ — same response shape, no success field.
func (w *WeatherWorker) geoIPAPICO(ctx context.Context) (float64, float64, string, error) {
	var data struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		City      string  `json:"city"`
		Error     bool    `json:"error"`
		Reason    string  `json:"reason"`
	}
	if err := w.fetchJSON(ctx, "https://ipapi.co/json/", &data); err != nil {
		return 0, 0, "", err
	}
	if data.Error {
		return 0, 0, "", fmt.Errorf("%s", data.Reason)
	}
	return data.Latitude, data.Longitude, data.City, nil
}

// geoFreeIPAPI — https://freeipapi.com/api/json — note: uses "cityName" not "city".
func (w *WeatherWorker) geoFreeIPAPI(ctx context.Context) (float64, float64, string, error) {
	var data struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		CityName  string  `json:"cityName"`
	}
	if err := w.fetchJSON(ctx, "https://freeipapi.com/api/json", &data); err != nil {
		return 0, 0, "", err
	}
	return data.Latitude, data.Longitude, data.CityName, nil
}

// fetchJSON is a small helper: GET the URL, decode into `out`, return error
// for any HTTP-level or parse-level problem.
func (w *WeatherWorker) fetchJSON(ctx context.Context, url string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "hackintosh/1.0")
	resp, err := w.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, body)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (w *WeatherWorker) fetchWeather(ctx context.Context) error {
	url := fmt.Sprintf(
		"https://api.open-meteo.com/v1/forecast?latitude=%f&longitude=%f"+
			"&current=temperature_2m,apparent_temperature,weather_code,wind_speed_10m"+
			"&timezone=auto",
		w.lat, w.lon)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil { return err }
	resp, err := w.Client.Do(req)
	if err != nil { return err }
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, b)
	}
	var data struct {
		Current struct {
			Temperature   float64 `json:"temperature_2m"`
			ApparentTemp  float64 `json:"apparent_temperature"`
			WeatherCode   int     `json:"weather_code"`
			WindSpeed     float64 `json:"wind_speed_10m"`
		} `json:"current"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil { return err }

	w.S.SetWeather(store.Weather{
		Valid:        true,
		TempC:        data.Current.Temperature,
		FeelsLikeC:   data.Current.ApparentTemp,
		WindKMH:      data.Current.WindSpeed,
		Condition:    weatherCodeToCondition(data.Current.WeatherCode),
		UpdatedAt:    time.Now(),
		LocationName: w.locName,
	})
	return nil
}

func (w *WeatherWorker) fetchAirQuality(ctx context.Context) error {
	url := fmt.Sprintf(
		"https://air-quality-api.open-meteo.com/v1/air-quality?latitude=%f&longitude=%f"+
			"&current=pm2_5,pm10,european_aqi",
		w.lat, w.lon)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil { return err }
	resp, err := w.Client.Do(req)
	if err != nil { return err }
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, b)
	}
	var data struct {
		Current struct {
			PM25 float64 `json:"pm2_5"`
			PM10 float64 `json:"pm10"`
			AQI  float64 `json:"european_aqi"`
		} `json:"current"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil { return err }

	// Map Open-Meteo's continuous EU AQI to the integer band 1..5.
	band := 1
	switch {
	case data.Current.AQI <= 20:
		band = 1
	case data.Current.AQI <= 40:
		band = 2
	case data.Current.AQI <= 60:
		band = 3
	case data.Current.AQI <= 80:
		band = 4
	default:
		band = 5
	}

	w.S.SetAirQuality(store.AirQuality{
		Valid:     true,
		AQI:       band,
		PM25:      data.Current.PM25,
		PM10:      data.Current.PM10,
		UpdatedAt: time.Now(),
	})
	return nil
}

// WMO weather codes (https://open-meteo.com/en/docs#weathervariables) bucketed
// into our coarse condition enum.
func weatherCodeToCondition(code int) store.Condition {
	switch {
	case code == 0:
		return store.CondSunny
	case code <= 3:
		return store.CondCloudy
	case code <= 48:
		return store.CondFog
	case code <= 67, code >= 80 && code <= 82:
		return store.CondRain
	case code <= 77, code >= 85 && code <= 86:
		return store.CondSnow
	case code >= 95:
		return store.CondThunder
	}
	return store.CondUnknown
}
