//go:build robotgo

package main

import (
	"log"
	"os"
	"runtime"

	"github.com/go-vgo/robotgo"
)

// Real OS input backend. Build with: go build -tags robotgo
// Requires a C toolchain + platform headers (see build.sh / README).
type rgInjector struct{}

func newInjector() Injector {
	log.Println("[inject] robotgo backend (real mouse/keyboard control)")
	return rgInjector{}
}

func btn(b string) string {
	switch b {
	case "right":
		return "right"
	case "middle":
		return "center"
	default:
		return "left"
	}
}

// our key names → robotgo key names
var keyMap = map[string]string{
	"enter": "enter", "backspace": "backspace", "escape": "escape", "tab": "tab",
	"up": "up", "down": "down", "left": "left", "right": "right",
	"delete": "delete", "home": "home", "end": "end",
	"pageup": "pageup", "pagedown": "pagedown", "space": "space",
	"f5": "f5", "f11": "f11",
}

var modMap = map[string]string{
	"ctrl": "ctrl", "alt": "alt", "shift": "shift", "meta": "cmd",
}

var mediaMap = map[string]string{
	"volup": "audio_vol_up", "voldown": "audio_vol_down", "mute": "audio_mute",
	"playpause": "audio_play", "prev": "audio_prev", "next": "audio_next",
}

func (rgInjector) MoveRel(dx, dy int) {
	x, y := robotgo.Location()
	robotgo.Move(x+dx, y+dy)
}

func (rgInjector) MoveAbs(x, y int) { robotgo.Move(x, y) }

func (rgInjector) Click(b string, dbl bool) { robotgo.Click(btn(b), dbl) }

func (rgInjector) Toggle(b string, down bool) {
	dir := "up"
	if down {
		dir = "down"
	}
	robotgo.Toggle(btn(b), dir)
}

func (rgInjector) Scroll(dx, dy int) { robotgo.Scroll(dx, dy) }

func (rgInjector) KeyTap(k string, mods []string) {
	key := k
	if mapped, ok := keyMap[k]; ok {
		key = mapped
	}
	args := make([]interface{}, 0, len(mods))
	for _, m := range mods {
		if mm, ok := modMap[m]; ok {
			args = append(args, mm)
		}
	}
	robotgo.KeyTap(key, args...)
}

func (rgInjector) TypeStr(s string) { robotgo.TypeStr(s) }

func (rgInjector) Media(k string) {
	if mk, ok := mediaMap[k]; ok {
		robotgo.KeyTap(mk)
	}
}

func (rgInjector) ScreenSize() (int, int) { return robotgo.GetScreenSize() }

func (rgInjector) HostInfo() (string, string) {
	h, _ := os.Hostname()
	return h, runtime.GOOS
}
