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
func newBrightener() Brightener {
	b := macBright{}
	if _, ok := b.Get(); !ok {
		return noBright{}
	}
	return b
}

const brightTimeout = 3 * time.Second

type macBright struct{}

func (macBright) Available() bool { return true }

func (macBright) Get() (int, bool) {
	// `brightness -l` prints e.g. "display 0: brightness 0.500000"
	out, err := runOut(brightTimeout, "brightness", "-l")
	if err != nil {
		return 0, false
	}
	for _, line := range strings.Split(out, "\n") {
		if i := strings.LastIndex(line, "brightness "); i >= 0 {
			if v, err := strconv.ParseFloat(strings.TrimSpace(line[i+len("brightness "):]), 64); err == nil {
				return int(v*100 + 0.5), true
			}
		}
	}
	return 0, false
}

func (macBright) Set(pct int) {
	runOut(brightTimeout, "brightness", fmt.Sprintf("%.2f", float64(pct)/100))
}
