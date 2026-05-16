package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/datcal/hackintosh/host/internal/store"
)

// CurrencyWorker polls a free FX API for EUR/TRY and USD/TRY rates.
// Uses Fawaz Ahmed's currency-api (no key, generous limits, CDN-served).
//
//	Endpoint: https://cdn.jsdelivr.net/npm/@fawazahmed0/currency-api@latest/v1/currencies/<base>.json
//	Returns:  { "date": "...", "<base>": { "<quote>": rate, ... } }
type CurrencyWorker struct {
	S      *store.Store
	Period time.Duration // default 5 min
	Client *http.Client

	sparkEUR []float64
	sparkUSD []float64
}

func (c *CurrencyWorker) Run(ctx context.Context) {
	if c.Period == 0 {
		c.Period = 5 * time.Minute
	}
	if c.Client == nil {
		c.Client = &http.Client{Timeout: 8 * time.Second}
	}
	t := time.NewTicker(c.Period)
	defer t.Stop()

	c.refresh(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			c.refresh(ctx)
		}
	}
}

func (c *CurrencyWorker) refresh(ctx context.Context) {
	eurNow, err := c.fetchRate(ctx, "eur", "try")
	if err != nil {
		log.Printf("currency: EUR fetch: %v", err)
		return
	}
	usdNow, err := c.fetchRate(ctx, "usd", "try")
	if err != nil {
		log.Printf("currency: USD fetch: %v", err)
		return
	}

	c.sparkEUR = appendSpark(c.sparkEUR, eurNow, 60)
	c.sparkUSD = appendSpark(c.sparkUSD, usdNow, 60)

	pctEUR := pctChange(c.sparkEUR)
	pctUSD := pctChange(c.sparkUSD)

	c.S.SetCurrency(store.Currency{
		Valid:     true,
		EURTRY:    eurNow,
		USDTRY:    usdNow,
		EURTRYPct: pctEUR,
		USDTRYPct: pctUSD,
		SparkEUR:  append([]float64(nil), c.sparkEUR...),
		SparkUSD:  append([]float64(nil), c.sparkUSD...),
		UpdatedAt: time.Now(),
	})
}

func (c *CurrencyWorker) fetchRate(ctx context.Context, base, quote string) (float64, error) {
	url := fmt.Sprintf(
		"https://cdn.jsdelivr.net/npm/@fawazahmed0/currency-api@latest/v1/currencies/%s.json",
		base)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil { return 0, err }
	resp, err := c.Client.Do(req)
	if err != nil { return 0, err }
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("status %d", resp.StatusCode)
	}
	var data map[string]json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil { return 0, err }
	raw, ok := data[base]
	if !ok {
		return 0, fmt.Errorf("missing base %q", base)
	}
	var rates map[string]float64
	if err := json.Unmarshal(raw, &rates); err != nil { return 0, err }
	rate, ok := rates[quote]
	if !ok {
		return 0, fmt.Errorf("missing quote %q", quote)
	}
	return rate, nil
}

func appendSpark(s []float64, v float64, max int) []float64 {
	s = append(s, v)
	if len(s) > max {
		s = s[len(s)-max:]
	}
	return s
}

func pctChange(s []float64) float64 {
	if len(s) < 2 {
		return 0
	}
	prev := s[0]
	if prev == 0 {
		return 0
	}
	cur := s[len(s)-1]
	return (cur - prev) / prev * 100
}
