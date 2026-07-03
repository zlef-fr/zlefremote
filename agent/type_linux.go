//go:build robotgo && linux

package main

// Layout-independent text injection for X11.
//
// robotgo's TypeStr on Linux is not layout-independent: ASCII goes through
// keyCodeForChar() plus a US-QWERTY shift heuristic (so on AZERTY, typing "1"
// presses the digit key without Shift and yields "&", etc.), and non-ASCII is
// turned into a lowercase Unicode keysym name ("U00e9") that XStringToKeysym
// rejects (it wants "U00E9"), dropping accented characters.
//
// Instead we do what `xdotool type` does: temporarily bind the target code
// point to a spare (unused) keycode at every modifier level, synthesise a
// modifier-free keypress via XTEST, then restore the keycode. Because the glyph
// is bound at level 0 (and every level as a safety net), the emitted character
// is exactly the code point regardless of the host keyboard layout.

/*
#cgo LDFLAGS: -lX11 -lXtst
#include <X11/Xlib.h>
#include <X11/extensions/XTest.h>
#include <stdlib.h>
#include <unistd.h>

static void zr_type_runes(unsigned int *cps, int n) {
	Display *dpy = XOpenDisplay(NULL);
	if (!dpy) return;

	int min_kc, max_kc, per;
	XDisplayKeycodes(dpy, &min_kc, &max_kc);
	KeySym *km = XGetKeyboardMapping(dpy, min_kc, max_kc - min_kc + 1, &per);
	if (!km || per <= 0) { if (km) XFree(km); XCloseDisplay(dpy); return; }

	// Pick a spare keycode: one with no keysym bound at any level. Scan from the
	// top (high keycodes are the usual unused range). Fall back to the last one.
	int spare = -1;
	for (int kc = max_kc; kc >= min_kc && spare < 0; kc--) {
		int empty = 1;
		for (int l = 0; l < per; l++) {
			if (km[(kc - min_kc) * per + l] != NoSymbol) { empty = 0; break; }
		}
		if (empty) spare = kc;
	}
	if (spare < 0) spare = max_kc;
	XFree(km);

	KeySym *buf = (KeySym *)malloc(sizeof(KeySym) * per);
	if (!buf) { XCloseDisplay(dpy); return; }

	for (int i = 0; i < n; i++) {
		unsigned int cp = cps[i];
		// X11 keysym for a code point: Latin-1 maps 1:1, everything else uses
		// the 0x01000000 Unicode keysym range.
		KeySym sym = (cp <= 0xff) ? (KeySym)cp : (KeySym)(cp | 0x01000000);

		for (int l = 0; l < per; l++) buf[l] = sym; // bind every level to the glyph
		XChangeKeyboardMapping(dpy, spare, per, buf, 1);
		XSync(dpy, False);
		usleep(1200); // let the server apply the remap before the keypress

		XTestFakeKeyEvent(dpy, (KeyCode)spare, True, 0);
		XTestFakeKeyEvent(dpy, (KeyCode)spare, False, 0);
		XSync(dpy, False);
		usleep(1200);
	}

	// Restore the spare keycode to unbound.
	for (int l = 0; l < per; l++) buf[l] = NoSymbol;
	XChangeKeyboardMapping(dpy, spare, per, buf, 1);
	XSync(dpy, False);

	free(buf);
	XCloseDisplay(dpy);
}
*/
import "C"

import "unsafe"

func typeText(s string) {
	runes := []rune(s)
	if len(runes) == 0 {
		return
	}
	cps := make([]C.uint, len(runes))
	for i, r := range runes {
		cps[i] = C.uint(r)
	}
	C.zr_type_runes((*C.uint)(unsafe.Pointer(&cps[0])), C.int(len(cps)))
}
