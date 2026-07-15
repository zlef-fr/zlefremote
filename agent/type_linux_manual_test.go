//go:build robotgo && linux

package main

import (
	"os"
	"testing"
)

// Manual/e2e driver for typeText: needs a live X display (DISPLAY) with a
// focused key-capture client. Skipped unless ZR_TYPE is set — see the
// stale-keymap regression harness in the repo history (Xvfb + setxkbmap fr +
// a capture client that ignores MappingNotify, like debounced GTK/Qt do).
func TestTypeTextManual(t *testing.T) {
	s := os.Getenv("ZR_TYPE")
	if s == "" {
		t.Skip("set ZR_TYPE (and DISPLAY) to run")
	}
	typeText(s)
}
