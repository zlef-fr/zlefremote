package main

import (
	"log"
	"os"
)

// newClipper picks the first clipboard tool that fits the session: Wayland's
// wl-clipboard when a Wayland compositor is running, otherwise xclip or xsel on
// X11. Nothing installed → clipboard sync is simply unavailable.
func newClipper() Clipper {
	wayland := os.Getenv("WAYLAND_DISPLAY") != ""
	if wayland && have("wl-copy") && have("wl-paste") {
		log.Println("[clip] wl-clipboard backend")
		return execClipper{
			readCmd:  []string{"wl-paste", "--no-newline"},
			writeCmd: []string{"wl-copy"},
		}
	}
	if have("xclip") {
		log.Println("[clip] xclip backend")
		return execClipper{
			readCmd:  []string{"xclip", "-selection", "clipboard", "-o"},
			writeCmd: []string{"xclip", "-selection", "clipboard", "-i"},
		}
	}
	if have("xsel") {
		log.Println("[clip] xsel backend")
		return execClipper{
			readCmd:  []string{"xsel", "--clipboard", "--output"},
			writeCmd: []string{"xsel", "--clipboard", "--input"},
		}
	}
	if have("wl-copy") && have("wl-paste") {
		log.Println("[clip] wl-clipboard backend")
		return execClipper{
			readCmd:  []string{"wl-paste", "--no-newline"},
			writeCmd: []string{"wl-copy"},
		}
	}
	return noClipper{}
}
