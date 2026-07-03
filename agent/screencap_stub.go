//go:build !robotgo

package main

import "errors"

// Stub screen backend: reports the live-view feature as unavailable so the
// phone hides the Screen tab. The relay/handshake still work; only actual
// screen capture is inert. Build with -tags robotgo for the real backend.
type stubScreen struct{}

func newScreener() Screener { return stubScreen{} }

func (stubScreen) Available() bool { return false }

func (stubScreen) Displays() []DisplayInfo { return nil }

func (stubScreen) Capture(display, scalePct, quality int) ([]byte, int, int, error) {
	return nil, 0, 0, errors.New("screen capture not supported in this build")
}
