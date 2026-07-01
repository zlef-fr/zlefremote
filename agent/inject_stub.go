//go:build !robotgo

package main

import (
	"log"
	"os"
	"runtime"
)

// Stub injector: logs actions instead of driving the OS. The agent compiles and
// the relay/E2EE/QR/handshake all work; only the actual mouse/keyboard movement
// is inert. Build with `-tags robotgo` for the real backend.
type stubInjector struct{}

func newInjector() Injector {
	log.Println("[inject] STUB backend (no OS control) — rebuild with -tags robotgo for real input")
	return stubInjector{}
}

func (stubInjector) MoveRel(dx, dy int)             { log.Printf("[stub] move %+d,%+d", dx, dy) }
func (stubInjector) MoveAbs(x, y int)               { log.Printf("[stub] moveabs %d,%d", x, y) }
func (stubInjector) Click(b string, dbl bool)       { log.Printf("[stub] click %s dbl=%v", b, dbl) }
func (stubInjector) Toggle(b string, down bool)     { log.Printf("[stub] toggle %s down=%v", b, down) }
func (stubInjector) Scroll(dx, dy int)              { log.Printf("[stub] scroll %+d,%+d", dx, dy) }
func (stubInjector) KeyTap(k string, mods []string) { log.Printf("[stub] key %q mods=%v", k, mods) }
func (stubInjector) TypeStr(s string)               { log.Printf("[stub] type %q", s) }
func (stubInjector) Media(k string)                 { log.Printf("[stub] media %s", k) }
func (stubInjector) ScreenSize() (int, int)         { return 1920, 1080 }
func (stubInjector) HostInfo() (string, string) {
	h, _ := os.Hostname()
	return h, runtime.GOOS
}
