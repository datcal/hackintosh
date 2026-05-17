package sources

import (
	"context"
	"time"

	"github.com/datcal/hackintosh/host/internal/store"
)

// GermanWorker selects the daily German word from the embedded list and
// publishes it to the store. No network calls — fully offline.
type GermanWorker struct {
	S       *store.Store
	lastDay int
}

func (w *GermanWorker) Run(ctx context.Context) {
	t := time.NewTicker(1 * time.Hour)
	defer t.Stop()
	w.refresh(time.Now())
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-t.C:
			w.refresh(now)
		}
	}
}

func (w *GermanWorker) refresh(now time.Time) {
	day := now.YearDay()
	if day == w.lastDay {
		return
	}
	w.lastDay = day
	idx := (day - 1) % len(GermanWords)
	entry := GermanWords[idx]
	w.S.SetGermanWord(store.GermanWord{
		Valid:   true,
		German:  entry.German,
		Turkish: entry.Turkish,
		English: entry.English,
		DayIdx:  idx,
	})
}
