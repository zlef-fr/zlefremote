//go:build linux

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Linux brightness backends, probed in order (best rendering first):
//  1. brightnessctl — laptops; real backlight, permissions via logind, works
//     on X11 and Wayland.
//  2. sysfs         — direct /sys/class/backlight write (root / video group);
//     also a real backlight.
//  3. xrandr        — software gamma on every connected output; washed-out
//     dimming, but the only option that reaches external monitors (X11 only).
//
// Unlike a laptop's single panel, a desktop often has several of these usable
// at once (a real backlight AND xrandr), so rather than silently locking onto
// the first hit we detect them all: newBrightener picks the best as default but
// wraps them in a switchBright so the phone can offer "dim via X or Y". Each
// backend is per-display — it enumerates its devices/outputs so the phone can
// target one screen, and Set(-1, pct) fans out to all of them.
func newBrightener() Brightener {
	// probes in priority order; each yields a live backend or nil when absent
	probes := []struct {
		id, label, kind string
		probe           func() Brightener
	}{
		{"brightnessctl", "Backlight (brightnessctl)", "hardware", probeBrightnessctl},
		{"sysfs", "Backlight (sysfs)", "hardware", probeSysfs},
		// external monitors over the video cable — the only mechanism that
		// reaches a desktop's screens at all
		{"ddcutil", "External monitors (DDC/CI)", "hardware", probeDDCUtil},
		{"xrandr", "Software dimming (xrandr)", "software", probeXrandr},
	}
	var avail []BrightBackend
	byID := map[string]Brightener{}
	for _, p := range probes {
		if b := p.probe(); b != nil {
			avail = append(avail, BrightBackend{ID: p.id, Label: p.label, Kind: p.kind})
			byID[p.id] = b
		}
	}
	if len(avail) == 0 {
		return noBright{}
	}
	// default = best-rendering available, unless the user pins one via env
	active := pickActiveBackend(avail, os.Getenv("ZLEFREMOTE_BRIGHTNESS_BACKEND"))
	return &switchBright{avail: avail, impl: byID, active: active}
}

// pickActiveBackend chooses the default backend index: the one whose ID matches
// want (ZLEFREMOTE_BRIGHTNESS_BACKEND), else 0 — the highest-priority available
// backend. An empty or unknown want falls back to the best default.
func pickActiveBackend(avail []BrightBackend, want string) int {
	if want != "" {
		for i, be := range avail {
			if be.ID == want {
				return i
			}
		}
	}
	return 0
}

const brightTimeout = 3 * time.Second

// ── switchBright: multi-backend holder ─────────────────────────────────────
//
// Delegates the Brightener interface to the currently-selected backend and
// implements BackendChooser so the phone can switch at runtime. The active
// index is guarded because Set runs on the brightness worker goroutine while
// Select/Screens are driven from the session read loop.
type switchBright struct {
	mu     sync.RWMutex
	avail  []BrightBackend       // metadata, priority order (stable, read-only)
	impl   map[string]Brightener // id → live backend
	active int                   // index into avail
}

func (s *switchBright) current() Brightener {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.impl[s.avail[s.active].ID]
}

func (s *switchBright) Available() bool           { return true }
func (s *switchBright) Screens() []BrightScreen   { return s.current().Screens() }
func (s *switchBright) Set(display, pct int)      { s.current().Set(display, pct) }
func (s *switchBright) Backends() []BrightBackend { return s.avail }

func (s *switchBright) Active() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.avail[s.active].ID
}

func (s *switchBright) Select(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, be := range s.avail {
		if be.ID == id {
			s.active = i
			return true
		}
	}
	return false
}

// ── brightnessctl ──────────────────────────────────────────────────────────

type ctlBright struct {
	devs []string // backlight device names, one per screen
}

// probeBrightnessctl accepts only if a backlight-class device exists (the
// tool also lists keyboard LEDs, which are not the screen).
func probeBrightnessctl() Brightener {
	// machine-readable list: device,class,current,percent%,max per line
	out, err := runOut(brightTimeout, "brightnessctl", "-m", "-l", "-c", "backlight")
	if err != nil || out == "" {
		return nil
	}
	b := &ctlBright{}
	for _, line := range strings.Split(out, "\n") {
		if f := strings.Split(line, ","); len(f) >= 5 && f[1] == "backlight" && f[0] != "" {
			b.devs = append(b.devs, f[0])
		}
	}
	if len(b.devs) == 0 {
		return nil
	}
	if b.Screens()[0].Pct < 0 {
		return nil
	}
	return b
}

func (b *ctlBright) Available() bool { return true }

func (b *ctlBright) Screens() []BrightScreen {
	out := make([]BrightScreen, len(b.devs))
	for i, d := range b.devs {
		out[i] = BrightScreen{Name: d, Pct: ctlGet(d)}
	}
	return out
}

// ctlGet reads one device's current percent (-1 on failure).
func ctlGet(dev string) int {
	// machine-readable: device,class,current,percent%,max
	out, err := runOut(brightTimeout, "brightnessctl", "-m", "-d", dev, "-c", "backlight")
	if err != nil {
		return -1
	}
	f := strings.Split(strings.SplitN(out, "\n", 2)[0], ",")
	if len(f) < 4 {
		return -1
	}
	pct, err := strconv.Atoi(strings.TrimSuffix(f[3], "%"))
	if err != nil {
		return -1
	}
	return pct
}

func (b *ctlBright) Set(display, pct int) {
	for i, d := range b.devs {
		if display >= 0 && i != display {
			continue
		}
		runOut(brightTimeout, "brightnessctl", "-q", "-d", d, "-c", "backlight", "set", fmt.Sprintf("%d%%", pct))
	}
}

// ── sysfs ──────────────────────────────────────────────────────────────────

type sysfsDev struct {
	dir string
	max int
}

type sysfsBright struct {
	devs []sysfsDev
}

func probeSysfs() Brightener {
	dirs, _ := filepath.Glob("/sys/class/backlight/*")
	b := &sysfsBright{}
	for _, d := range dirs {
		max := readSysInt(filepath.Join(d, "max_brightness"))
		if max <= 0 {
			continue
		}
		// must actually be writable for Set to work
		f, err := os.OpenFile(filepath.Join(d, "brightness"), os.O_WRONLY, 0)
		if err != nil {
			continue
		}
		f.Close()
		b.devs = append(b.devs, sysfsDev{dir: d, max: max})
	}
	if len(b.devs) == 0 {
		return nil
	}
	return b
}

func readSysInt(path string) int {
	b, err := os.ReadFile(path)
	if err != nil {
		return -1
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil {
		return -1
	}
	return n
}

func (s *sysfsBright) Available() bool { return true }

func (s *sysfsBright) Screens() []BrightScreen {
	out := make([]BrightScreen, len(s.devs))
	for i, d := range s.devs {
		pct := -1
		if cur := readSysInt(filepath.Join(d.dir, "brightness")); cur >= 0 {
			pct = (cur*100 + d.max/2) / d.max
		}
		out[i] = BrightScreen{Name: filepath.Base(d.dir), Pct: pct}
	}
	return out
}

func (s *sysfsBright) Set(display, pct int) {
	for i, d := range s.devs {
		if display >= 0 && i != display {
			continue
		}
		raw := (pct*d.max + 50) / 100
		if raw < 1 {
			raw = 1
		}
		os.WriteFile(filepath.Join(d.dir, "brightness"), []byte(strconv.Itoa(raw)), 0)
	}
}

// ── xrandr (software gamma) ────────────────────────────────────────────────

type xrandrBright struct {
	mu      sync.Mutex
	outputs []string
	last    []int // xrandr has no cheap "get", so remember what we set (per output)
}

func probeXrandr() Brightener {
	if os.Getenv("DISPLAY") == "" {
		return nil
	}
	out, err := runOut(brightTimeout, "xrandr", "--verbose")
	if err != nil {
		return nil
	}
	x := &xrandrBright{}
	for _, line := range strings.Split(out, "\n") {
		if f := strings.Fields(line); len(f) >= 2 && f[1] == "connected" {
			x.outputs = append(x.outputs, f[0])
			x.last = append(x.last, -1)
		} else if n := len(x.outputs); n > 0 && x.last[n-1] < 0 && strings.HasPrefix(strings.TrimSpace(line), "Brightness:") {
			// current software gamma of the output being described
			if v, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "Brightness:")), 64); err == nil {
				x.last[n-1] = int(v*100 + 0.5)
			}
		}
	}
	if len(x.outputs) == 0 {
		return nil
	}
	return x
}

func (x *xrandrBright) Available() bool { return true }

func (x *xrandrBright) Screens() []BrightScreen {
	x.mu.Lock()
	defer x.mu.Unlock()
	out := make([]BrightScreen, len(x.outputs))
	for i, o := range x.outputs {
		out[i] = BrightScreen{Name: o, Pct: x.last[i]}
	}
	return out
}

func (x *xrandrBright) Set(display, pct int) {
	v := fmt.Sprintf("%.2f", float64(pct)/100)
	x.mu.Lock()
	var targets []string
	for i, o := range x.outputs {
		if display >= 0 && i != display {
			continue
		}
		targets = append(targets, o)
		x.last[i] = pct
	}
	x.mu.Unlock()
	for _, o := range targets {
		runOut(brightTimeout, "xrandr", "--output", o, "--brightness", v)
	}
}

// ── DDC/CI (external monitors) ──────────────────────────────────────────────
//
// Backlight devices and xrandr gamma between them still miss the common case:
// a desktop with monitors on a cable. Those answer DDC/CI, and ddcutil speaks
// it. Reads are slow (an I²C round trip per monitor, ~200 ms), so the display
// list is probed once at startup and only levels are re-read.
//
// ddcutil needs access to the i2c devices: on most distributions that means the
// i2c-dev module loaded and the user in the `i2c` group. When it isn't set up
// `detect` returns nothing and this backend simply doesn't appear.

type ddcUtilBright struct{ buses []ddcBus }

type ddcBus struct {
	bus  string
	name string
}

func probeDDCUtil() Brightener {
	if !have("ddcutil") {
		return nil
	}
	out, err := runOut(8*time.Second, "ddcutil", "detect", "--brief")
	if err != nil || out == "" {
		return nil
	}
	var buses []ddcBus
	var current ddcBus
	for _, line := range strings.Split(out, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "Display "):
			if current.bus != "" {
				buses = append(buses, current)
			}
			current = ddcBus{}
		case strings.HasPrefix(trimmed, "I2C bus:"):
			// "I2C bus: /dev/i2c-5" → the number is what --bus wants
			if i := strings.LastIndex(trimmed, "-"); i >= 0 {
				current.bus = strings.TrimSpace(trimmed[i+1:])
			}
		case strings.HasPrefix(trimmed, "Monitor:"):
			// "Monitor: DEL:DELL U2515H:..." → keep the model
			fields := strings.Split(strings.TrimPrefix(trimmed, "Monitor:"), ":")
			if len(fields) >= 2 {
				current.name = strings.TrimSpace(fields[1])
			}
		}
	}
	if current.bus != "" {
		buses = append(buses, current)
	}
	if len(buses) == 0 {
		return nil
	}
	return ddcUtilBright{buses: buses}
}

func (ddcUtilBright) Available() bool { return true }

func (d ddcUtilBright) Screens() []BrightScreen {
	screens := make([]BrightScreen, 0, len(d.buses))
	for i, b := range d.buses {
		name := b.name
		if name == "" {
			name = fmt.Sprintf("Monitor %d", i+1)
		}
		screens = append(screens, BrightScreen{Name: name, Pct: d.level(b)})
	}
	return screens
}

// level reads VCP feature 10 (luminance) as a percentage, -1 when the monitor
// won't answer.
func (d ddcUtilBright) level(b ddcBus) int {
	out, err := runOut(6*time.Second, "ddcutil", "--bus", b.bus, "getvcp", "10", "--brief")
	if err != nil {
		return -1
	}
	// brief form: "VCP 10 C <current> <max>"
	fields := strings.Fields(out)
	if len(fields) < 5 {
		return -1
	}
	current, err1 := strconv.Atoi(fields[3])
	max, err2 := strconv.Atoi(fields[4])
	if err1 != nil || err2 != nil || max <= 0 {
		return -1
	}
	return current * 100 / max
}

func (d ddcUtilBright) Set(display, pct int) {
	set := func(b ddcBus) {
		runOut(6*time.Second, "ddcutil", "--bus", b.bus, "setvcp", "10", strconv.Itoa(pct))
	}
	if display < 0 {
		for _, b := range d.buses {
			set(b)
		}
		return
	}
	if display < len(d.buses) {
		set(d.buses[display])
	}
}
