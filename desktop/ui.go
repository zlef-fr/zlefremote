package main

import (
	"fmt"
	"image/color"
	"math"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/zlef-fr/zlefremote/desktop/internal/core"
)

const (
	viewConnect = "connect"
	viewSession = "session"
)

type hostInfo struct {
	name    string
	os      string
	w, h    int
	screens [][2]int
	caps    map[string]bool
}

type toast struct {
	text string
	born time.Time
	kind string // "" | "warn" | "bad"
}

// App is the whole client: an ebiten Game with two views (pick a machine, drive
// a machine). Everything it draws is redrawn every tick, so state lives here in
// plain fields — there is no retained widget tree to keep in sync.
type App struct {
	faces *faces
	lang  *Lang
	store *core.Store
	clipb core.Clipboard

	view string
	fade float64 // view entrance fade, 0..1
	spin float64 // spinner phase
	now  time.Time

	// ── connect view ──
	field    string
	fieldErr string
	sel      int // -1 = the paste field, else index into store.Devices

	// ── session view ──
	conn      *core.Conn
	stream    *core.Stream
	pump      *InputPump
	target    core.Target
	state     string
	statusMsg string
	host      hostInfo
	display   int
	preset    core.Preset
	hudOn     bool
	helpOn    bool
	tex       *ebiten.Image
	texSeq    uint64
	toasts    []toast
	pingSeq   int
	pingAt    time.Time
	pingSent  time.Time
	latency   time.Duration

	clipCh        chan string
	clipStop      chan struct{}
	clipLocalLast string

	sw, sh int
	quit   bool
}

func NewApp(store *core.Store, lang *Lang) *App {
	a := &App{
		faces:  loadFaces(),
		lang:   lang,
		store:  store,
		clipb:  core.NewClipboard(),
		view:   viewConnect,
		sel:    -1,
		hudOn:  true,
		preset: core.PresetByName(store.Prefs.Quality),
		clipCh: make(chan string, 4),
	}
	return a
}

func (a *App) T(k string) string { return a.lang.T(k) }

// ── ebiten.Game ─────────────────────────────────────────────────────────────

func (a *App) Layout(ow, oh int) (int, int) {
	a.sw, a.sh = ow, oh
	return ow, oh
}

func (a *App) Update() error {
	a.now = time.Now()
	a.spin += 1.0 / 60
	if a.fade < 1 {
		a.fade = math.Min(1, a.fade+0.08)
	}
	a.pruneToasts()

	if a.conn != nil {
		a.drainEvents()
	}
	if a.quit {
		return ebiten.Termination
	}

	switch a.view {
	case viewConnect:
		a.updateConnect()
	case viewSession:
		a.updateSession()
	}
	return nil
}

func (a *App) Draw(screen *ebiten.Image) {
	screen.Fill(colBG)
	switch a.view {
	case viewConnect:
		a.drawConnect(screen)
	case viewSession:
		a.drawSession(screen)
	}
	a.drawToasts(screen)
}

// ── connect view ────────────────────────────────────────────────────────────

func (a *App) updateConnect() {
	if a.helpOn {
		if len(inpututil.AppendJustPressedKeys(nil)) > 0 {
			a.helpOn = false
		}
		return
	}
	for _, r := range ebiten.AppendInputChars(nil) {
		if r >= 0x20 && r != 0x7f {
			a.sel = -1
			a.field += string(r)
			a.fieldErr = ""
		}
	}
	ctrl := ebiten.IsKeyPressed(ebiten.KeyControlLeft) || ebiten.IsKeyPressed(ebiten.KeyControlRight) ||
		ebiten.IsKeyPressed(ebiten.KeyMetaLeft) || ebiten.IsKeyPressed(ebiten.KeyMetaRight)

	for _, k := range inpututil.AppendJustPressedKeys(nil) {
		switch {
		case k == ebiten.KeyV && ctrl:
			if s, err := a.clipb.Read(); err == nil {
				a.field += strings.TrimSpace(s)
				a.sel = -1
				a.fieldErr = ""
			}
		case k == ebiten.KeyBackspace:
			if a.sel == -1 && a.field != "" {
				r := []rune(a.field)
				a.field = string(r[:len(r)-1])
			}
		case k == ebiten.KeyEnter || k == ebiten.KeyNumpadEnter:
			a.connectSelected()
		case k == ebiten.KeyTab || k == ebiten.KeyArrowDown:
			if len(a.store.Devices) > 0 {
				a.sel++
				if a.sel >= len(a.store.Devices) {
					a.sel = -1
				}
			}
		case k == ebiten.KeyArrowUp:
			if len(a.store.Devices) > 0 {
				a.sel--
				if a.sel < -1 {
					a.sel = len(a.store.Devices) - 1
				}
			}
		case k == ebiten.KeyDelete:
			if a.sel >= 0 {
				a.store.Forget(a.sel)
				a.sel = -1
			}
		case k == ebiten.KeySlash && ctrl:
			a.helpOn = true
		case k == ebiten.KeyEscape:
			if a.field != "" {
				a.field = ""
			} else {
				a.quit = true
			}
		}
	}

	// clicking a saved machine selects it; a second click connects
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		for i := range a.store.Devices {
			r := a.deviceRowRect(i)
			if float64(mx) >= r.x && float64(mx) <= r.x+r.w && float64(my) >= r.y && float64(my) <= r.y+r.h {
				if a.sel == i {
					a.connectSelected()
				} else {
					a.sel = i
				}
			}
		}
	}
}

// connectSelected opens whichever target the user picked: the pasted link or a
// saved machine.
func (a *App) connectSelected() {
	var t core.Target
	if a.sel >= 0 && a.sel < len(a.store.Devices) {
		t = a.store.Devices[a.sel].Target
	} else {
		s := strings.TrimSpace(a.field)
		if s == "" {
			return
		}
		parsed, err := core.ParseTarget(s)
		if err != nil {
			a.fieldErr = a.T("error.link") + err.Error()
			return
		}
		t = parsed
		// only a --remember agent keeps a stable address worth saving
		if t.Persistent {
			a.store.Remember(t, "")
			a.toast(a.T("toast.saved"), "")
		}
	}
	a.startSession(t)
}

// connectLayout is the single source of truth for the picker's geometry, so
// what is drawn and what is clickable can never drift apart.
type connectLayout struct {
	x, y, w, h     float64 // card
	pad            float64
	titleY         float64
	taglineY       float64
	fieldLabelY    float64
	fieldY, fieldH float64
	hintY          float64
	savedLabelY    float64
	rowsY, rowH    float64
	footerY        float64
}

func (a *App) connectLayout() connectLayout {
	w, h := float64(a.sw), float64(a.sh)
	l := connectLayout{pad: 36, fieldH: 50, rowH: 56}
	rows := float64(len(a.store.Devices))
	if rows > 4 {
		rows = 4
	}
	l.w = math.Min(760, w-80)
	l.h = math.Min(math.Max(430, 380+rows*(l.rowH+8)), h-60)
	l.x = (w - l.w) / 2
	l.y = (h - l.h) / 2

	y := l.y + l.pad
	l.titleY = y
	y += 50
	l.taglineY = y
	y += 44
	l.fieldLabelY = y
	y += 24
	l.fieldY = y
	y += l.fieldH + 12
	l.hintY = y
	y += 34
	l.savedLabelY = y
	y += 26
	l.rowsY = y
	l.footerY = l.y + l.h - 54
	return l
}

func (a *App) drawConnect(screen *ebiten.Image) {
	a.drawParticles(screen)
	l := a.connectLayout()
	a.panel(screen, l.x, l.y, l.w, l.h, a.fade)

	cardW, pad := l.w, l.pad
	x := l.x
	cx := l.x + pad

	// wordmark: extra tracking, the way the site sets its logotype
	a.textTracked(screen, "ZLEFREMOTE", a.faces.xl, cx, l.titleY, 3, alpha(colText, a.fade))
	a.text(screen, a.T("app.tagline"), a.faces.sm, cx, l.taglineY, alpha(colTextMute, a.fade))

	// ── paste field ──
	a.text(screen, a.T("connect.paste"), a.faces.xs, cx, l.fieldLabelY, alpha(colTextSoft, a.fade))
	cy := l.fieldY
	fieldH := l.fieldH
	focus := a.sel == -1
	fieldCol := colSurface2
	if focus {
		fieldCol = colSurface3
	}
	vector.DrawFilledRect(screen, float32(cx), float32(cy), float32(cardW-2*pad), float32(fieldH), alpha(fieldCol, a.fade), true)
	border := colLine
	if focus {
		border = colLineHot
	}
	vector.StrokeRect(screen, float32(cx), float32(cy), float32(cardW-2*pad), float32(fieldH), 1, alpha(border, a.fade), true)

	shown := a.field
	if shown == "" {
		a.text(screen, "https://remote.zlef.fr/r/AB12CD#k=…", a.faces.md, cx+14, cy+15, alpha(colTextFain, a.fade))
	} else {
		// keep the tail visible while pasting a long link
		if tw, _ := text.Measure(shown, a.faces.md, 0); tw > cardW-2*pad-28 {
			for len(shown) > 8 {
				shown = shown[1:]
				if tw, _ := text.Measure(shown, a.faces.md, 0); tw <= cardW-2*pad-40 {
					break
				}
			}
			shown = "…" + shown
		}
		a.text(screen, shown, a.faces.md, cx+14, cy+15, alpha(colText, a.fade))
	}
	if focus && math.Mod(a.spin, 1.0) < 0.55 {
		tw, _ := text.Measure(shown, a.faces.md, 0)
		vector.DrawFilledRect(screen, float32(cx+16+tw), float32(cy+13), 2, 24, alpha(colOliveSft, a.fade), true)
	}
	if a.fieldErr != "" {
		a.text(screen, a.fieldErr, a.faces.xs, cx, l.hintY, alpha(colDanger, a.fade))
	} else {
		a.text(screen, a.T("connect.hint"), a.faces.xs, cx, l.hintY, alpha(colTextFain, a.fade))
	}

	// ── saved machines ──
	a.text(screen, strings.ToUpper(a.T("connect.saved")), a.faces.xs, cx, l.savedLabelY, alpha(colTextMute, a.fade))
	if len(a.store.Devices) == 0 {
		a.wrapText(screen, a.T("connect.none"), a.faces.sm, cx, l.rowsY, cardW-2*pad, alpha(colTextFain, a.fade))
	}
	for i, d := range a.store.Devices {
		r := a.deviceRowRect(i)
		if r.y+r.h > l.footerY-10 {
			break
		}
		bg := colSurface2
		if a.sel == i {
			bg = colSurface3
		}
		vector.DrawFilledRect(screen, float32(r.x), float32(r.y), float32(r.w), float32(r.h), alpha(bg, a.fade), true)
		if a.sel == i {
			vector.DrawFilledRect(screen, float32(r.x), float32(r.y), 3, float32(r.h), alpha(colOliveSft, a.fade), true)
		}
		a.text(screen, d.Label, a.faces.md, r.x+16, r.y+8, alpha(colText, a.fade))
		sub := d.Describe()
		if !d.LastUsed.IsZero() {
			sub += " · " + d.LastUsed.Local().Format("2006-01-02 15:04")
		}
		a.text(screen, sub, a.faces.xs, r.x+16, r.y+32, alpha(colTextMute, a.fade))
		if a.sel == i {
			a.textRight(screen, a.T("connect.forget"), a.faces.xs, r.x+r.w-16, r.y+32, alpha(colTextFain, a.fade))
		}
	}

	// footer help
	a.wrapText(screen, a.T("connect.help"), a.faces.xs, x+pad, l.footerY, cardW-2*pad, alpha(colTextFain, a.fade*0.9))

	if a.helpOn {
		a.drawHelp(screen)
	}
}

func (a *App) deviceRowRect(i int) rect {
	l := a.connectLayout()
	return rect{x: l.x + l.pad, y: l.rowsY + float64(i)*(l.rowH+8), w: l.w - 2*l.pad, h: l.rowH}
}

// ── session ─────────────────────────────────────────────────────────────────

func (a *App) startSession(t core.Target) {
	a.stopSession()
	conn, err := core.NewConn(t)
	if err != nil {
		a.fieldErr = a.T("error.link") + err.Error()
		return
	}
	a.target = t
	a.conn = conn
	a.stream = core.NewStream()
	a.pump = NewInputPump(a.sendCmd, &a.store.Prefs)
	a.pump.SetRawKeyboard(a.store.Prefs.RawKeyboard)
	a.display = 0
	a.host = hostInfo{}
	a.tex, a.texSeq = nil, 0
	a.state = core.StateConnecting
	a.statusMsg = ""
	a.latency = 0
	a.view = viewSession
	a.fade = 0
	conn.Start()
	ebiten.SetWindowTitle(a.T("app.title") + " — " + t.Describe())
}

func (a *App) stopSession() {
	if a.pump != nil {
		a.pump.Grab(false)
	}
	a.stopClipWatch()
	if a.conn != nil {
		a.conn.Close()
		a.conn = nil
	}
	if a.stream != nil {
		a.stream.Reset()
	}
	ebiten.SetCursorMode(ebiten.CursorModeVisible)
	ebiten.SetWindowTitle(a.T("app.title"))
}

func (a *App) sendCmd(v any) {
	if a.conn != nil {
		_ = a.conn.Send(v)
	}
}

// drainEvents consumes everything the transport produced since the last tick.
func (a *App) drainEvents() {
	for {
		select {
		case ev, ok := <-a.conn.Events():
			if !ok {
				return
			}
			a.handleEvent(ev)
		default:
			return
		}
	}
}

func (a *App) handleEvent(ev core.Event) {
	switch ev.Kind {
	case "state":
		a.state = ev.Text
		if ev.Text == core.StateReconnecting || ev.Text == core.StateClosed {
			// a half-open socket must not leave keys held on the host
			if a.pump != nil {
				a.pump.ReleaseAll()
			}
			if a.stream != nil {
				a.stream.Reset()
			}
			a.tex = nil
		}
	case "error":
		switch ev.Text {
		case "no_such_room":
			a.statusMsg = a.T("error.room")
		case "host_left":
			a.statusMsg = a.T("error.hostgone")
		case "":
		default:
			a.statusMsg = a.T("error.conn") + ev.Text
		}
	case "cmd":
		a.handleCmd(ev.Cmd)
	}
}

func (a *App) handleCmd(m map[string]any) {
	switch m["t"] {
	case "welcome":
		a.onWelcome(m)
	case "f":
		a.stream.Feed(m)
	case "viewerr":
		if r, _ := m["reason"].(string); r == "unsupported" {
			a.statusMsg = a.T("state.noscreen")
		}
	case "pong":
		if int(core.Num(m["i"])) == a.pingSeq && !a.pingSent.IsZero() {
			a.latency = time.Since(a.pingSent)
			a.pingSent = time.Time{}
		}
	case "clip":
		s, _ := m["s"].(string)
		if s == "" || !a.store.Prefs.ClipSync {
			return
		}
		a.clipLocalLast = s
		if err := a.clipb.Write(s); err == nil {
			a.toast(a.T("toast.clipin"), "")
		}
	}
}

func (a *App) onWelcome(m map[string]any) {
	a.state = core.StatePaired
	a.statusMsg = ""
	a.host.name, _ = m["name"].(string)
	a.host.os, _ = m["os"].(string)
	if s, ok := m["screen"].(map[string]any); ok {
		a.host.w, a.host.h = int(core.Num(s["w"])), int(core.Num(s["h"]))
	}
	a.host.screens = nil
	if list, ok := m["screens"].([]any); ok {
		for _, it := range list {
			if s, ok := it.(map[string]any); ok {
				a.host.screens = append(a.host.screens, [2]int{int(core.Num(s["w"])), int(core.Num(s["h"]))})
			}
		}
	}
	a.host.caps = map[string]bool{}
	if c, ok := m["cap"].(map[string]any); ok {
		for k, v := range c {
			b, _ := v.(bool)
			a.host.caps[k] = b
		}
	}
	a.pump.SetHostScreen(a.host.w, a.host.h)
	a.pump.SetKeyHold(a.host.caps["keyhold"])
	ebiten.SetWindowTitle(a.T("app.title") + " — " + a.host.name)

	if a.host.caps["screen"] {
		a.sendView(true)
	} else {
		a.statusMsg = a.T("state.noscreen")
	}
	if a.host.caps["clip"] && a.store.Prefs.ClipSync {
		a.sendCmd(map[string]any{"t": "clipwatch", "on": true})
		a.startClipWatch()
	}
	// a reconnect must not re-save the device or bounce the grab
	if a.target.Persistent {
		a.store.Remember(a.target, a.host.name)
	}
}

func (a *App) sendView(on bool) {
	if !on {
		a.sendCmd(map[string]any{"t": "view", "on": false})
		return
	}
	a.sendCmd(map[string]any{
		"t": "view", "on": true,
		"fps": a.preset.FPS, "q": a.preset.Quality, "scale": a.preset.Scale,
		"d": a.display,
	})
}

func (a *App) updateSession() {
	if a.helpOn {
		if len(inpututil.AppendJustPressedKeys(nil)) > 0 {
			a.helpOn = false
		}
		return
	}

	// latency probe (the host echoes the id back)
	if a.state == core.StatePaired && time.Since(a.pingAt) > 2*time.Second {
		a.pingAt = time.Now()
		a.pingSeq++
		a.pingSent = time.Now()
		a.sendCmd(map[string]any{"t": "ping", "i": a.pingSeq})
	}

	// local core.Clipboard changes → host
	select {
	case s := <-a.clipCh:
		if a.host.caps["clip"] && a.store.Prefs.ClipSync {
			a.sendCmd(map[string]any{"t": "clip", "s": s})
		}
	default:
	}

	a.pump.SetView(a.viewRect())
	for _, act := range a.pump.Update() {
		a.handleAction(act)
	}

	// clicking inside the picture is the natural "I want to drive this" gesture
	if !a.pump.Grabbed() && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && a.state == core.StatePaired {
		mx, my := ebiten.CursorPosition()
		r := a.viewRect()
		if float64(mx) >= r.x && float64(mx) <= r.x+r.w && float64(my) >= r.y && float64(my) <= r.y+r.h {
			// start the remote pointer where the user clicked
			a.pump.Grab(true)
			a.pump.WarpTo((float64(mx)-r.x)/r.w, (float64(my)-r.y)/r.h)
			a.toast(a.T("hud.grabbed"), "")
		}
	}
	if !a.pump.Grabbed() {
		for _, k := range inpututil.AppendJustPressedKeys(nil) {
			if k == ebiten.KeyEscape {
				a.leaveSession()
			}
		}
	}

	// upload the newest decoded frame
	if img, seq := a.stream.Latest(); img != nil && seq != a.texSeq {
		a.texSeq = seq
		b := img.Bounds()
		if a.tex == nil || a.tex.Bounds() != b {
			a.tex = ebiten.NewImage(b.Dx(), b.Dy())
		}
		a.tex.WritePixels(img.Pix)
	}
}

func (a *App) handleAction(act string) {
	switch act {
	case actGrabOn:
		a.toast(a.T("hud.grabbed"), "")
	case actGrabOff:
		a.toast(a.T("hud.released"), "")
	case actFullscreen:
		fs := !ebiten.IsFullscreen()
		ebiten.SetFullscreen(fs)
		a.store.Prefs.Fullscreen = fs
		_ = a.store.Save()
	case actPreset:
		a.preset = core.NextPreset(a.preset.Name)
		a.store.Prefs.Quality = a.preset.Name
		_ = a.store.Save()
		a.sendView(true)
		a.toast(a.T("toast.preset")+a.preset.Name, "")
	case actMonitor:
		if n := len(a.host.screens); n > 1 {
			a.display = (a.display + 1) % n
			a.stream.Reset()
			a.tex = nil
			a.pump.SetPointer(.5, .5)
			a.sendView(true)
			a.toast(fmt.Sprintf("%s%d/%d", a.T("toast.monitor"), a.display+1, n), "")
		}
	case actRawKeyboard:
		a.pump.SetRawKeyboard(!a.pump.RawKeyboard())
		a.store.Prefs.RawKeyboard = a.pump.RawKeyboard()
		_ = a.store.Save()
		if a.pump.RawKeyboard() {
			a.toast(a.T("toast.raw.on"), "warn")
		} else {
			a.toast(a.T("toast.raw.off"), "")
		}
	case actHUD:
		a.hudOn = !a.hudOn
	case actHelp:
		a.helpOn = true
	case actClipPush:
		if s, err := a.clipb.Read(); err == nil && s != "" && a.host.caps["clip"] {
			a.clipLocalLast = s
			a.sendCmd(map[string]any{"t": "clip", "s": s})
			a.toast(a.T("toast.clipout"), "")
		}
	case actDisconnect:
		a.leaveSession()
	}
}

func (a *App) leaveSession() {
	a.stopSession()
	a.view = viewConnect
	a.fade = 0
	a.sel = -1
}

// viewRect is where the host's picture is drawn: the largest rectangle with the
// remote aspect ratio that fits the window (letterboxed, never cropped — a
// cropped remote screen would hide UI the user is trying to click).
func (a *App) viewRect() rect {
	w, h := float64(a.sw), float64(a.sh)
	ar := 16.0 / 9.0
	if a.tex != nil {
		b := a.tex.Bounds()
		if b.Dy() > 0 {
			ar = float64(b.Dx()) / float64(b.Dy())
		}
	} else if a.host.h > 0 {
		ar = float64(a.host.w) / float64(a.host.h)
	}
	vw, vh := w, w/ar
	if vh > h {
		vh, vw = h, h*ar
	}
	return rect{x: (w - vw) / 2, y: (h - vh) / 2, w: vw, h: vh}
}

func (a *App) drawSession(screen *ebiten.Image) {
	r := a.viewRect()
	if a.tex != nil {
		op := &ebiten.DrawImageOptions{}
		b := a.tex.Bounds()
		op.GeoM.Scale(r.w/float64(b.Dx()), r.h/float64(b.Dy()))
		op.GeoM.Translate(r.x, r.y)
		op.Filter = ebiten.FilterLinear
		op.ColorScale.ScaleAlpha(float32(a.fade))
		screen.DrawImage(a.tex, op)
		a.drawRemoteCursor(screen, r)
	} else {
		a.drawWaiting(screen)
	}

	if !a.pump.Grabbed() && a.tex != nil {
		// a light scrim makes it obvious the picture is not live-controlled
		vector.DrawFilledRect(screen, 0, 0, float32(a.sw), float32(a.sh), alpha(colBG, 0.35), false)
		a.pill(screen, float64(a.sw)/2, float64(a.sh)-72, a.T("hud.click"))
	}
	if a.hudOn {
		a.drawHUD(screen)
	}
	if a.helpOn {
		a.drawHelp(screen)
	}
}

// drawRemoteCursor paints where the host pointer is. The agent's capture does
// not include the cursor, and a remote desktop with an invisible pointer is
// unusable — so we draw the one we command ourselves.
func (a *App) drawRemoteCursor(screen *ebiten.Image, r rect) {
	if !a.pump.Grabbed() {
		return
	}
	nx, ny := a.pump.Pointer()
	x := float32(r.x + nx*r.w)
	y := float32(r.y + ny*r.h)
	arrow := func(off, scale float32) *vector.Path {
		p := &vector.Path{}
		p.MoveTo(x-off, y-off)
		p.LineTo(x-off, y+17*scale+off)
		p.LineTo(x+4.5*scale, y+13*scale)
		p.LineTo(x+8*scale, y+20*scale+off)
		p.LineTo(x+11*scale+off, y+18.5*scale)
		p.LineTo(x+7.5*scale, y+11.5*scale)
		p.LineTo(x+13*scale+off, y+11*scale)
		p.Close()
		return p
	}
	// a dark outline first, so the arrow stays visible over a white remote window
	outline := &vector.DrawPathOptions{AntiAlias: true}
	outline.ColorScale.ScaleWithColor(colBG)
	vector.FillPath(screen, arrow(1.6, 1), nil, outline)
	fill := &vector.DrawPathOptions{AntiAlias: true}
	fill.ColorScale.ScaleWithColor(colOliveBrt)
	vector.FillPath(screen, arrow(0, 1), nil, fill)
}

func (a *App) drawWaiting(screen *ebiten.Image) {
	msg := a.T("state.nostream")
	switch a.state {
	case core.StateConnecting:
		msg = a.T("state.connecting")
	case core.StateLinked:
		msg = a.T("state.linked")
	case core.StateReconnecting:
		msg = a.T("state.reconnecting")
	case core.StateClosed:
		msg = a.T("state.closed")
	}
	cx, cy := float64(a.sw)/2, float64(a.sh)/2
	a.spinner(screen, cx, cy-30, 18)
	a.textCenter(screen, msg, a.faces.md, cx, cy+16, colTextSoft)
	if a.statusMsg != "" {
		a.textCenter(screen, a.statusMsg, a.faces.sm, cx, cy+44, colDanger)
	}
	a.textCenter(screen, "Esc — "+a.T("help.quit"), a.faces.xs, cx, float64(a.sh)-40, colTextFain)
}

func (a *App) drawHUD(screen *ebiten.Image) {
	fps, kbps, stale := a.stream.Stats()
	parts := []string{}
	if a.host.name != "" {
		parts = append(parts, a.host.name)
	}
	if n := len(a.host.screens); n > 1 {
		parts = append(parts, fmt.Sprintf("%s %d/%d", a.T("hud.monitor"), a.display+1, n))
	}
	if a.state == core.StatePaired {
		parts = append(parts, fmt.Sprintf("%.0f fps · %.0f kB/s", fps, kbps))
		if a.latency > 0 {
			parts = append(parts, fmt.Sprintf("%s %d ms", a.T("hud.latency"), a.latency.Milliseconds()))
		}
	} else {
		// numbers from before the drop would read as a live link — they are not
		parts = append(parts, "— fps · — kB/s")
	}
	parts = append(parts, a.preset.Name)
	if a.pump.RawKeyboard() {
		parts = append(parts, a.T("hud.raw"))
	}
	if a.host.caps["clip"] && a.store.Prefs.ClipSync {
		parts = append(parts, a.T("hud.clipon"))
	}
	line := strings.Join(parts, "   ·   ")

	tw, _ := text.Measure(line, a.faces.monoSm, 0)
	w := tw + 54
	x := (float64(a.sw) - w) / 2
	y := 14.0
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(w), 34, alpha(colSurface1, 0.92), true)
	vector.StrokeRect(screen, float32(x), float32(y), float32(w), 34, 1, colLine, true)

	// state dot: olive when live, amber when the picture went stale, rose when
	// the transport is down
	dot := colSuccess
	switch {
	case a.state != core.StatePaired:
		dot = colDanger
	case stale:
		dot = colWarning
	}
	vector.DrawFilledCircle(screen, float32(x+18), float32(y+17), 4, dot, true)
	a.text(screen, line, a.faces.monoSm, x+32, y+9, colTextSoft)

	hint := a.T("hud.take")
	if a.pump.Grabbed() {
		hint = a.T("hud.release")
	}
	a.textCenter(screen, hint+"   ·   "+a.T("hud.help"), a.faces.xs, float64(a.sw)/2, y+40, colTextFain)
}

func (a *App) drawHelp(screen *ebiten.Image) {
	vector.DrawFilledRect(screen, 0, 0, float32(a.sw), float32(a.sh), alpha(colBG, 0.82), false)
	rows := []string{
		"help.grab", "help.fullscr", "help.preset", "help.monitor", "help.raw",
		"help.hud", "help.clip", "help.cad", "help.alttab", "help.super",
		"help.esc", "help.quit",
	}
	w := 520.0
	h := float64(len(rows))*28 + 110
	x := (float64(a.sw) - w) / 2
	y := (float64(a.sh) - h) / 2
	a.panel(screen, x, y, w, h, 1)
	a.text(screen, a.T("help.title"), a.faces.lg, x+30, y+26, colText)
	for i, k := range rows {
		a.text(screen, a.T(k), a.faces.sm, x+30, y+70+float64(i)*28, colTextSoft)
	}
	a.text(screen, a.T("help.close"), a.faces.xs, x+30, y+h-30, colTextFain)
}

// ── core.Clipboard watching ──────────────────────────────────────────────────────

// startClipWatch polls this machine's core.Clipboard and forwards changes. Polling
// (rather than an X11/Win32 hook) keeps the client cgo-free; 1.2 s matches the
// agent's own cadence and is imperceptible in practice.
func (a *App) startClipWatch() {
	if !a.clipb.Available() || a.clipStop != nil {
		return
	}
	stop := make(chan struct{})
	a.clipStop = stop
	if s, err := a.clipb.Read(); err == nil {
		a.clipLocalLast = s
	}
	go func() {
		t := time.NewTicker(1200 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				s, err := a.clipb.Read()
				if err != nil || s == "" || len(s) > 16*1024 || s == a.clipLocalLast {
					continue
				}
				a.clipLocalLast = s
				select {
				case a.clipCh <- s:
				default:
				}
			}
		}
	}()
}

func (a *App) stopClipWatch() {
	if a.clipStop != nil {
		close(a.clipStop)
		a.clipStop = nil
	}
}

// ── small drawing helpers ───────────────────────────────────────────────────

func (a *App) panel(dst *ebiten.Image, x, y, w, h, fade float64) {
	vector.DrawFilledRect(dst, float32(x), float32(y), float32(w), float32(h), alpha(colSurface1, fade), true)
	vector.StrokeRect(dst, float32(x), float32(y), float32(w), float32(h), 1, alpha(colLine, fade), true)
	// a hairline of olive along the top edge — the zlef signature
	vector.DrawFilledRect(dst, float32(x), float32(y), float32(w), 2, alpha(colOliveMid, fade), true)
}

func (a *App) pill(dst *ebiten.Image, cx, cy float64, s string) {
	tw, _ := text.Measure(s, a.faces.sm, 0)
	w := tw + 34
	x := cx - w/2
	vector.DrawFilledRect(dst, float32(x), float32(cy-16), float32(w), 32, alpha(colSurface2, 0.95), true)
	vector.StrokeRect(dst, float32(x), float32(cy-16), float32(w), 32, 1, colLineHot, true)
	a.textCenter(dst, s, a.faces.sm, cx, cy-9, colText)
}

func (a *App) spinner(dst *ebiten.Image, cx, cy, r float64) {
	// three orbiting dots, olive→grape, is enough motion to read as "working"
	for i := 0; i < 3; i++ {
		ang := a.spin*2.4 + float64(i)*2.1
		x := cx + math.Cos(ang)*r
		y := cy + math.Sin(ang)*r
		c := colOliveSft
		if i == 2 {
			c = colGrapeSft
		}
		vector.DrawFilledCircle(dst, float32(x), float32(y), float32(4-float64(i)*0.7), alpha(c, 1-float64(i)*0.22), true)
	}
}

// drawParticles is the connect screen's atmosphere: a slow olive/grape drift,
// the native echo of the site's particle field.
func (a *App) drawParticles(dst *ebiten.Image) {
	w, h := float64(a.sw), float64(a.sh)
	for i := 0; i < 46; i++ {
		f := float64(i)
		x := math.Mod(f*137.5+a.spin*(6+math.Mod(f, 5)), w+80) - 40
		y := math.Mod(f*91.3+math.Sin(a.spin*0.4+f)*24, h)
		c := colOlive
		if i%5 == 0 {
			c = colGrapeSft
		}
		vector.DrawFilledCircle(dst, float32(x), float32(y), float32(1+math.Mod(f, 3)*0.5), alpha(c, 0.5), true)
	}
}

func (a *App) toast(s, kind string) {
	a.toasts = append(a.toasts, toast{text: s, born: time.Now(), kind: kind})
	if len(a.toasts) > 4 {
		a.toasts = a.toasts[len(a.toasts)-4:]
	}
}

func (a *App) pruneToasts() {
	keep := a.toasts[:0]
	for _, t := range a.toasts {
		if time.Since(t.born) < 3200*time.Millisecond {
			keep = append(keep, t)
		}
	}
	a.toasts = keep
}

func (a *App) drawToasts(dst *ebiten.Image) {
	y := float64(a.sh) - 46
	for i := len(a.toasts) - 1; i >= 0; i-- {
		t := a.toasts[i]
		age := time.Since(t.born).Seconds()
		fade := 1.0
		if age > 2.4 {
			fade = math.Max(0, (3.2-age)/0.8)
		}
		rise := math.Min(1, age*6) // slide up on entrance
		col := colText
		switch t.kind {
		case "warn":
			col = colWarning
		case "bad":
			col = colDanger
		}
		tw, _ := text.Measure(t.text, a.faces.sm, 0)
		w := tw + 30
		x := (float64(a.sw) - w) / 2
		ty := y - (1-rise)*10
		vector.DrawFilledRect(dst, float32(x), float32(ty-15), float32(w), 30, alpha(colSurface2, 0.92*fade), true)
		vector.StrokeRect(dst, float32(x), float32(ty-15), float32(w), 30, 1, alpha(colLine, fade), true)
		a.textCenter(dst, t.text, a.faces.sm, float64(a.sw)/2, ty-9, alpha(col, fade))
		y -= 38
	}
}

func (a *App) text(dst *ebiten.Image, s string, f *text.GoTextFace, x, y float64, col color.Color) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(col)
	text.Draw(dst, s, f, op)
}

// textTracked draws letter-spaced text — used for the wordmark, which the
// zlef identity sets wide.
func (a *App) textTracked(dst *ebiten.Image, s string, f *text.GoTextFace, x, y, track float64, col color.Color) {
	for _, r := range s {
		g := string(r)
		a.text(dst, g, f, x, y, col)
		w, _ := text.Measure(g, f, 0)
		x += w + track
	}
}

func (a *App) textCenter(dst *ebiten.Image, s string, f *text.GoTextFace, cx, y float64, col color.Color) {
	tw, _ := text.Measure(s, f, 0)
	a.text(dst, s, f, cx-tw/2, y, col)
}

func (a *App) textRight(dst *ebiten.Image, s string, f *text.GoTextFace, right, y float64, col color.Color) {
	tw, _ := text.Measure(s, f, 0)
	a.text(dst, s, f, right-tw, y, col)
}

// wrapText is a naive word wrapper — enough for the two paragraphs we show.
func (a *App) wrapText(dst *ebiten.Image, s string, f *text.GoTextFace, x, y, maxW float64, col color.Color) {
	words := strings.Fields(s)
	line := ""
	dy := 0.0
	for _, w := range words {
		test := strings.TrimSpace(line + " " + w)
		if tw, _ := text.Measure(test, f, 0); tw > maxW && line != "" {
			a.text(dst, line, f, x, y+dy, col)
			dy += f.Size * 1.45
			line = w
			continue
		}
		line = test
	}
	if line != "" {
		a.text(dst, line, f, x, y+dy, col)
	}
}
