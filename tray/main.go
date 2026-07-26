//go:build windows

// ZlefRemote — Windows tray app.
//
// The Windows counterpart of the Xfce panel plugin: start a ZlefRemote session
// and show the pairing QR from the notification area, without ever opening a
// terminal. It drives the same `zlefremote-agent.exe` in `-machine` mode, so
// the transport, the end-to-end encryption and the input injection all stay in
// the one agent implementation (see agent.go / protocol.go).
package main

import (
	"flag"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

const version = "1.0.0"

type App struct {
	hwnd  uintptr
	hinst uintptr
	hicon uintptr

	dpi     int32
	fonts   *fonts
	lay     *layout
	visible bool

	sess *Session
	run  *Runner
	set  Settings

	qr    *qrBitmap
	qrURL string
	qrMax int32 // current QR budget in px (shrinks to fit a short screen)

	hover   int
	pressed int
	copied  bool
	busy    bool
	errMsg  string

	animate  bool    // honours "show animations" in Windows ease-of-access
	alpha    float64 // fade-in progress 0..1
	pulse    float64 // 0..1 breathing value for the waiting state
	pulsePhi float64
	targetX  int32
	targetY  int32
}

var (
	app             *App
	msgTaskbarCreat uint32
	msgShowPopup    uint32
)

func goArch() string { return runtime.GOARCH }

func main() {
	runtime.LockOSThread()

	lang := flag.String("lang", "", "force the interface language (en|fr)")
	agentPath := flag.String("agent", "", "path to zlefremote-agent.exe")
	openNow := flag.Bool("window", false, "open the control popup on launch")
	startMode := flag.String("start", "", "start a session immediately: lan | remote")
	ver := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *ver {
		os.Stdout.WriteString("zlefremote-tray " + version + "\n")
		return
	}

	if *lang != "" {
		setLang(*lang)
	} else if env := os.Getenv("ZLEFREMOTE_LANG"); env != "" {
		setLang(env)
	} else {
		setLang(uiLanguage())
	}

	// one tray icon per user session; a second launch just opens the popup
	msgShowPopup = registerMessage("ZlefRemoteTray.ShowPopup")
	if singleInstanceTaken() {
		pPostMessage.Call(0xFFFF /*HWND_BROADCAST*/, uintptr(msgShowPopup), 0, 0)
		return
	}

	if *agentPath != "" {
		os.Setenv("ZLEFREMOTE_AGENT", *agentPath)
	}

	makeDPIAware()

	app = &App{
		sess:    NewSession(),
		run:     NewRunner(),
		set:     loadSettings(),
		dpi:     96,
		hover:   hsNone,
		pressed: hsNone,
		animate: animationsEnabled(),
	}
	app.sess.OnPaired = func() { app.trayBalloon(t("title"), t("balloon_paired")) }
	app.sess.OnDisconnect = func() { app.trayBalloon(t("title"), t("balloon_bye")) }
	if app.run.Path == "" {
		app.errMsg = t("no_agent")
	}

	app.createWindow()
	app.hicon = makeIcon(app.trayIconPx(), StatusIdle)
	app.trayAdd()
	defer app.trayRemove()

	switch strings.ToLower(*startMode) {
	case "lan":
		app.set.Remote = false
		app.startSession()
	case "remote":
		app.set.Remote = true
		app.startSession()
	}
	if *openNow {
		app.showPopup()
	}

	app.loop()
}

// ── window plumbing ─────────────────────────────────────────────────────────

func (a *App) createWindow() {
	hinst, _, _ := pGetModuleHandle.Call(0)
	a.hinst = hinst
	msgTaskbarCreat = registerMessage("TaskbarCreated")

	className := u16("ZlefRemoteTrayWindow")
	cursor, _, _ := pLoadCursor.Call(0, idcArrow)
	wc := wndClassEx{
		Size:      uint32(unsafe.Sizeof(wndClassEx{})),
		Style:     csHRedraw | csVRedraw | csDropShadow,
		WndProc:   windows.NewCallback(wndProc),
		Instance:  a.hinst,
		Cursor:    cursor,
		ClassName: className,
	}
	pRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc)))

	hwnd, _, _ := pCreateWindowEx.Call(
		wsExToolWin|wsExTopmost|wsExLayered,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(u16("ZlefRemote"))),
		wsPopup,
		0, 0, 100, 100,
		0, 0, a.hinst, 0)
	a.hwnd = hwnd
	a.dpi = windowDPI(hwnd)
	a.qrMax = a.sc(196)
	a.fonts = newFonts(a.dpi)
	pSetLayeredWindowAttr.Call(a.hwnd, 0, 255, lwaAlpha)
}

func (a *App) loop() {
	var m msg
	for {
		r, _, _ := pGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			return
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		pDispatchMessage.Call(uintptr(unsafe.Pointer(&m)))
	}
}

func wndProc(hwnd uintptr, message uintptr, wparam uintptr, lparam uintptr) uintptr {
	a := app
	if a == nil {
		r, _, _ := pDefWindowProc.Call(hwnd, message, wparam, lparam)
		return r
	}
	switch uint32(message) {
	case wmTrayCallback:
		switch uint32(loWord(lparam)) & 0xFFFF {
		case wmLButtonUp:
			a.togglePopup()
		case wmRButtonUp, 0x007B /*WM_CONTEXTMENU*/ :
			a.showMenu()
		}
		return 0

	case wmAgentLine:
		a.drainAgent()
		return 0

	case wmAgentExit:
		a.onAgentExit()
		return 0

	case wmAsyncDone:
		a.busy = false
		a.run.Rescan()
		if a.run.Path != "" && a.errMsg == t("no_agent") {
			a.errMsg = ""
		}
		a.refresh()
		return 0

	case wmPaint:
		var ps paintStruct
		hdc, _, _ := pBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		var client rect
		pGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&client)))
		a.paint(hdc, client)
		pEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		return 0

	case wmEraseBkgnd:
		return 1 // painted in full by WM_PAINT; skip the flicker

	case wmMouseMove:
		a.onMouseMove(loWord(lparam), hiWord(lparam))
		return 0

	case wmMouseLeave:
		if a.hover != hsNone {
			a.hover = hsNone
			a.invalidate()
		}
		return 0

	case wmLButtonDown:
		id, enabled := a.hitAt(loWord(lparam), hiWord(lparam))
		if enabled {
			a.pressed = id
			a.invalidate()
		}
		return 0

	case wmLButtonUp:
		id, enabled := a.hitAt(loWord(lparam), hiWord(lparam))
		if enabled && id == a.pressed {
			a.click(id)
		}
		a.pressed = hsNone
		a.invalidate()
		return 0

	case wmSetCursor:
		if _, enabled := a.hitAt2(); enabled {
			c, _, _ := pLoadCursor.Call(0, idcHand)
			pSetCursor.Call(c)
			return 1
		}

	case wmActivate:
		if wparam == 0 { // WA_INACTIVE — clicking anywhere else closes the popup
			a.hidePopup()
		}
		return 0

	case wmKeyDown:
		if wparam == 0x1B { // VK_ESCAPE
			a.hidePopup()
		}
		return 0

	case wmTimer:
		a.onTimer(uint32(wparam))
		return 0

	case wmCommand:
		a.command(uint32(loWord(wparam)))
		return 0

	case wmDPIChanged:
		a.dpi = int32(loWord(wparam))
		a.fonts.free()
		a.fonts = newFonts(a.dpi)
		a.qrMax = a.sc(196)
		a.rebuildQR(true)
		a.trayUpdate()
		a.refresh()
		return 0

	case wmDestroy:
		a.run.Stop()
		pPostQuitMessage.Call(0)
		return 0

	case wmClose:
		a.hidePopup()
		return 0
	}

	switch uint32(message) {
	case msgTaskbarCreat: // Explorer restarted — re-add our icon
		a.trayAdd()
		return 0
	case msgShowPopup: // a second launch asked us to show ourselves
		a.showPopup()
		return 0
	}

	r, _, _ := pDefWindowProc.Call(hwnd, message, wparam, lparam)
	return r
}

// ── popup show / hide ───────────────────────────────────────────────────────

func (a *App) togglePopup() {
	if a.visible {
		a.hidePopup()
	} else {
		a.showPopup()
	}
}

func (a *App) showPopup() {
	a.rebuildQR(false)
	a.relayout()
	w, h := a.lay.width, a.lay.height
	x, y := a.place(w, h)
	a.targetX, a.targetY = x, y

	// rounded window shape (Win10 has no DWM rounding for WS_POPUP)
	if rgn := roundRegion(w, h, a.sc(12)); rgn != 0 {
		setWindowRgn(a.hwnd, rgn)
	}

	if a.animate {
		a.alpha = 0
		pSetLayeredWindowAttr.Call(a.hwnd, 0, 0, lwaAlpha)
		pSetWindowPos.Call(a.hwnd, hwndTopmost, uintptr(x), uintptr(y+a.sc(8)),
			uintptr(w), uintptr(h), swpShowWindow)
		pSetTimer.Call(a.hwnd, timerFade, 15, 0)
	} else {
		a.alpha = 1
		pSetLayeredWindowAttr.Call(a.hwnd, 0, 255, lwaAlpha)
		pSetWindowPos.Call(a.hwnd, hwndTopmost, uintptr(x), uintptr(y),
			uintptr(w), uintptr(h), swpShowWindow)
	}
	pSetForegroundWindow.Call(a.hwnd)
	a.visible = true
	a.syncPulse()
	a.invalidate()
}

func (a *App) hidePopup() {
	if !a.visible {
		return
	}
	a.visible = false
	a.hover, a.pressed = hsNone, hsNone
	pKillTimer.Call(a.hwnd, timerFade)
	pKillTimer.Call(a.hwnd, timerPulse)
	pShowWindow.Call(a.hwnd, swHide)
}

// relayout re-measures the popup and resizes the window if it is on screen.
func (a *App) relayout() {
	a.measure()

	// A tall state (QR + roster + a wrapped French hint at 150% DPI) can outgrow
	// a small screen. Shrink the QR — the only elastic block — until it fits,
	// rather than letting the bottom of the popup fall off the work area.
	avail := a.workArea(a.anchorRect()).height() - a.sc(24)
	if a.lay.height > avail && a.qr != nil {
		want := a.qrMax - (a.lay.height - avail)
		if floor := a.sc(96); want < floor {
			want = floor
		}
		if want < a.qrMax {
			a.qrMax = want
			a.rebuildQR(true)
			a.measure()
		}
	}

	if a.visible {
		w, h := a.lay.width, a.lay.height
		x, y := a.place(w, h)
		a.targetX, a.targetY = x, y
		if rgn := roundRegion(w, h, a.sc(12)); rgn != 0 {
			setWindowRgn(a.hwnd, rgn)
		}
		pSetWindowPos.Call(a.hwnd, hwndTopmost, uintptr(x), uintptr(y),
			uintptr(w), uintptr(h), swpNoActivate)
	}
}

// measure rebuilds the layout on a scratch DC (DT_CALCRECT needs a DC with the
// right font selected, but nothing is drawn).
func (a *App) measure() {
	mem, _, _ := pCreateCompatibleDC.Call(0)
	d := &dc{h: mem}
	a.lay = a.buildLayout(d)
	d.release()
	pDeleteDC.Call(mem)
}

func (a *App) invalidate() {
	pInvalidateRect.Call(a.hwnd, 0, 0)
}

// refresh = state changed: re-render QR, re-measure, repaint, update the icon.
func (a *App) refresh() {
	a.rebuildQR(false)
	a.relayout()
	a.trayUpdate()
	a.syncPulse()
	a.invalidate()
}

func (a *App) rebuildQR(force bool) {
	if !a.showPairing() {
		if a.qr != nil {
			a.qr.free()
			a.qr = nil
			a.qrURL = ""
		}
		return
	}
	if !force && a.qrURL == a.sess.URL && a.qr != nil {
		return
	}
	if a.qr != nil {
		a.qr.free()
	}
	if a.qrURL != a.sess.URL {
		a.qrMax = a.sc(196) // new session: start from the preferred size again
	}
	a.qr = makeQR(a.sess.URL, a.qrMax)
	if a.qr == nil && a.sess.QRPath != "" {
		a.qr = makeQRFromPNG(a.sess.QRPath, a.qrMax)
	}
	a.qrURL = a.sess.URL
}

// syncPulse runs the breathing animation only while it means something.
func (a *App) syncPulse() {
	want := a.visible && a.animate &&
		(a.sess.Status == StatusWaiting || a.sess.Status == StatusStarting)
	if want {
		pSetTimer.Call(a.hwnd, timerPulse, 60, 0)
	} else {
		pKillTimer.Call(a.hwnd, timerPulse)
	}
}

func (a *App) onTimer(id uint32) {
	switch id {
	case timerFade:
		a.alpha += 0.12
		if a.alpha >= 1 {
			a.alpha = 1
			pKillTimer.Call(a.hwnd, timerFade)
		}
		// ease-out: opacity plus a small upward settle
		e := 1 - (1-a.alpha)*(1-a.alpha)
		pSetLayeredWindowAttr.Call(a.hwnd, 0, uintptr(byte(255*e)), lwaAlpha)
		off := int32((1 - e) * float64(a.sc(8)))
		pSetWindowPos.Call(a.hwnd, hwndTopmost, uintptr(a.targetX), uintptr(a.targetY+off),
			0, 0, swpNoSize|swpNoActivate)
	case timerPulse:
		a.pulsePhi += 0.22
		a.pulse = 0.5 + 0.5*math.Sin(a.pulsePhi)
		a.invalidate()
	case timerCopy:
		a.copied = false
		pKillTimer.Call(a.hwnd, timerCopy)
		a.invalidate()
	}
}

// ── interaction ─────────────────────────────────────────────────────────────

func (a *App) hitAt(x, y int32) (int, bool) {
	if a.lay == nil {
		return hsNone, false
	}
	return a.lay.hit(x, y)
}

// hitAt2 re-tests under the current cursor (WM_SETCURSOR carries no position).
func (a *App) hitAt2() (int, bool) {
	if a.lay == nil {
		return hsNone, false
	}
	for _, s := range a.lay.spots {
		if s.id == a.hover {
			return s.id, s.enabled
		}
	}
	return hsNone, false
}

func (a *App) onMouseMove(x, y int32) {
	id, _ := a.hitAt(x, y)
	if id != a.hover {
		a.hover = id
		a.invalidate()
	}
	tme := trackMouseEvent{
		Size:      uint32(unsafe.Sizeof(trackMouseEvent{})),
		Flags:     tmeLeave,
		HwndTrack: a.hwnd,
	}
	pTrackMouseEvent.Call(uintptr(unsafe.Pointer(&tme)))
}

func (a *App) click(id int) {
	switch id {
	case hsClose:
		a.hidePopup()
	case hsModeLan:
		a.set.Remote = false
		a.set.save()
		a.refresh()
	case hsModeRemote:
		a.set.Remote = true
		a.set.save()
		a.refresh()
	case hsRemember:
		a.set.Remember = !a.set.Remember
		a.set.save()
		a.invalidate()
	case hsStart:
		if a.run.Running() {
			a.stopSession()
		} else {
			a.startSession()
		}
	case hsCopy:
		if copyToClipboard(a.hwnd, a.sess.URL) {
			a.copied = true
			pSetTimer.Call(a.hwnd, timerCopy, 1600, 0)
			a.invalidate()
		}
	case hsInstall:
		a.installAgent()
	}
}

func (a *App) command(cmd uint32) {
	switch cmd {
	case cmdShow:
		a.showPopup()
	case cmdStartLan:
		a.set.Remote = false
		a.set.save()
		a.startSession()
	case cmdStartRemote:
		a.set.Remote = true
		a.set.save()
		a.startSession()
	case cmdStop:
		a.stopSession()
	case cmdCopy:
		copyToClipboard(a.hwnd, a.sess.URL)
	case cmdAutostart:
		setAutostart(!autostartEnabled())
	case cmdUpdate:
		a.updateAgent()
	case cmdWebsite:
		pShellExecute.Call(0, uintptr(unsafe.Pointer(u16("open"))),
			uintptr(unsafe.Pointer(u16("https://"+relayHost))), 0, 0, swShow)
	case cmdQuit:
		a.run.Stop()
		pDestroyWindow.Call(a.hwnd)
	}
}

// ── session control ─────────────────────────────────────────────────────────

func (a *App) startSession() {
	if a.run.Running() {
		return
	}
	a.errMsg = ""
	a.sess.Reset()
	notify := func() { pPostMessage.Call(a.hwnd, wmAgentLine, 0, 0) }
	onExit := func() { pPostMessage.Call(a.hwnd, wmAgentExit, 0, 0) }
	if err := a.run.Start(a.set.Remote, a.set.Remember, notify, onExit); err != nil {
		if a.run.Path == "" {
			a.errMsg = t("no_agent")
		} else {
			a.errMsg = err.Error()
		}
		a.refresh()
		return
	}
	a.sess.Status = StatusStarting
	a.refresh()
}

func (a *App) stopSession() {
	a.run.Stop()
}

func (a *App) drainAgent() {
	for {
		select {
		case line := <-a.run.Lines():
			a.sess.Handle(line)
		default:
			if a.sess.Changed {
				a.sess.Changed = false
				a.refresh()
			}
			return
		}
	}
}

func (a *App) onAgentExit() {
	a.drainAgent()
	a.sess.Reset()
	a.sess.Changed = false
	a.refresh()
}

// installAgent fetches the agent for people running the portable tray exe on a
// machine where the agent isn't installed (the installer bundles it).
func (a *App) installAgent() {
	if a.busy {
		return
	}
	a.busy = true
	a.errMsg = ""
	a.invalidate()
	go func() {
		if _, err := InstallAgent(); err != nil {
			a.errMsg = err.Error()
		}
		pPostMessage.Call(a.hwnd, wmAsyncDone, 0, 0)
	}()
}

func (a *App) updateAgent() {
	if a.busy || a.run.Path == "" {
		return
	}
	a.busy = true
	go func() {
		out, err := UpdateAgent(a.run.Path)
		title := t("title")
		body := t("update_done")
		if err != nil {
			body = t("update_fail")
			if out != "" {
				body = out
			}
		} else if out != "" {
			body = firstLine(out)
		}
		a.errMsg = ""
		pPostMessage.Call(a.hwnd, wmAsyncDone, 0, 0)
		a.trayBalloon(title, body)
	}()
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}

// ── small win32 odds and ends ───────────────────────────────────────────────

func registerMessage(name string) uint32 {
	r, _, _ := pRegisterWindowMsg.Call(uintptr(unsafe.Pointer(u16(name))))
	return uint32(r)
}

// singleInstanceTaken reports whether another tray is already running for this
// user session (the mutex handle is deliberately leaked for process lifetime).
func singleInstanceTaken() bool {
	h, err := windows.CreateMutex(nil, false, u16(`Local\ZlefRemoteTray`))
	if h == 0 {
		return false
	}
	return err == windows.ERROR_ALREADY_EXISTS
}

func makeDPIAware() {
	if pSetDpiAwarenessCtx.Find() == nil {
		if r, _, _ := pSetDpiAwarenessCtx.Call(dpiPerMonitorV2); r != 0 {
			return
		}
	}
	pSetProcessDPIAware.Call()
}

func windowDPI(hwnd uintptr) int32 {
	if pGetDpiForWindow.Find() == nil {
		if d, _, _ := pGetDpiForWindow.Call(hwnd); d >= 72 {
			return int32(d)
		}
	}
	return 96
}

// animationsEnabled mirrors Settings → Ease of Access → "Show animations".
func animationsEnabled() bool {
	var on int32
	r, _, _ := pSystemParametersInfo.Call(spiClientAnim, 0, uintptr(unsafe.Pointer(&on)), 0)
	if r == 0 {
		return true
	}
	return on != 0
}

// uiLanguage returns the user's preferred UI language as a BCP-47 tag.
func uiLanguage() string {
	if langs, err := windows.GetUserPreferredUILanguages(windows.MUI_LANGUAGE_NAME); err == nil && len(langs) > 0 {
		return langs[0]
	}
	return "en"
}

var (
	pGetModuleHandle    = kernel32.NewProc("GetModuleHandleW")
	pCreateRoundRectRgn = gdi32.NewProc("CreateRoundRectRgn")
	pSetWindowRgn       = user32.NewProc("SetWindowRgn")
)

func roundRegion(w, h, r int32) uintptr {
	rgn, _, _ := pCreateRoundRectRgn.Call(0, 0, uintptr(w+1), uintptr(h+1),
		uintptr(r*2), uintptr(r*2))
	return rgn
}

func setWindowRgn(hwnd, rgn uintptr) {
	pSetWindowRgn.Call(hwnd, rgn, 1) // the window owns the region afterwards
}

// selfDir is where a portable install keeps its agent next to the tray exe.
func selfDir() string {
	p, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(p)
}
