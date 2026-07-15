//go:build darwin

package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// macOS backlight via the `brightness` CLI (brew install brightness). There
// is no built-in scriptable absolute-brightness tool on macOS, so without it
// the probe fails and the phone hides the slider.
//
// Per-screen: `brightness -l` lists every display in index order and
// `brightness -d N` targets one; names are left empty so the phone numbers
// them itself.
func newBrightener() Brightener {
	b := macBright{}
	if s := b.Screens(); len(s) == 0 || s[0].Pct < 0 {
		return noBright{}
	}
	return b
}

const brightTimeout = 3 * time.Second

type macBright struct{}

func (macBright) Available() bool { return true }

func (macBright) Screens() []BrightScreen {
	// `brightness -l` prints e.g. "display 0: brightness 0.500000" per display
	out, err := runOut(brightTimeout, "brightness", "-l")
	if err != nil {
		return nil
	}
	var screens []BrightScreen
	for _, line := range strings.Split(out, "\n") {
		if i := strings.LastIndex(line, "brightness "); i >= 0 {
			pct := -1
			if v, err := strconv.ParseFloat(strings.TrimSpace(line[i+len("brightness "):]), 64); err == nil {
				pct = int(v*100 + 0.5)
			}
			screens = append(screens, BrightScreen{Pct: pct})
		}
	}
	return screens
}

func (macBright) Set(display, pct int) {
	v := fmt.Sprintf("%.2f", float64(pct)/100)
	if display < 0 {
		runOut(brightTimeout, "brightness", v)
		return
	}
	runOut(brightTimeout, "brightness", "-d", strconv.Itoa(display), v)
}
