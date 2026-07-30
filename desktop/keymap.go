package main

import "github.com/hajimehoshi/ebiten/v2"

// Ebiten reports PHYSICAL keys (KeyA is the key labelled Q on an AZERTY
// board). The host's injector resolves a key NAME through its own keymap, so a
// physical key travels as the name robotgo knows (agent/inject_robotgo.go,
// keyNames), and the host applies its own layout to it.
//
// That is exactly what we want for shortcuts (Ctrl+C stays "the copy chord" on
// both machines) but NOT for typing text: on mismatched layouts the same
// physical key means different characters. Characters therefore travel through
// the {"t":"text"} path instead, which the agent injects layout-correctly (see
// agent/type_linux.go). textKeys below marks the keys whose characters we let
// the OS compose for us, so the input pump can skip their key events.
var keyNames = map[ebiten.Key]string{
	ebiten.KeyA: "a", ebiten.KeyB: "b", ebiten.KeyC: "c", ebiten.KeyD: "d",
	ebiten.KeyE: "e", ebiten.KeyF: "f", ebiten.KeyG: "g", ebiten.KeyH: "h",
	ebiten.KeyI: "i", ebiten.KeyJ: "j", ebiten.KeyK: "k", ebiten.KeyL: "l",
	ebiten.KeyM: "m", ebiten.KeyN: "n", ebiten.KeyO: "o", ebiten.KeyP: "p",
	ebiten.KeyQ: "q", ebiten.KeyR: "r", ebiten.KeyS: "s", ebiten.KeyT: "t",
	ebiten.KeyU: "u", ebiten.KeyV: "v", ebiten.KeyW: "w", ebiten.KeyX: "x",
	ebiten.KeyY: "y", ebiten.KeyZ: "z",

	ebiten.KeyDigit0: "0", ebiten.KeyDigit1: "1", ebiten.KeyDigit2: "2",
	ebiten.KeyDigit3: "3", ebiten.KeyDigit4: "4", ebiten.KeyDigit5: "5",
	ebiten.KeyDigit6: "6", ebiten.KeyDigit7: "7", ebiten.KeyDigit8: "8",
	ebiten.KeyDigit9: "9",

	// punctuation: robotgo has no name for these, so it falls back to its
	// character path — send the character itself.
	ebiten.KeyMinus: "-", ebiten.KeyEqual: "=", ebiten.KeyBracketLeft: "[",
	ebiten.KeyBracketRight: "]", ebiten.KeyBackslash: "\\", ebiten.KeySemicolon: ";",
	ebiten.KeyQuote: "'", ebiten.KeyComma: ",", ebiten.KeyPeriod: ".",
	ebiten.KeySlash: "/", ebiten.KeyBackquote: "`", ebiten.KeyIntlBackslash: "\\",

	ebiten.KeyEnter: "enter", ebiten.KeyNumpadEnter: "enter",
	ebiten.KeyEscape: "escape", ebiten.KeyTab: "tab", ebiten.KeySpace: "space",
	ebiten.KeyBackspace: "backspace", ebiten.KeyDelete: "delete",
	ebiten.KeyInsert: "insert", ebiten.KeyHome: "home", ebiten.KeyEnd: "end",
	ebiten.KeyPageUp: "pageup", ebiten.KeyPageDown: "pagedown",
	ebiten.KeyArrowUp: "up", ebiten.KeyArrowDown: "down",
	ebiten.KeyArrowLeft: "left", ebiten.KeyArrowRight: "right",
	ebiten.KeyCapsLock: "capslock", ebiten.KeyPrintScreen: "printscreen",
	ebiten.KeyContextMenu: "menu", ebiten.KeyNumLock: "num_lock",

	ebiten.KeyF1: "f1", ebiten.KeyF2: "f2", ebiten.KeyF3: "f3", ebiten.KeyF4: "f4",
	ebiten.KeyF5: "f5", ebiten.KeyF6: "f6", ebiten.KeyF7: "f7", ebiten.KeyF8: "f8",
	ebiten.KeyF9: "f9", ebiten.KeyF10: "f10", ebiten.KeyF11: "f11", ebiten.KeyF12: "f12",

	ebiten.KeyNumpad0: "num0", ebiten.KeyNumpad1: "num1", ebiten.KeyNumpad2: "num2",
	ebiten.KeyNumpad3: "num3", ebiten.KeyNumpad4: "num4", ebiten.KeyNumpad5: "num5",
	ebiten.KeyNumpad6: "num6", ebiten.KeyNumpad7: "num7", ebiten.KeyNumpad8: "num8",
	ebiten.KeyNumpad9: "num9", ebiten.KeyNumpadAdd: "num_plus",
	ebiten.KeyNumpadSubtract: "num_minus", ebiten.KeyNumpadMultiply: "num_asterisk",
	ebiten.KeyNumpadDivide: "num_slash",

	ebiten.KeyShiftLeft: "shift", ebiten.KeyShiftRight: "shift",
	ebiten.KeyControlLeft: "ctrl", ebiten.KeyControlRight: "ctrl",
	ebiten.KeyAltLeft: "alt", ebiten.KeyAltRight: "alt",
	ebiten.KeyMetaLeft: "cmd", ebiten.KeyMetaRight: "cmd",
}

// textKeys are the keys whose output is a character: their presses are covered
// by the composed-text path, so we don't also send them as key events (that
// would type everything twice, and wrongly on a different layout).
var textKeys = map[ebiten.Key]bool{
	ebiten.KeyA: true, ebiten.KeyB: true, ebiten.KeyC: true, ebiten.KeyD: true,
	ebiten.KeyE: true, ebiten.KeyF: true, ebiten.KeyG: true, ebiten.KeyH: true,
	ebiten.KeyI: true, ebiten.KeyJ: true, ebiten.KeyK: true, ebiten.KeyL: true,
	ebiten.KeyM: true, ebiten.KeyN: true, ebiten.KeyO: true, ebiten.KeyP: true,
	ebiten.KeyQ: true, ebiten.KeyR: true, ebiten.KeyS: true, ebiten.KeyT: true,
	ebiten.KeyU: true, ebiten.KeyV: true, ebiten.KeyW: true, ebiten.KeyX: true,
	ebiten.KeyY: true, ebiten.KeyZ: true,
	ebiten.KeyDigit0: true, ebiten.KeyDigit1: true, ebiten.KeyDigit2: true,
	ebiten.KeyDigit3: true, ebiten.KeyDigit4: true, ebiten.KeyDigit5: true,
	ebiten.KeyDigit6: true, ebiten.KeyDigit7: true, ebiten.KeyDigit8: true,
	ebiten.KeyDigit9: true,
	ebiten.KeyMinus:  true, ebiten.KeyEqual: true, ebiten.KeyBracketLeft: true,
	ebiten.KeyBracketRight: true, ebiten.KeyBackslash: true, ebiten.KeySemicolon: true,
	ebiten.KeyQuote: true, ebiten.KeyComma: true, ebiten.KeyPeriod: true,
	ebiten.KeySlash: true, ebiten.KeyBackquote: true, ebiten.KeyIntlBackslash: true,
	ebiten.KeySpace:   true,
	ebiten.KeyNumpad0: true, ebiten.KeyNumpad1: true, ebiten.KeyNumpad2: true,
	ebiten.KeyNumpad3: true, ebiten.KeyNumpad4: true, ebiten.KeyNumpad5: true,
	ebiten.KeyNumpad6: true, ebiten.KeyNumpad7: true, ebiten.KeyNumpad8: true,
	ebiten.KeyNumpad9: true, ebiten.KeyNumpadAdd: true, ebiten.KeyNumpadSubtract: true,
	ebiten.KeyNumpadMultiply: true, ebiten.KeyNumpadDivide: true,
}

// modifierKeys map to the modifier names the host understands. Modifiers are
// handled lazily (see input.go): forwarding them eagerly would corrupt typed
// characters, because AltGr composition happens on THIS keyboard.
var modifierKeys = map[ebiten.Key]string{
	ebiten.KeyShiftLeft: "shift", ebiten.KeyShiftRight: "shift",
	ebiten.KeyControlLeft: "ctrl", ebiten.KeyControlRight: "ctrl",
	ebiten.KeyAltLeft: "alt", ebiten.KeyAltRight: "alt",
	ebiten.KeyMetaLeft: "cmd", ebiten.KeyMetaRight: "cmd",
}

// hotkeyModifier is the local-only "menu" key, borrowed from the remote-desktop
// tradition (virt-manager, RDP): while Right Ctrl is held nothing is forwarded,
// and its chords drive this client. Tapping it alone toggles the input grab.
const hotkeyModifier = ebiten.KeyControlRight

func keyName(k ebiten.Key) (string, bool) {
	n, ok := keyNames[k]
	return n, ok
}
