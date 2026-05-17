// hackintosh host application.
//
// Modes:
//
//	(default)          Run the full app: data sources + render loop + serial to MCU.
//	--simulate=:8080   Run the full app against a browser-based virtual OLED.
//	                   No hardware needed. Open http://localhost:8080 to view.
//	--probe            Connect to the MCU, exchange PING/PONG, dump button events.
//	--list-ports       Print available serial ports and exit.
package main

import (
	"context"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/datcal/hackintosh/host/internal/app"
	"github.com/datcal/hackintosh/host/internal/device"
	"github.com/datcal/hackintosh/host/internal/tea"
	"github.com/datcal/hackintosh/host/internal/render"
	"github.com/datcal/hackintosh/host/internal/render/screens"
	"github.com/datcal/hackintosh/host/internal/sources"
	"github.com/datcal/hackintosh/host/internal/sources/media"
	"github.com/datcal/hackintosh/host/internal/store"
	"github.com/datcal/hackintosh/host/internal/transport"
	"github.com/datcal/hackintosh/host/internal/tray"
)

var version = "dev"

func main() {
	var (
		port      = flag.String("port", "", "serial port for the MCU (auto-pick if empty)")
		simulate  = flag.String("simulate", "", "if set (e.g. :8080), runs without hardware and serves a virtual OLED at this address")
		probe     = flag.Bool("probe", false, "probe mode: PING the MCU and dump events")
		listPorts = flag.Bool("list-ports", false, "list serial ports and exit")
		noNet     = flag.Bool("no-net", false, "disable network-backed data sources (weather, currency)")
		noHW      = flag.Bool("no-hw", false, "disable hardware monitor source")
		noMedia   = flag.Bool("no-media", false, "disable OS now-playing source")
		snapshot  = flag.String("snapshot", "", "render one PNG per screen into the given directory and exit")
		demoData  = flag.Bool("demo-data", false, "populate the store with sample data when running --snapshot")
		scale     = flag.Int("scale", 4, "pixel scale for snapshot PNGs")
		flash     = flag.String("flash", "", "wait for the RPI-RP2 bootloader drive to appear, then copy the given .uf2 onto it. Hold A+B on the device for 3 sec to trigger bootloader mode.")
		flashTimeout = flag.Int("flash-timeout", 60, "seconds to wait for the RPI-RP2 drive when using --flash")
		showVer   = flag.Bool("version", false, "print version and exit")
	)
	flag.Parse()

	if *showVer {
		fmt.Println("hackintosh", version)
		return
	}
	if *listPorts {
		listAvailablePorts()
		return
	}
	if *probe {
		runProbe(*port)
		return
	}
	if *snapshot != "" {
		runSnapshot(*snapshot, *scale, *demoData)
		return
	}
	if *flash != "" {
		runFlash(*flash, *flashTimeout)
		return
	}
	if *simulate == "" {
		runTrayApp(*simulate, *port, *noNet, *noHW, *noMedia)
	} else {
		runApp(*simulate, *port, *noNet, *noHW, *noMedia)
	}
}

// runFlash waits for the RPI-RP2 mass-storage drive to appear (the bootloader's
// virtual disk) and copies the given .uf2 onto it. The user triggers bootloader
// mode by holding A+B on the device for 3 seconds — no physical RESET/BOOT
// button presses required.
func runFlash(uf2Path string, timeoutSec int) {
	if !strings.HasSuffix(strings.ToLower(uf2Path), ".uf2") {
		log.Fatalf("flash: file must have a .uf2 extension: %s", uf2Path)
	}

	resolved, src, err := readUF2(uf2Path)
	if err != nil {
		log.Fatalf("flash: %v", err)
	}
	uf2Path = resolved

	if len(src) < 512 {
		log.Fatalf("flash: %s looks too small to be a valid .uf2 (%d bytes)", uf2Path, len(src))
	}

	log.Printf("flash: waiting for RPI-RP2 drive — hold A+B on the device for 3 seconds...")

	var drive string
	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	for time.Now().Before(deadline) {
		if d, ok := findRpiRP2Drive(); ok {
			drive = d
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if drive == "" {
		log.Fatalf("flash: timed out after %ds waiting for RPI-RP2 drive", timeoutSec)
	}

	log.Printf("flash: found %s, copying %s (%d bytes)...", drive, uf2Path, len(src))
	dst := filepath.Join(drive, filepath.Base(uf2Path))
	if err := os.WriteFile(dst, src, 0o644); err != nil {
		log.Fatalf("flash: write %s: %v", dst, err)
	}
	log.Printf("flash: done — the board will reboot into the new firmware momentarily.")
}

// readUF2 reads the .uf2 file, trying the given path as-is first, then falling
// back to the parent directory. The fallback exists because `dev-host.ps1`
// `cd`s into `host/` before invoking `go run`, but users naturally pass paths
// relative to the project root ("dist\hackintosh.uf2"). Returns the path
// that actually worked, the file bytes, and any error.
func readUF2(uf2Path string) (string, []byte, error) {
	// Attempt 1: the path the user gave us.
	if data, err := os.ReadFile(uf2Path); err == nil {
		return uf2Path, data, nil
	}

	// Attempt 2: the path interpreted as "one directory up." Skip for
	// absolute paths since adding ".." is meaningless there.
	if !filepath.IsAbs(uf2Path) {
		alt := filepath.Join("..", uf2Path)
		if data, err := os.ReadFile(alt); err == nil {
			return alt, data, nil
		}
	}

	// Couldn't find it either way — report BOTH paths we tried so the user
	// can see what went wrong.
	abs1, _ := filepath.Abs(uf2Path)
	tried := []string{abs1}
	if !filepath.IsAbs(uf2Path) {
		abs2, _ := filepath.Abs(filepath.Join("..", uf2Path))
		tried = append(tried, abs2)
	}
	return uf2Path, nil, fmt.Errorf("cannot read .uf2 file. Tried:\n  - %s", strings.Join(tried, "\n  - "))
}

// findRpiRP2Drive scans the filesystem for the RP2040 bootloader's virtual
// mass-storage drive. We probe for INFO_UF2.TXT (which the bootloader always
// writes to its root) instead of checking volume labels — that's portable
// across Windows / macOS / Linux without OS-specific APIs.
func findRpiRP2Drive() (string, bool) {
	var candidates []string
	if runtime.GOOS == "windows" {
		for c := 'A'; c <= 'Z'; c++ {
			candidates = append(candidates, string(c)+":\\")
		}
	} else {
		// macOS auto-mounts USB drives under /Volumes; common Linux mount
		// points are /media/<user>/ and /mnt.
		candidates = []string{
			"/Volumes/RPI-RP2",
			"/media/" + os.Getenv("USER") + "/RPI-RP2",
			"/mnt/RPI-RP2",
			"/run/media/" + os.Getenv("USER") + "/RPI-RP2",
		}
	}
	for _, path := range candidates {
		marker := filepath.Join(path, "INFO_UF2.TXT")
		if info, err := os.Stat(marker); err == nil && !info.IsDir() {
			return path, true
		}
	}
	return "", false
}

// runSnapshot renders one PNG per screen to the given directory. Useful for
// eyeballing layout fixes without running the simulator.
func runSnapshot(dir string, scale int, demoData bool) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Fatalf("mkdir %s: %v", dir, err)
	}
	pomo := tea.New()
	st := store.New()
	if demoData {
		populateDemoData(st)
	}
	chrome := render.NewChromeState()

	// Warm up the animation state so springs settle and ambient motion is
	// mid-cycle (rather than at t=0 where everything is identical).
	dt := 1.0 / 30
	for i := 0; i < 60; i++ {
		chrome.Tick(dt)
	}

	all := screens.All()
	now := time.Now()
	for _, scr := range all {
		// Pass 1: render once to seed each screen's spring Targets from the
		// store snapshot. Output is discarded.
		seed := render.New()
		scr.Render(seed, st.Snapshot(), now)
		// Now tick for ~2 seconds (60 frames) so the springs settle near
		// their targets — otherwise temperature reads "0C" while spring
		// glides up to 14.
		for i := 0; i < 120; i++ {
			scr.Tick(dt)
			// Re-render so any "read store, set Target" logic stays current.
			scratch := render.New()
			scr.Render(scratch, st.Snapshot(), now)
		}

		c := render.New()
		render.DrawChrome(c, chrome, scr.Name(), pomo.Snapshot())
		scr.Render(c, st.Snapshot(), now)
		render.DrawTimerStrip(c, pomo.Snapshot(), 0)

		path := strings.ToLower(scr.Name()) + ".png"
		fullPath := dir + string(os.PathSeparator) + path
		if err := writeFrameAsPNG(fullPath, c.Bytes(), scale); err != nil {
			log.Fatalf("write %s: %v", fullPath, err)
		}
		fmt.Printf("  wrote %s\n", fullPath)
	}
}

// populateDemoData fills the store with believable sample values so the
// snapshot screens show real-looking data instead of zeros.
func populateDemoData(st *store.Store) {
	now := time.Now()
	st.SetWeather(store.Weather{
		Valid: true, TempC: 14, FeelsLikeC: 12, WindKMH: 8,
		Condition: store.CondSunny, LocationName: "Istanbul", UpdatedAt: now,
	})
	st.SetAirQuality(store.AirQuality{
		Valid: true, AQI: 3, PM25: 18.4, PM10: 32.1, UpdatedAt: now,
	})
	spark := []float64{44.10, 44.12, 44.18, 44.15, 44.20, 44.21, 44.19, 44.24, 44.21}
	usdSpark := []float64{39.01, 39.05, 39.03, 39.06, 39.04, 39.07, 39.05}
	st.SetCurrency(store.Currency{
		Valid: true, EURTRY: 44.21, USDTRY: 39.07,
		EURTRYPct: 0.18, USDTRYPct: -0.05,
		SparkEUR: spark, SparkUSD: usdSpark, UpdatedAt: now,
	})
	st.SetHW(store.HW{
		Valid: true, CPUPct: 42, RAMPct: 67, DiskPct: 71,
		NetUpKBs: 22, NetDownKBs: 480,
		Uptime: 38*time.Hour + 14*time.Minute, UpdatedAt: now,
	})
	st.SetMedia(store.Media{
		Valid: true, Title: "Bohemian Rhapsody", Artist: "Queen",
		Playing: true, Position: 2*time.Minute + 14*time.Second, Length: 5*time.Minute + 55*time.Second,
		UpdatedAt: now,
	})
}

// writeFrameAsPNG converts the 1-bit SSD1306 page-major framebuffer to a PNG,
// scaled up so it's actually viewable. Pixels: blue-on-near-black like the
// simulator's OLED preview.
func writeFrameAsPNG(path string, frame []byte, scale int) error {
	if scale < 1 { scale = 1 }
	const W, H = render.Width, render.Height
	on  := color.RGBA{0x74, 0xDC, 0xFF, 0xFF}
	off := color.RGBA{0x07, 0x15, 0x1B, 0xFF}
	img := image.NewRGBA(image.Rect(0, 0, W*scale, H*scale))
	for page := 0; page < H/8; page++ {
		for col := 0; col < W; col++ {
			b := frame[page*W+col]
			for bit := 0; bit < 8; bit++ {
				y := page*8 + bit
				on1 := b&(1<<uint(bit)) != 0
				c := off
				if on1 {
					c = on
				}
				for dy := 0; dy < scale; dy++ {
					for dx := 0; dx < scale; dx++ {
						img.SetRGBA(col*scale+dx, y*scale+dy, c)
					}
				}
			}
		}
	}
	f, err := os.Create(path)
	if err != nil { return err }
	defer f.Close()
	return png.Encode(f, img)
}

func runApp(simulateAddr, portName string, noNet, noHW, noMedia bool) {
	s := &appSession{}
	if err := s.run(func(ctx context.Context) error {
		return runAppSession(ctx, s, simulateAddr, portName, noNet, noHW, noMedia)
	}); err != nil && err != context.Canceled {
		log.Printf("app loop ended: %v", err)
	}
}

// runAppSession is the actual work -- broken out so tray-mode can call it
// repeatedly under fresh contexts on Restart.
func runAppSession(ctx context.Context, s *appSession, simulateAddr, portName string, noNet, noHW, noMedia bool) error {
	// --- 1. Pick a device backend ---
	var dev device.Device
	var err error
	if simulateAddr != "" {
		dev, err = device.NewSimulator(simulateAddr)
		if err != nil {
			return fmt.Errorf("simulator: %w", err)
		}
		log.Printf("simulating without hardware -- open http://%s in a browser",
			normalizeAddr(simulateAddr))
		s.setSimulatorURL("http://" + normalizeAddr(simulateAddr))
	} else {
		if portName == "" {
			guessed, gerr := guessPort()
			if gerr != nil {
				return fmt.Errorf("no serial port found (pass --port=COM5 or use --simulate=:8080): %w", gerr)
			}
			portName = guessed
			log.Printf("auto-picked serial port: %s", portName)
		}
		dev, err = device.OpenSerial(ctx, portName)
		if err != nil {
			return fmt.Errorf("open serial %s: %w", portName, err)
		}
	}
	defer dev.Close()

	// --- 2. Data sources ---
	st := store.New()
	if !noNet {
		go (&sources.WeatherWorker{S: st}).Run(ctx)
		go (&sources.CurrencyWorker{S: st}).Run(ctx)
	}
	if !noHW {
		go (&sources.HWWorker{S: st}).Run(ctx)
	}
	if !noMedia {
		go (&media.Worker{S: st}).Run(ctx)
	}

	// --- 3. Pomodoro + app loop ---
	pomo := tea.New()
	a := app.New(dev, st, pomo)
	return a.Run(ctx)
}

// runTrayApp is the default mode when the binary is launched with no flags
// (typically by autostart or by the user double-clicking the launcher
// shortcut). It runs the systray on the main goroutine and the app session
// on a child goroutine.
func runTrayApp(simulateAddr, portName string, noNet, noHW, noMedia bool) {
	s := &appSession{}
	controller := newHostController(s)

	go func() {
		// Loop so that if the session exits cleanly (e.g., serial disconnect
		// after the device was unplugged), the tray stays alive. Restart from
		// the menu re-execs the whole binary so it's not handled here.
		for {
			err := s.run(func(ctx context.Context) error {
				return runAppSession(ctx, s, simulateAddr, portName, noNet, noHW, noMedia)
			})
			if err != nil && err != context.Canceled {
				log.Printf("app loop ended: %v", err)
			}
			// If quit was triggered, break out so we stop relooping.
			if s.getQuit() {
				return
			}
			// Otherwise wait a moment and try again -- this handles the
			// "device was unplugged, will it come back?" case.
			time.Sleep(2 * time.Second)
		}
	}()

	tray.Run(controller)
}

func normalizeAddr(addr string) string {
	if strings.HasPrefix(addr, ":") {
		return "localhost" + addr
	}
	return addr
}

// --- legacy probe & helpers ---

func listAvailablePorts() {
	ports, err := transport.List()
	if err != nil { log.Fatal(err) }
	if len(ports) == 0 { fmt.Println("(no serial ports found)"); return }
	for _, p := range ports { fmt.Println(p) }
}

func guessPort() (string, error) {
	ports, err := transport.List()
	if err != nil { return "", err }
	for _, p := range ports {
		lo := strings.ToLower(p)
		if strings.Contains(lo, "usbmodem") || strings.Contains(lo, "ttyacm") {
			return p, nil
		}
	}
	if len(ports) > 0 { return ports[0], nil }
	return "", fmt.Errorf("no serial ports found")
}

func runProbe(portName string) {
	if portName == "" {
		guessed, err := guessPort()
		if err != nil { log.Fatal(err) }
		portName = guessed
		log.Printf("auto-picked port: %s", portName)
	}
	cli, err := transport.Open(portName)
	if err != nil { log.Fatalf("open port %s: %v", portName, err) }
	defer cli.Close()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	go func() {
		if err := cli.Run(ctx); err != nil && err != context.Canceled {
			log.Printf("read loop ended: %v", err)
		}
	}()
	pingTicker := time.NewTicker(2 * time.Second)
	defer pingTicker.Stop()
	pingSentAt := time.Time{}
	log.Println("probe: waiting for events. Ctrl-C to exit.")
	if err := cli.Ping(); err != nil { log.Printf("initial PING: %v", err) } else { pingSentAt = time.Now() }
	for {
		select {
		case <-ctx.Done(): return
		case <-pingTicker.C:
			if err := cli.Ping(); err != nil { log.Printf("PING: %v", err); continue }
			pingSentAt = time.Now()
		case ev, ok := <-cli.Events():
			if !ok { return }
			describeEvent(ev, pingSentAt)
		}
	}
}

func describeEvent(ev transport.Event, pingSentAt time.Time) {
	switch ev.Type {
	case transport.TypePong:
		if !pingSentAt.IsZero() {
			log.Printf("PONG  (RTT %s)", time.Since(pingSentAt).Round(time.Microsecond))
		} else { log.Println("PONG (unsolicited)") }
	case transport.TypeButton:
		if len(ev.Payload) != 2 { log.Printf("BUTTON malformed: %v", ev.Payload); return }
		log.Printf("BUTTON %s %s", btnIDName(ev.Payload[0]), btnEventName(ev.Payload[1]))
	case transport.TypeLog:
		log.Printf("MCU LOG: %s", string(ev.Payload))
	default:
		log.Printf("unknown event type=0x%02X len=%d", ev.Type, len(ev.Payload))
	}
}

func btnIDName(id byte) string {
	switch id {
	case transport.ButtonIDA: return "A"
	case transport.ButtonIDB: return "B"
	}
	return fmt.Sprintf("?%d", id)
}

func btnEventName(e byte) string {
	switch e {
	case transport.ButtonPress: return "press"
	case transport.ButtonRelease: return "release"
	case transport.ButtonLongPress: return "long-press"
	}
	return fmt.Sprintf("evt?%d", e)
}
