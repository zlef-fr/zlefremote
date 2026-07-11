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

type ctlBright struct{}

// probeBrightnessctl accepts only if a backlight-class device exists (the
// tool also lists keyboard LEDs, which are not the screen).
func probeBrightnessctl() Brightener {
	out, err := runOut(brightTimeout, "brightnessctl", "-m", "-c", "backlight")
	if err != nil || out == "" {
		return nil
	}
	if _, ok := (ctlBright{}).Get(); !ok {
		return nil
	}
	return ctlBright{}
}

func (ctlBright) Available() bool { return true }

func (ctlBright) Get() (int, bool) {
	// machine-readable: device,class,current,percent%,max
	out, err := runOut(brightTimeout, "brightnessctl", "-m", "-c", "backlight")
	if err != nil {
		return 0, false
	}
	f := strings.Split(strings.SplitN(out, "\n", 2)[0], ",")
	if len(f) < 4 {
		return 0, false
	}
	pct, err := strconv.Atoi(strings.TrimSuffix(f[3], "%"))
	if err != nil {
		return 0, false
	}
	return pct, true
}

func (ctlBright) Set(pct int) {
	runOut(brightTimeout, "brightnessctl", "-q", "-c", "backlight", "set", fmt.Sprintf("%d%%", pct))
}

// ── sysfs ──────────────────────────────────────────────────────────────────

type sysfsBright struct {
	dir string
	max int
}

func probeSysfs() Brightener {
	dirs, _ := filepath.Glob("/sys/class/backlight/*")
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
		return &sysfsBright{dir: d, max: max}
	}
	return nil
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

func (s *sysfsBright) Get() (int, bool) {
	cur := readSysInt(filepath.Join(s.dir, "brightness"))
	if cur < 0 {
		return 0, false
	}
	return (cur*100 + s.max/2) / s.max, true
}

func (s *sysfsBright) Set(pct int) {
	raw := (pct*s.max + 50) / 100
	if raw < 1 {
		raw = 1
	}
	os.WriteFile(filepath.Join(s.dir, "brightness"), []byte(strconv.Itoa(raw)), 0)
}

// ── xrandr (software gamma) ────────────────────────────────────────────────

type xrandrBright struct {
	mu      sync.Mutex
	outputs []string
	last    int // xrandr has no cheap "get", so remember what we set
}

func probeXrandr() Brightener {
	if os.Getenv("DISPLAY") == "" {
		return nil
	}
	out, err := runOut(brightTimeout, "xrandr", "--verbose")
	if err != nil {
		return nil
	}
	x := &xrandrBright{last: -1}
	for _, line := range strings.Split(out, "\n") {
		if f := strings.Fields(line); len(f) >= 2 && f[1] == "connected" {
			x.outputs = append(x.outputs, f[0])
		} else if x.last < 0 && len(x.outputs) > 0 && strings.HasPrefix(strings.TrimSpace(line), "Brightness:") {
			// first connected output's current software gamma
			if v, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "Brightness:")), 64); err == nil {
				x.last = int(v*100 + 0.5)
			}
		}
	}
	if len(x.outputs) == 0 {
		return nil
	}
	return x
}

func (x *xrandrBright) Available() bool { return true }

func (x *xrandrBright) Get() (int, bool) {
	x.mu.Lock()
	defer x.mu.Unlock()
	if x.last < 0 {
		return 0, false
	}
	return x.last, true
}

func (x *xrandrBright) Set(pct int) {
	v := fmt.Sprintf("%.2f", float64(pct)/100)
	x.mu.Lock()
	outputs := append([]string(nil), x.outputs...)
	x.last = pct
	x.mu.Unlock()
	for _, o := range outputs {
		runOut(brightTimeout, "xrandr", "--output", o, "--brightness", v)
	}
}
