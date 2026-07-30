package main

import (
	"math"
	"time"
	"unicode"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/zlef-fr/zlefremote/desktop/internal/core"
)

// InputPump turns this machine's mouse and keyboard into host commands.
//
// Pointer: while grabbed the OS cursor is *captured* (ebiten reports unbounded
// virtual coordinates), so we keep our own cursor position, clamp it to the
// remote screen and send it as a normalized {"t":"mvabs"}. That gives absolute,
// self-correcting positioning — a dropped packet can't leave the two pointers
// permanently offset the way relative motion would — while the pointer can
// never escape the window mid-drag.
//
// Keyboard: characters and chords travel by different routes, see keymap.go.
// Modifiers are forwarded LAZILY: a modifier is only pressed on the host right
// before a chord or a click needs it, because the AltGr/Shift composition that
// produces a character already happened on this keyboard — holding those
// modifiers on the host too would corrupt the injected text.
type InputPump struct {
	send  func(any)
	prefs *core.Prefs

	grabbed  bool
	keyHold  bool // host supports kdown/kup (agent ≥ 1.7.0)
	rawKeys  bool
	screenW  int
	screenH  int
	viewRect rect

	// virtual pointer, normalized 0..1 over the viewed display
	nx, ny   float64
	lastCur  [2]int
	haveCur  bool
	lastSent time.Time

	sentMods    map[string]bool
	heldKeys    map[string]bool
	heldButtons map[string]bool

	scrollX, scrollY float64

	hotkeyUsed bool
}

type rect struct{ x, y, w, h float64 }

// hotkey actions the app must handle (everything the pump can't do itself).
const (
	actFullscreen  = "fullscreen"
	actPreset      = "core.Preset"
	actMonitor     = "monitor"
	actRawKeyboard = "raw"
	actDisconnect  = "disconnect"
	actHUD         = "hud"
	actClipPush    = "clip"
	actGrabOn      = "grab-on"
	actGrabOff     = "grab-off"
	actHelp        = "help"
)

func NewInputPump(send func(any), prefs *core.Prefs) *InputPump {
	return &InputPump{
		send:        send,
		prefs:       prefs,
		nx:          .5,
		ny:          .5,
		sentMods:    map[string]bool{},
		heldKeys:    map[string]bool{},
		heldButtons: map[string]bool{},
	}
}

func (p *InputPump) SetHostScreen(w, h int) { p.screenW, p.screenH = w, h }
func (p *InputPump) SetKeyHold(v bool)      { p.keyHold = v }
func (p *InputPump) SetView(r rect)         { p.viewRect = r }
func (p *InputPump) Grabbed() bool          { return p.grabbed }
func (p *InputPump) Pointer() (float64, float64) {
	return p.nx, p.ny
}

// Grab starts/stops forwarding. Releasing everything on the way out is not
// optional: a modifier or button left down on the host would keep "typing" long
// after the user looked away.
func (p *InputPump) Grab(on bool) {
	if p.grabbed == on {
		return
	}
	p.grabbed = on
	if on {
		ebiten.SetCursorMode(ebiten.CursorModeCaptured)
		p.haveCur = false
	} else {
		ebiten.SetCursorMode(ebiten.CursorModeVisible)
		p.ReleaseAll()
	}
}

// ReleaseAll lifts every key/button we are holding on the host.
func (p *InputPump) ReleaseAll() {
	for b := range p.heldButtons {
		p.send(map[string]any{"t": "up", "b": b})
		delete(p.heldButtons, b)
	}
	for k := range p.heldKeys {
		if p.keyHold {
			p.send(map[string]any{"t": "kup", "k": k})
		}
		delete(p.heldKeys, k)
	}
	p.releaseMods()
}

// Update runs once per tick and returns the local hotkey actions it recognised.
func (p *InputPump) Update() []string {
	var actions []string

	// The window losing focus (alt-tab away, screen lock) must never leave keys
	// stuck on the other machine.
	if !ebiten.IsFocused() {
		if p.grabbed {
			p.Grab(false)
			actions = append(actions, actGrabOff)
		}
		return actions
	}

	if acts, handled := p.hotkeys(); handled {
		return append(actions, acts...)
	}
	if !p.grabbed {
		return actions
	}

	p.pointer()
	p.buttons()
	p.wheel()
	p.keyboard()
	return actions
}

// hotkeys implements the Right-Ctrl "menu key" chords. While it is held nothing
// reaches the host; a tap on its own toggles the grab.
func (p *InputPump) hotkeys() ([]string, bool) {
	held := ebiten.IsKeyPressed(hotkeyModifier)
	if inpututil.IsKeyJustPressed(hotkeyModifier) {
		p.hotkeyUsed = false
		// drop anything currently held so the chord can't stick
		p.ReleaseAll()
	}
	if !held {
		if inpututil.IsKeyJustReleased(hotkeyModifier) && !p.hotkeyUsed {
			p.Grab(!p.grabbed)
			if p.grabbed {
				return []string{actGrabOn}, true
			}
			return []string{actGrabOff}, true
		}
		return nil, false
	}

	var out []string
	for _, k := range inpututil.AppendJustPressedKeys(nil) {
		// A modifier is never a chord by itself. This also covers the extra
		// keys ebiten reports alongside the physical one (the "either side"
		// aliases, and on X11 a Right-Ctrl press that also surfaces as
		// ControlLeft) — counting any of them would stop a bare tap of the
		// menu key from toggling the grab.
		if _, isMod := modifierKeys[k]; isMod || k == ebiten.KeyControl ||
			k == ebiten.KeyShift || k == ebiten.KeyAlt || k == ebiten.KeyMeta {
			continue
		}
		switch k {
		case ebiten.KeyF:
			out = append(out, actFullscreen)
		case ebiten.KeyQ:
			out = append(out, actDisconnect)
		case ebiten.KeyM:
			out = append(out, actMonitor)
		case ebiten.KeyP:
			out = append(out, actPreset)
		case ebiten.KeyK:
			out = append(out, actRawKeyboard)
		case ebiten.KeyH:
			out = append(out, actHUD)
		case ebiten.KeyC:
			out = append(out, actClipPush)
		case ebiten.KeySlash:
			out = append(out, actHelp)
		case ebiten.KeyDelete:
			// the one chord no OS lets an app receive: send it to the host
			p.sendChord("delete", []string{"ctrl", "alt"})
		case ebiten.KeyTab:
			p.sendChord("tab", []string{"alt"})
		case ebiten.KeyS:
			p.sendChord("cmd", nil)
		case ebiten.KeyEscape:
			p.sendChord("escape", nil)
		}
		p.hotkeyUsed = true
	}
	return out, true
}

// sendChord presses a key with modifiers as one atomic action on the host.
func (p *InputPump) sendChord(key string, mods []string) {
	p.send(map[string]any{"t": "key", "k": key, "mods": mods})
}

func (p *InputPump) pointer() {
	cx, cy := ebiten.CursorPosition()
	if !p.haveCur {
		p.lastCur = [2]int{cx, cy}
		p.haveCur = true
		return
	}
	dx, dy := float64(cx-p.lastCur[0]), float64(cy-p.lastCur[1])
	p.lastCur = [2]int{cx, cy}
	if dx == 0 && dy == 0 {
		return
	}
	sens := p.prefs.Sensitivity
	if sens <= 0 {
		sens = 1
	}
	// One pixel of local motion moves the remote pointer by one *remote* pixel:
	// the view is a scaled copy of the host screen, so divide by the view size.
	if p.viewRect.w > 0 && p.viewRect.h > 0 {
		p.nx += dx * sens / p.viewRect.w
		p.ny += dy * sens / p.viewRect.h
	}
	p.nx = clamp01(p.nx)
	p.ny = clamp01(p.ny)

	// The host injects at most one move per frame anyway; 120 Hz is plenty and
	// keeps a fast flick from flooding a mobile uplink.
	if time.Since(p.lastSent) < 8*time.Millisecond {
		return
	}
	p.lastSent = time.Now()
	p.send(map[string]any{"t": "mvabs", "nx": round6(p.nx), "ny": round6(p.ny)})
}

func (p *InputPump) buttons() {
	for btn, name := range map[ebiten.MouseButton]string{
		ebiten.MouseButtonLeft:   "left",
		ebiten.MouseButtonRight:  "right",
		ebiten.MouseButtonMiddle: "middle",
	} {
		if inpututil.IsMouseButtonJustPressed(btn) {
			p.syncMods()
			// pin the pointer before the press: a click must land where the
			// user sees the cursor even if the last move was throttled away.
			p.send(map[string]any{"t": "mvabs", "nx": round6(p.nx), "ny": round6(p.ny)})
			p.send(map[string]any{"t": "down", "b": name})
			p.heldButtons[name] = true
		}
		if inpututil.IsMouseButtonJustReleased(btn) {
			p.send(map[string]any{"t": "up", "b": name})
			delete(p.heldButtons, name)
		}
	}
}

func (p *InputPump) wheel() {
	dx, dy := ebiten.Wheel()
	if dx == 0 && dy == 0 {
		return
	}
	speed := p.prefs.ScrollSpeed
	if speed <= 0 {
		speed = 1
	}
	if p.prefs.Natural {
		dy = -dy
	}
	// ebiten reports notches (±1 per detent, fractional on touchpads); robotgo
	// scrolls in "clicks", so 3 lines per notch matches a normal desktop.
	p.scrollX += dx * speed * 3
	p.scrollY += dy * speed * 3
	ix, iy := int(p.scrollX), int(p.scrollY)
	if ix == 0 && iy == 0 {
		return
	}
	p.scrollX -= float64(ix)
	p.scrollY -= float64(iy)
	p.send(map[string]any{"t": "scroll", "dx": ix, "dy": iy})
}

func (p *InputPump) keyboard() {
	pressed := inpututil.AppendJustPressedKeys(nil)
	released := inpututil.AppendJustReleasedKeys(nil)
	chars := ebiten.AppendInputChars(nil)

	if p.rawKeys {
		p.rawKeyboard(pressed, released)
		return
	}

	// composed characters first: they are what the user actually typed
	if s := printable(chars); s != "" {
		p.releaseMods() // never type through a held modifier (see type above)
		p.send(map[string]any{"t": "text", "s": s})
	}

	for _, k := range pressed {
		if _, isMod := modifierKeys[k]; isMod {
			continue // lazy: pressed on demand by syncMods
		}
		if textKeys[k] && !p.modsActive() {
			continue // plain character: already handled by the text path
		}
		name, ok := keyName(k)
		if !ok {
			continue
		}
		p.syncMods()
		if p.keyHold {
			p.send(map[string]any{"t": "kdown", "k": name})
			p.heldKeys[name] = true
		} else {
			// older agent: no hold/release, tap with the live modifiers
			p.send(map[string]any{"t": "key", "k": name, "mods": p.activeMods()})
		}
	}

	for _, k := range released {
		if _, isMod := modifierKeys[k]; isMod {
			p.syncModsRelease(k)
			continue
		}
		name, ok := keyName(k)
		if !ok || !p.heldKeys[name] {
			continue
		}
		delete(p.heldKeys, name)
		if p.keyHold {
			p.send(map[string]any{"t": "kup", "k": name})
		}
	}
}

// rawKeyboard forwards physical keys verbatim, modifiers included, and never
// sends composed text. Right for games and for remote-controlling a machine
// whose layout matches this one; wrong for typing across layouts.
func (p *InputPump) rawKeyboard(pressed, released []ebiten.Key) {
	for _, k := range pressed {
		if name, ok := keyName(k); ok {
			if p.keyHold {
				p.send(map[string]any{"t": "kdown", "k": name})
				p.heldKeys[name] = true
			} else {
				p.send(map[string]any{"t": "key", "k": name, "mods": p.activeMods()})
			}
		}
	}
	for _, k := range released {
		if name, ok := keyName(k); ok {
			delete(p.heldKeys, name)
			if p.keyHold {
				p.send(map[string]any{"t": "kup", "k": name})
			}
		}
	}
}

// modsActive reports whether a chord-forming modifier (not Shift, which only
// changes the character) is held locally.
func (p *InputPump) modsActive() bool {
	return ebiten.IsKeyPressed(ebiten.KeyControlLeft) ||
		ebiten.IsKeyPressed(ebiten.KeyAltLeft) || ebiten.IsKeyPressed(ebiten.KeyAltRight) ||
		ebiten.IsKeyPressed(ebiten.KeyMetaLeft) || ebiten.IsKeyPressed(ebiten.KeyMetaRight)
}

// activeMods is the live modifier list for the tap fallback.
func (p *InputPump) activeMods() []string {
	var m []string
	if ebiten.IsKeyPressed(ebiten.KeyShiftLeft) || ebiten.IsKeyPressed(ebiten.KeyShiftRight) {
		m = append(m, "shift")
	}
	if ebiten.IsKeyPressed(ebiten.KeyControlLeft) {
		m = append(m, "ctrl")
	}
	if ebiten.IsKeyPressed(ebiten.KeyAltLeft) || ebiten.IsKeyPressed(ebiten.KeyAltRight) {
		m = append(m, "alt")
	}
	if ebiten.IsKeyPressed(ebiten.KeyMetaLeft) || ebiten.IsKeyPressed(ebiten.KeyMetaRight) {
		m = append(m, "meta")
	}
	return m
}

// syncMods presses on the host exactly the modifiers held here, right before an
// event that needs them.
func (p *InputPump) syncMods() {
	if !p.keyHold {
		return
	}
	want := map[string]bool{}
	for _, m := range p.activeMods() {
		if m == "meta" {
			m = "cmd"
		}
		want[m] = true
	}
	for m := range want {
		if !p.sentMods[m] {
			p.send(map[string]any{"t": "kdown", "k": m})
			p.sentMods[m] = true
		}
	}
	for m := range p.sentMods {
		if !want[m] {
			p.send(map[string]any{"t": "kup", "k": m})
			delete(p.sentMods, m)
		}
	}
}

// syncModsRelease lifts a modifier on the host when it is released here (only
// if we had actually pressed it).
func (p *InputPump) syncModsRelease(k ebiten.Key) {
	name := modifierKeys[k]
	if name == "" || !p.sentMods[name] {
		return
	}
	// left/right variants share one host name — wait until both are up
	switch name {
	case "shift":
		if ebiten.IsKeyPressed(ebiten.KeyShiftLeft) || ebiten.IsKeyPressed(ebiten.KeyShiftRight) {
			return
		}
	case "ctrl":
		if ebiten.IsKeyPressed(ebiten.KeyControlLeft) {
			return
		}
	case "alt":
		if ebiten.IsKeyPressed(ebiten.KeyAltLeft) || ebiten.IsKeyPressed(ebiten.KeyAltRight) {
			return
		}
	case "cmd":
		if ebiten.IsKeyPressed(ebiten.KeyMetaLeft) || ebiten.IsKeyPressed(ebiten.KeyMetaRight) {
			return
		}
	}
	p.send(map[string]any{"t": "kup", "k": name})
	delete(p.sentMods, name)
}

func (p *InputPump) releaseMods() {
	for m := range p.sentMods {
		if p.keyHold {
			p.send(map[string]any{"t": "kup", "k": m})
		}
		delete(p.sentMods, m)
	}
}

// SetRawKeyboard switches typing modes, releasing whatever the old mode held.
func (p *InputPump) SetRawKeyboard(v bool) {
	if p.rawKeys == v {
		return
	}
	p.ReleaseAll()
	p.rawKeys = v
}

func (p *InputPump) RawKeyboard() bool { return p.rawKeys }

// SetPointer re-centres the virtual pointer (used when the viewed monitor
// changes, so the cursor doesn't jump to a stale corner).
func (p *InputPump) SetPointer(nx, ny float64) {
	p.nx, p.ny = clamp01(nx), clamp01(ny)
}

// WarpTo moves the host pointer to a normalized point right away. Taking
// control by clicking in the picture uses it, so the remote cursor lands under
// the local one instead of wherever it was left.
func (p *InputPump) WarpTo(nx, ny float64) {
	p.SetPointer(nx, ny)
	p.send(map[string]any{"t": "mvabs", "nx": round6(p.nx), "ny": round6(p.ny)})
}

func printable(rs []rune) string {
	out := make([]rune, 0, len(rs))
	for _, r := range rs {
		if r == '\n' || r == '\r' || r == '\t' || r == 0x7f {
			continue // these travel as key events, not as text
		}
		if unicode.IsControl(r) {
			continue
		}
		out = append(out, r)
	}
	return string(out)
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// round6 keeps the normalized pointer accurate to the pixel on a 4K screen
// (1/4096 needs ~5 decimals) while keeping frames small.
func round6(v float64) float64 { return math.Round(v*1e6) / 1e6 }
