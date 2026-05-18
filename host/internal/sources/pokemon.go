package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/datcal/hackintosh/host/internal/store"
)

const (
	pokemonGen1Count  = 151
	pokemonSpriteSize = 44 // target max dimension in pixels
)

// PokemonWorker fetches today's Gen 1 Pokémon from PokeAPI once per day.
// The daily index is (YearDay()-1) % 151. Sprites are cached to disk.
type PokemonWorker struct {
	S       *store.Store
	Client  *http.Client
	lastDay int
}

func (w *PokemonWorker) Run(ctx context.Context) {
	if w.Client == nil {
		w.Client = &http.Client{Timeout: 15 * time.Second}
	}
	t := time.NewTicker(1 * time.Hour)
	defer t.Stop()
	w.refresh(ctx, time.Now())
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-t.C:
			w.refresh(ctx, now)
		}
	}
}

func (w *PokemonWorker) refresh(ctx context.Context, now time.Time) {
	day := now.YearDay()
	if day == w.lastDay {
		return
	}
	w.lastDay = day

	idx := (day - 1) % pokemonGen1Count
	id := idx + 1 // PokeAPI IDs are 1-based

	name, type1, type2, err := w.fetchMeta(ctx, id)
	if err != nil {
		log.Printf("pokemon: meta fetch id=%d: %v", id, err)
		return
	}

	bitmap, bw, bh, err := w.fetchSprite(ctx, id)
	if err != nil {
		log.Printf("pokemon: sprite fetch id=%d: %v", id, err)
		// Store without sprite — screen shows name+types with no image.
	}

	w.S.SetPokemon(store.Pokemon{
		Valid:     true,
		ID:        id,
		Name:      name,
		Type1:     type1,
		Type2:     type2,
		Sprite:    bitmap,
		SpriteW:   bw,
		SpriteH:   bh,
		UpdatedAt: now,
	})
}

func (w *PokemonWorker) fetchMeta(ctx context.Context, id int) (name, type1, type2 string, err error) {
	url := fmt.Sprintf("https://pokeapi.co/api/v2/pokemon/%d", id)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("User-Agent", "hackintosh/1.0")
	resp, err := w.Client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var data struct {
		Name  string `json:"name"`
		Types []struct {
			Slot int `json:"slot"`
			Type struct {
				Name string `json:"name"`
			} `json:"type"`
		} `json:"types"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return
	}

	name = strings.Title(strings.ReplaceAll(data.Name, "-", " "))
	for _, t := range data.Types {
		if t.Slot == 1 {
			type1 = strings.Title(t.Type.Name)
		}
		if t.Slot == 2 {
			type2 = strings.Title(t.Type.Name)
		}
	}
	return
}

func (w *PokemonWorker) fetchSprite(ctx context.Context, id int) (bitmap []byte, bw, bh int, err error) {
	cacheDir, _ := os.UserCacheDir()
	binFile := filepath.Join(cacheDir, "hackintosh", fmt.Sprintf("pokemon-%d.bin", id))
	metaFile := filepath.Join(cacheDir, "hackintosh", fmt.Sprintf("pokemon-%d.meta", id))

	if data, rerr := os.ReadFile(binFile); rerr == nil {
		if meta, merr := os.ReadFile(metaFile); merr == nil {
			fmt.Sscanf(string(meta), "%dx%d", &bw, &bh)
			if bw > 0 && bh > 0 {
				return data, bw, bh, nil
			}
		}
	}

	url := fmt.Sprintf("https://raw.githubusercontent.com/PokeAPI/sprites/master/sprites/pokemon/%d.png", id)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := w.Client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	src, err := png.Decode(resp.Body)
	if err != nil {
		return
	}

	bitmap, bw, bh = convertSpriteTo1Bit(src)

	os.MkdirAll(filepath.Dir(binFile), 0o755)
	os.WriteFile(binFile, bitmap, 0o644)
	os.WriteFile(metaFile, []byte(fmt.Sprintf("%dx%d", bw, bh)), 0o644)
	return
}

// convertSpriteTo1Bit scales src to fit within pokemonSpriteSize×pokemonSpriteSize,
// then converts to a 1-bit MSB-first row-major bitmap via Floyd-Steinberg dithering.
// Dark pixels → bit SET (bright on the OLED).
func convertSpriteTo1Bit(src image.Image) (bitmap []byte, w, h int) {
	sb := src.Bounds()
	sw, sh := sb.Dx(), sb.Dy()

	// Scale to fit within pokemonSpriteSize preserving aspect ratio.
	scale := float64(pokemonSpriteSize) / float64(pokemonMax(sw, sh))
	w = int(math.Round(float64(sw)*scale))
	h = int(math.Round(float64(sh)*scale))
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}

	// Nearest-neighbor scale (pixel art sprites look fine without bilinear).
	gray := make([]float64, w*h)
	for dy := 0; dy < h; dy++ {
		for dx := 0; dx < w; dx++ {
			sx := dx * sw / w
			sy := dy * sh / h
			r, g, b, a := src.At(sb.Min.X+sx, sb.Min.Y+sy).RGBA()
			if a < 0x8000 {
				gray[dy*w+dx] = 1.0 // transparent → white (background)
			} else {
				lum := 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)
				gray[dy*w+dx] = lum / 65535.0
			}
		}
	}

	// Floyd-Steinberg dithering.
	for dy := 0; dy < h; dy++ {
		for dx := 0; dx < w; dx++ {
			old := gray[dy*w+dx]
			var newVal float64
			if old < 0.5 {
				newVal = 0.0
			} else {
				newVal = 1.0
			}
			gray[dy*w+dx] = newVal
			e := old - newVal
			if dx+1 < w {
				gray[dy*w+dx+1] += e * 7.0 / 16.0
			}
			if dy+1 < h && dx > 0 {
				gray[(dy+1)*w+dx-1] += e * 3.0 / 16.0
			}
			if dy+1 < h {
				gray[(dy+1)*w+dx] += e * 5.0 / 16.0
			}
			if dy+1 < h && dx+1 < w {
				gray[(dy+1)*w+dx+1] += e * 1.0 / 16.0
			}
		}
	}

	// Pack to MSB-first 1-bit: dark pixel (< 0.5) → bit SET (ON = bright).
	rowBytes := (w + 7) / 8
	bitmap = make([]byte, h*rowBytes)
	for dy := 0; dy < h; dy++ {
		for dx := 0; dx < w; dx++ {
			if gray[dy*w+dx] < 0.5 {
				bitmap[dy*rowBytes+dx/8] |= 0x80 >> uint(dx%8)
			}
		}
	}
	return
}

func pokemonMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}
