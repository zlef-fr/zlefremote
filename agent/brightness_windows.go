//go:build windows

package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Windows backlight via WMI (root/WMI WmiMonitorBrightness*), driven through
// Windows PowerShell (present on every supported Windows). Works on laptops
// and monitors whose driver exposes a backlight; typical external desktop
// monitors do not — the probe then fails and the phone hides the slider.
//
// Per-screen: both WMI classes are enumerated Sort-Object InstanceName so the
// index the phone sends targets the same physical monitor in Get and Set.
func newBrightener() Brightener {
	b := wmiBright{}
	if s := b.Screens(); len(s) == 0 || s[0].Pct < 0 {
		return noBright{}
	}
	return b
}

const brightTimeout = 5 * time.Second // powershell startup is slow

type wmiBright struct{}

func (wmiBright) Available() bool { return true }

func (wmiBright) Screens() []BrightScreen {
	out, err := runOut(brightTimeout, "powershell", "-NoProfile", "-NonInteractive", "-Command",
		"Get-WmiObject -Namespace root/WMI -Class WmiMonitorBrightness | Sort-Object InstanceName | ForEach-Object { \"$($_.InstanceName)|$($_.CurrentBrightness)\" }")
	if err != nil || out == "" {
		return nil
	}
	var screens []BrightScreen
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		i := strings.LastIndex(line, "|")
		if i < 0 {
			continue
		}
		pct, err := strconv.Atoi(line[i+1:])
		if err != nil {
			pct = -1
		}
		screens = append(screens, BrightScreen{Name: wmiMonName(line[:i]), Pct: pct})
	}
	return screens
}

// wmiMonName shortens "DISPLAY\LEN40A9\4&…" to the monitor id ("LEN40A9").
func wmiMonName(instance string) string {
	if f := strings.Split(instance, `\`); len(f) >= 2 && f[1] != "" {
		return f[1]
	}
	return ""
}

func (wmiBright) Set(display, pct int) {
	var cmd string
	if display < 0 {
		cmd = fmt.Sprintf(
			"Get-WmiObject -Namespace root/WMI -Class WmiMonitorBrightnessMethods | ForEach-Object { $_.WmiSetBrightness(1,%d) }", pct)
	} else {
		cmd = fmt.Sprintf(
			"$m = @(Get-WmiObject -Namespace root/WMI -Class WmiMonitorBrightnessMethods | Sort-Object InstanceName); if ($m.Count -gt %d) { $m[%d].WmiSetBrightness(1,%d) }",
			display, display, pct)
	}
	runOut(brightTimeout, "powershell", "-NoProfile", "-NonInteractive", "-Command", cmd)
}
