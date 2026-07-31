package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
)

// waylandSocket reports whether a Wayland compositor is running here, and under
// which display name.
//
// WAYLAND_DISPLAY on its own is not enough. The agent is routinely started from
// a systemd user unit, an SSH shell or a terminal that never inherited it, and
// with the variable missing the old code fell straight through to xclip. On a
// Wayland session that writes into XWayland's clipboard rather than the
// compositor's: the copy never reaches the real selection, and a clipboard
// manager watching it (clipman, cliphist, wl-clip-persist — all of them sit on
// `wl-paste --watch`) never sees the entry either. So look for the socket.
func waylandSocket() (string, bool) {
	if d := os.Getenv("WAYLAND_DISPLAY"); d != "" {
		return d, true
	}
	if strings.EqualFold(os.Getenv("XDG_SESSION_TYPE"), "wayland") {
		// the session says Wayland but the name didn't reach us; wayland-0 is
		// the compositor default and the only guess worth making.
		return "wayland-0", true
	}
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		return "", false
	}
	matches, err := filepath.Glob(filepath.Join(runtimeDir, "wayland-*"))
	if err != nil {
		return "", false
	}
	for _, m := range matches {
		if strings.HasSuffix(m, ".lock") {
			continue
		}
		if info, err := os.Stat(m); err == nil && info.Mode()&os.ModeSocket != 0 {
			return filepath.Base(m), true
		}
	}
	return "", false
}

// newClipper picks the clipboard tool that matches the session: wl-clipboard on
// Wayland, xclip or xsel on X11. Nothing installed → clipboard sync is simply
// unavailable, and the phone says so rather than failing quietly.
//
// On Wayland the write goes through `wl-copy`, which sets the compositor's own
// selection — that is what puts the copy into a clipboard manager's history.
func newClipper() Clipper {
	if display, ok := waylandSocket(); ok && have("wl-copy") && have("wl-paste") {
		// pass the display explicitly: it is exactly the variable that may be
		// missing from the agent's environment.
		log.Printf("[clip] wl-clipboard backend (WAYLAND_DISPLAY=%s)", display)
		return execClipper{
			readCmd:  []string{"wl-paste", "--no-newline"},
			writeCmd: []string{"wl-copy"},
			env:      []string{"WAYLAND_DISPLAY=" + display},
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
		log.Println("[clip] wl-clipboard backend (no session detected)")
		return execClipper{
			readCmd:  []string{"wl-paste", "--no-newline"},
			writeCmd: []string{"wl-copy"},
		}
	}
	return noClipper{}
}
