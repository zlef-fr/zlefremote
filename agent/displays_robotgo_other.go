//go:build robotgo && !linux

package main

import "github.com/vcaesar/screenshot"

// displayBounds returns each monitor's rect. On Windows (EnumDisplayMonitors)
// and macOS (CGDisplayBounds) vcaesar/screenshot already reports global
// desktop coordinates — the same space robotgo.Move uses — so no translation
// is needed. Index order matches screenshot.CaptureDisplay.
func displayBounds() []DisplayInfo {
	n := screenshot.NumActiveDisplays()
	var out []DisplayInfo
	for i := 0; i < n; i++ {
		b := screenshot.GetDisplayBounds(i)
		if b.Dx() > 0 && b.Dy() > 0 {
			out = append(out, DisplayInfo{X: b.Min.X, Y: b.Min.Y, W: b.Dx(), H: b.Dy()})
		}
	}
	return out
}
