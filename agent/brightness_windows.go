//go:build windows

package main

import (
	"fmt"
	"strconv"
	"time"
)

// Windows backlight via WMI (root/WMI WmiMonitorBrightness*), driven through
// Windows PowerShell (present on every supported Windows). Works on laptops
// and monitors whose driver exposes a backlight; typical external desktop
// monitors do not — the probe then fails and the phone hides the slider.
func newBrightener() Brightener {
	b := wmiBright{}
	if _, ok := b.Get(); !ok {
		return noBright{}
	}
	return b
}

const brightTimeout = 5 * time.Second // powershell startup is slow

type wmiBright struct{}

func (wmiBright) Available() bool { return true }

func (wmiBright) Get() (int, bool) {
	out, err := runOut(brightTimeout, "powershell", "-NoProfile", "-NonInteractive", "-Command",
		"(Get-WmiObject -Namespace root/WMI -Class WmiMonitorBrightness | Select-Object -First 1).CurrentBrightness")
	if err != nil || out == "" {
		return 0, false
	}
	pct, err := strconv.Atoi(out)
	if err != nil {
		return 0, false
	}
	return pct, true
}

func (wmiBright) Set(pct int) {
	runOut(brightTimeout, "powershell", "-NoProfile", "-NonInteractive", "-Command",
		fmt.Sprintf("(Get-WmiObject -Namespace root/WMI -Class WmiMonitorBrightnessMethods | Select-Object -First 1).WmiSetBrightness(1,%d)", pct))
}
