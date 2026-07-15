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
// The first version of this file fixed that by rebinding ONE spare keycode to
// each character and pressing it — but that races against XKB-aware clients
// (GTK/Qt): they reload the keymap asynchronously (often debounced) after a
// MappingNotify, so with a remap per keystroke every press can resolve against
// the FIRST keymap the app loaded, and the whole session types the first
// character over and over.
//
// So now we do it the way `xdotool type` really does:
//   1. Look the keysym up in the HOST'S OWN keymap first. Almost every char a
//      user types exists on their layout — press its real keycode with the
//      modifiers its column requires (Shift / AltGr / Mode_switch) via XTEST.
//      No remap, no MappingNotify, no race, and the layout is honoured by
//      construction.
//   2. Only glyphs absent from the layout (emoji, foreign scripts) fall back
//      to binding a spare keycode — with a real settle delay (30 ms) after the
//      remap so clients can reload the keymap, and no rebind when the same
//      glyph repeats.

/*
#cgo LDFLAGS: -lX11 -lXtst
#include <X11/Xlib.h>
#include <X11/keysym.h>
#include <X11/XF86keysym.h>
#include <X11/extensions/XTest.h>
#include <stdlib.h>
#include <unistd.h>

// Column → modifier convention of the core keymap (what xmodmap -pke shows):
//   col 0: none          col 1: Shift
//   col 2: Mode_switch   col 3: Shift+Mode_switch   (group 2)
//   col 4: AltGr         col 5: Shift+AltGr         (ISO_Level3_Shift)
// Preference order picks the cheapest modifier combo first.
static KeyCode zr_find(KeySym *km, int min_kc, int max_kc, int per,
                       KeySym sym, int *out_col) {
	static const int prefs[6] = {0, 1, 4, 5, 2, 3};
	for (int p = 0; p < 6; p++) {
		int col = prefs[p];
		if (col >= per) continue;
		for (int kc = min_kc; kc <= max_kc; kc++) {
			if (km[(kc - min_kc) * per + col] == sym) {
				*out_col = col;
				return (KeyCode)kc;
			}
		}
	}
	return 0;
}

// Press kc with the modifiers its column requires. Modifiers are injected as
// real XTEST presses of the modifier keys themselves, so the server tracks
// state exactly as if a user typed the combo.
static void zr_press(Display *dpy, KeyCode kc, int col,
                     KeyCode kc_shift, KeyCode kc_altgr, KeyCode kc_mode) {
	int shift = (col == 1 || col == 3 || col == 5);
	KeyCode level = 0;
	if (col == 4 || col == 5) level = kc_altgr;
	else if (col == 2 || col == 3) level = kc_mode ? kc_mode : kc_altgr;

	if (shift && kc_shift) XTestFakeKeyEvent(dpy, kc_shift, True, 0);
	if (level) XTestFakeKeyEvent(dpy, level, True, 0);
	XTestFakeKeyEvent(dpy, kc, True, 0);
	XTestFakeKeyEvent(dpy, kc, False, 0);
	if (level) XTestFakeKeyEvent(dpy, level, False, 0);
	if (shift && kc_shift) XTestFakeKeyEvent(dpy, kc_shift, False, 0);
	XSync(dpy, False);
	usleep(2000);
}

static void zr_type_runes(unsigned int *cps, int n) {
	Display *dpy = XOpenDisplay(NULL);
	if (!dpy) return;

	int min_kc, max_kc, per;
	XDisplayKeycodes(dpy, &min_kc, &max_kc);
	KeySym *km = XGetKeyboardMapping(dpy, min_kc, max_kc - min_kc + 1, &per);
	if (!km || per <= 0) { if (km) XFree(km); XCloseDisplay(dpy); return; }

	KeyCode kc_shift = XKeysymToKeycode(dpy, XK_Shift_L);
	KeyCode kc_altgr = XKeysymToKeycode(dpy, XK_ISO_Level3_Shift);
	if (!kc_altgr) kc_altgr = XKeysymToKeycode(dpy, XK_Mode_switch);
	KeyCode kc_mode = XKeysymToKeycode(dpy, XK_Mode_switch);

	// Spare keycode for glyphs the layout doesn't have: one with no keysym
	// bound at any level. Scan from the top (the usual unused range).
	int spare = -1;
	for (int kc = max_kc; kc >= min_kc && spare < 0; kc--) {
		int empty = 1;
		for (int l = 0; l < per; l++) {
			if (km[(kc - min_kc) * per + l] != NoSymbol) { empty = 0; break; }
		}
		if (empty) spare = kc;
	}
	if (spare < 0) spare = max_kc;

	KeySym *buf = (KeySym *)malloc(sizeof(KeySym) * per);
	if (!buf) { XFree(km); XCloseDisplay(dpy); return; }
	KeySym bound = NoSymbol; // what the spare keycode currently emits
	int used_spare = 0;

	for (int i = 0; i < n; i++) {
		unsigned int cp = cps[i];
		// X11 keysym for a code point: Latin-1 maps 1:1, everything else uses
		// the 0x01000000 Unicode keysym range.
		KeySym sym = (cp <= 0xff) ? (KeySym)cp : (KeySym)(cp | 0x01000000);

		int col = 0;
		KeyCode kc = zr_find(km, min_kc, max_kc, per, sym, &col);
		if (kc && kc != (KeyCode)spare) {
			zr_press(dpy, kc, col, kc_shift, kc_altgr, kc_mode);
			continue;
		}

		// Not on the layout — bind it to the spare keycode. Rebind only when
		// the glyph changes, and give XKB clients time to reload the keymap
		// before the press (they refresh asynchronously on MappingNotify).
		if (sym != bound) {
			for (int l = 0; l < per; l++) buf[l] = sym;
			XChangeKeyboardMapping(dpy, spare, per, buf, 1);
			XSync(dpy, False);
			usleep(30000);
			bound = sym;
			used_spare = 1;
		}
		zr_press(dpy, (KeyCode)spare, 0, kc_shift, kc_altgr, kc_mode);
	}

	if (used_spare) {
		for (int l = 0; l < per; l++) buf[l] = NoSymbol;
		XChangeKeyboardMapping(dpy, spare, per, buf, 1);
		XSync(dpy, False);
	}

	free(buf);
	XFree(km);
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
