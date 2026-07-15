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

// Linux brightness backends, probed in order at startup:
//  1. brightnessctl — laptops; handles permissions via logind, works on
//     X11 and Wayland.
//  2. sysfs         — direct /sys/class/backlight write (root / video group).
//  3. xrandr        — software gamma on every connected output; the only
//     option for desktops with external monitors (X11 only).
//
// Each backend is per-display: it enumerates its devices/outputs so the phone
// can target one screen, and Set(-1, pct) fans out to all of them.
func newBrightener() Brightener {
	if b := probeBrightnessctl(); b != nil {
		return b
	}
	if b := probeSysfs(); b != nil {
		return b
	}
	if b := probeXrandr(); b != nil {
		return b
	}
	return noBright{}
}

const brightTimeout = 3 * time.Second

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
