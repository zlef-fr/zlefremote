//go:build windows

// popup.go — the control surface that drops out of the tray icon.
//
// It is a single owner-drawn WS_POPUP window rather than a stack of native
// controls: the zlef design identity (dark surfaces, olive/grape accents,
// hairlines) fights the Win32 common controls at every step, and hand-painting
// keeps the whole UI on one DPI-aware layout pass. Everything is drawn into a
// memory DC and blitted once, so there is no flicker while it animates.
package main

import (
	"fmt"
	"unsafe"
)

// hotspot ids — the popup's clickable regions.
const (
	hsNone = iota
	hsModeLan
	hsModeRemote
	hsRemember
	hsStart
	hsCopy
	hsInstall
	hsClose
)

type hotspot struct {
	id      int
	r       rect
	enabled bool
}

// layout holds one computed frame of geometry (recomputed whenever the state
// or the DPI changes) so paint and hit-testing can never disagree.
type layout struct {
	width, height int32
	title, tag    rect
	sep           int32
	lan, remote   rect
	remember      rect
	start         rect
	status        rect
	roster        rect
	qrPanel       rect
	saved         rect
	linkCap       rect
	url           rect
	copy          rect
	install       rect
	errBox        rect
	hint          rect
	closeBtn      rect
	spots         []hotspot
}

func (a *App) sc(v int32) int32 { return v * a.dpi / 96 }

// buildLayout measures every block top-down. Blocks that don't apply to the
// current state get a zero rect and simply aren't painted.
func (a *App) buildLayout(d *dc) *layout {
	l := &layout{}
	P := a.sc(16)
	W := a.sc(320)
	l.width = W
	inner := W - 2*P
	y := P

	l.closeBtn = rect{W - P - a.sc(18), P - a.sc(2), W - P + a.sc(2), P + a.sc(18)}
	l.title = rect{P, y, W - P - a.sc(24), y + a.sc(22)}
	y += a.sc(22)
	l.tag = rect{P, y, W - P, y + a.sc(17)}
	y += a.sc(17) + a.sc(12)

	l.sep = y
	y += a.sc(12)

	modeH := a.sc(46)
	l.lan = rect{P, y, W - P, y + modeH}
	y += modeH + a.sc(8)
	l.remote = rect{P, y, W - P, y + modeH}
	y += modeH + a.sc(10)

	// The "remember" description wraps (it is two lines in French at 150% DPI),
	// so the row is measured, never assumed.
	remIndent := a.sc(16) + a.sc(10)
	remH := a.sc(18) + d.measure(a.rememberDesc(), a.fonts.small, inner-remIndent,
		dtWordBreak|dtNoPrefix)
	if min := a.sc(40); remH < min {
		remH = min
	}
	l.remember = rect{P, y, W - P, y + remH}
	y += remH + a.sc(12)

	if a.run.Path == "" {
		// no agent: the primary action becomes "install it for me"
		l.install = rect{P, y, W - P, y + a.sc(38)}
		y += a.sc(38) + a.sc(10)
	} else {
		l.start = rect{P, y, W - P, y + a.sc(38)}
		y += a.sc(38) + a.sc(10)
	}

	statusTxt := a.statusText()
	sh := d.measure(statusTxt, a.fonts.body, inner, dtCenter|dtWordBreak|dtNoPrefix)
	l.status = rect{P, y, W - P, y + sh}
	y += sh + a.sc(6)

	if n := len(a.sess.Peers); n > 0 {
		rh := a.sc(18) + int32(n)*a.sc(15)
		l.roster = rect{P, y, W - P, y + rh}
		y += rh + a.sc(6)
	}

	if a.showPairing() {
		pad := a.sc(8)
		side := a.qrSide() + 2*pad
		x0 := (W - side) / 2
		l.qrPanel = rect{x0, y, x0 + side, y + side}
		y += side + a.sc(10)

		if a.sess.Persistent {
			h := d.measure(t("saved_hint"), a.fonts.small, inner, dtCenter|dtWordBreak|dtNoPrefix)
			l.saved = rect{P, y, W - P, y + h}
			y += h + a.sc(8)
		}
		h := d.measure(t("open_phone"), a.fonts.small, inner, dtCenter|dtWordBreak|dtNoPrefix)
		l.linkCap = rect{P, y, W - P, y + h}
		y += h + a.sc(5)

		l.url = rect{P, y, W - P, y + a.sc(28)}
		y += a.sc(28) + a.sc(8)
		l.copy = rect{P, y, W - P, y + a.sc(32)}
		y += a.sc(32) + a.sc(4)
	}

	if a.errMsg != "" {
		h := d.measure(a.errMsg, a.fonts.small, inner, dtCenter|dtWordBreak|dtNoPrefix)
		l.errBox = rect{P, y, W - P, y + h}
		y += h + a.sc(6)
	}

	hh := d.measure(t("hint_tray"), a.fonts.small, inner, dtCenter|dtNoPrefix)
	l.hint = rect{P, y + a.sc(2), W - P, y + a.sc(2) + hh}
	y += hh + a.sc(2)

	l.height = y + P

	running := a.run.Running()
	l.spots = []hotspot{
		{hsClose, l.closeBtn, true},
		{hsModeLan, l.lan, !running},
		{hsModeRemote, l.remote, !running},
		{hsRemember, l.remember, !running && a.set.Remote && a.run.Remember},
		{hsStart, l.start, a.run.Path != ""},
		{hsCopy, l.copy, a.sess.URL != ""},
		{hsInstall, l.install, !a.busy},
	}
	return l
}

func (a *App) showPairing() bool {
	return (a.sess.Status == StatusWaiting || a.sess.Status == StatusPaired) && a.sess.URL != ""
}

func (a *App) qrSide() int32 {
	if a.qr != nil {
		return a.qr.size
	}
	return a.sc(180)
}

func (a *App) statusText() string {
	if a.errMsg != "" && a.sess.Status == StatusIdle {
		return t("st_idle")
	}
	switch a.sess.Status {
	case StatusStarting:
		return t("st_starting")
	case StatusWaiting:
		return t("st_waiting")
	case StatusPaired:
		return t("st_paired")
	}
	return t("st_idle")
}

// ── painting ────────────────────────────────────────────────────────────────

func (a *App) paint(hdc uintptr, client rect) {
	// double buffer: everything lands in a memory bitmap, then one blit
	mem, _, _ := pCreateCompatibleDC.Call(hdc)
	bmp, _, _ := pCreateCompatibleBM.Call(hdc, uintptr(client.width()), uintptr(client.height()))
	oldBmp, _, _ := pSelectObject.Call(mem, bmp)
	d := &dc{h: mem}
	defer func() {
		d.release()
		pSelectObject.Call(mem, oldBmp)
		pDeleteObject.Call(bmp)
		pDeleteDC.Call(mem)
	}()

	l := a.lay
	if l == nil {
		l = a.buildLayout(d)
		a.lay = l
	}

	// body + hairline border
	d.roundRect(client, a.sc(12), colBg, colLine, true)

	// header
	d.text(t("title"), l.title, a.fonts.title, colText, dtLeft|dtSingleLine|dtNoPrefix)
	d.text(t("tagline"), l.tag, a.fonts.body, colTextMuted, dtLeft|dtSingleLine|dtEndEllipsis|dtNoPrefix)
	a.paintClose(d, l)
	d.line(l.title.Left, l.sep, client.Right-a.sc(16), l.sep, colLine, 1)

	// mode cards
	a.paintMode(d, l.lan, t("mode_lan"), t("mode_lan_d"), !a.set.Remote, hsModeLan)
	a.paintMode(d, l.remote, t("mode_remote"), t("mode_remote_d"), a.set.Remote, hsModeRemote)

	// remember
	a.paintRemember(d, l.remember)

	// primary action
	if l.install.Bottom > 0 {
		label := t("no_agent")
		if a.busy {
			label = t("st_starting")
		}
		a.paintButton(d, l.install, label, hsInstall, true)
	} else {
		label := t("start")
		if a.run.Running() {
			label = t("stop")
		}
		a.paintButton(d, l.start, label, hsStart, true)
	}

	// status + activity pulse
	col := colTextSoft
	if a.sess.Status == StatusPaired {
		col = colGrapeSoft
	} else if a.sess.Status == StatusWaiting {
		col = colAccentSft
	}
	d.text(a.statusText(), l.status, a.fonts.body, col, dtCenter|dtWordBreak|dtNoPrefix)

	a.paintRoster(d, l)

	if a.showPairing() {
		a.paintQR(d, l)
		if a.sess.Persistent {
			d.text(t("saved_hint"), l.saved, a.fonts.small, colTextMuted, dtCenter|dtWordBreak|dtNoPrefix)
		}
		d.text(t("open_phone"), l.linkCap, a.fonts.small, colTextMuted, dtCenter|dtWordBreak|dtNoPrefix)
		d.roundRect(l.url, a.sc(6), colSurface2, colLine, true)
		ur := l.url
		ur.Left += a.sc(8)
		ur.Right -= a.sc(8)
		d.text(a.sess.URL, ur, a.fonts.mono, colTextSoft,
			dtLeft|dtSingleLine|dtVCenter|dtEndEllipsis|dtNoPrefix)
		lbl := t("copy")
		if a.copied {
			lbl = t("copied")
		}
		a.paintButton(d, l.copy, lbl, hsCopy, false)
	}

	if a.errMsg != "" {
		d.text(a.errMsg, l.errBox, a.fonts.small, colError, dtCenter|dtWordBreak|dtNoPrefix)
	}
	d.text(t("hint_tray"), l.hint, a.fonts.small, colTextFaint, dtCenter|dtNoPrefix)

	pBitBlt.Call(hdc, 0, 0, uintptr(client.width()), uintptr(client.height()), mem, 0, 0, srcCopy)
}

func (a *App) paintClose(d *dc, l *layout) {
	c := colTextFaint
	if a.hover == hsClose {
		c = colText
	}
	r := l.closeBtn
	m := a.sc(4)
	d.line(r.Left+m, r.Top+m, r.Right-m, r.Bottom-m, c, a.sc(1)+1)
	d.line(r.Right-m, r.Top+m, r.Left+m, r.Bottom-m, c, a.sc(1)+1)
}

func (a *App) paintMode(d *dc, r rect, title, desc string, on bool, id int) {
	enabled := !a.run.Running()
	bg := uint32(colSurface2)
	border := uint32(colLine)
	if on {
		bg = mix(colSurface2, colAccent, 0.45)
		border = colAccentMid
	} else if a.hover == id && enabled {
		bg = colSurface3
	}
	d.roundRect(r, a.sc(10), bg, border, true)

	// radio
	cy := (r.Top + r.Bottom) / 2
	rad := a.sc(7)
	cx := r.Left + a.sc(14)
	ring := uint32(colTextMuted)
	if on {
		ring = colAccentBr
	}
	d.ellipse(rect{cx - rad, cy - rad, cx + rad, cy + rad}, bg, ring, true, a.sc(1)+1)
	if on {
		ir := a.sc(3)
		d.ellipse(rect{cx - ir, cy - ir, cx + ir, cy + ir}, colAccentBr, 0, false, 0)
	}

	tx := cx + rad + a.sc(10)
	tcol := uint32(colText)
	dcol := uint32(colTextMuted)
	if !enabled {
		tcol, dcol = colTextMuted, colTextFaint
	}
	d.text(title, rect{tx, r.Top + a.sc(7), r.Right - a.sc(8), r.Top + a.sc(24)},
		a.fonts.bodyB, tcol, dtLeft|dtSingleLine|dtEndEllipsis|dtNoPrefix)
	d.text(desc, rect{tx, r.Top + a.sc(24), r.Right - a.sc(8), r.Bottom - a.sc(4)},
		a.fonts.small, dcol, dtLeft|dtSingleLine|dtEndEllipsis|dtNoPrefix)
}

func (a *App) paintRemember(d *dc, r rect) {
	enabled := !a.run.Running() && a.set.Remote && a.run.Remember
	box := a.sc(16)
	cy := r.Top + a.sc(11)
	br := rect{r.Left, cy - box/2, r.Left + box, cy + box/2}

	fill := uint32(colSurface2)
	border := uint32(colLine)
	if a.set.Remember && enabled {
		fill = colAccentMid
		border = colAccentBr
	}
	if !enabled {
		fill = colBg
	}
	d.roundRect(br, a.sc(4), fill, border, true)
	if a.set.Remember {
		// checkmark
		c := uint32(colOnAccent)
		if !enabled {
			c = colTextFaint
		}
		d.line(br.Left+a.sc(4), cy, br.Left+box/2, br.Bottom-a.sc(4), c, a.sc(1)+1)
		d.line(br.Left+box/2, br.Bottom-a.sc(4), br.Right-a.sc(3), br.Top+a.sc(4), c, a.sc(1)+1)
	}

	tcol, dcol := uint32(colText), uint32(colTextMuted)
	if !enabled {
		tcol, dcol = colTextFaint, colTextFaint
	}
	tx := br.Right + a.sc(10)
	d.text(t("remember"), rect{tx, r.Top, r.Right, r.Top + a.sc(18)},
		a.fonts.body, tcol, dtLeft|dtSingleLine|dtEndEllipsis|dtNoPrefix)
	d.text(a.rememberDesc(), rect{tx, r.Top + a.sc(18), r.Right, r.Bottom},
		a.fonts.small, dcol, dtLeft|dtWordBreak|dtNoPrefix)
}

// rememberDesc explains the checkbox, or why it is unavailable on an agent too
// old to understand -remember.
func (a *App) rememberDesc() string {
	if !a.run.Remember && a.run.Path != "" {
		return t("remember_old")
	}
	return t("remember_d")
}

func (a *App) paintButton(d *dc, r rect, label string, id int, primary bool) {
	if r.Bottom == 0 {
		return
	}
	enabled := true
	for _, s := range a.lay.spots {
		if s.id == id {
			enabled = s.enabled
		}
	}
	var bg, fg, border uint32
	switch {
	case !enabled:
		bg, fg, border = colSurface2, colTextFaint, colLine
	case primary:
		bg, fg, border = colAccent, colOnAccent, colAccentMid
		if a.hover == id {
			bg = colAccentMid
		}
		if a.pressed == id {
			bg = mix(colAccent, 0x000000, 0.2)
		}
	default:
		bg, fg, border = colSurface2, colText, colLine
		if a.hover == id {
			bg = colSurface3
		}
		if a.pressed == id {
			bg = colSurface2
		}
	}
	d.roundRect(r, a.sc(9), bg, border, true)
	d.text(label, r, a.fonts.button, fg, dtCenter|dtVCenter|dtSingleLine|dtNoPrefix)
}

func (a *App) paintRoster(d *dc, l *layout) {
	peers := a.sess.PeerList()
	if len(peers) == 0 || l.roster.Bottom == 0 {
		return
	}
	head := t("clients_1")
	if len(peers) > 1 {
		head = fmt.Sprintf(t("clients_n"), len(peers))
	}
	y := l.roster.Top
	d.text(head, rect{l.roster.Left, y, l.roster.Right, y + a.sc(18)},
		a.fonts.bodyB, colText, dtCenter|dtSingleLine|dtNoPrefix)
	y += a.sc(18)
	for _, p := range peers {
		ip := p.IP
		if ip == "" {
			ip = t("unknown_ip")
		}
		d.text(ip, rect{l.roster.Left, y, l.roster.Right, y + a.sc(15)},
			a.fonts.small, colTextMuted, dtCenter|dtSingleLine|dtEndEllipsis|dtNoPrefix)
		y += a.sc(15)
	}
}

func (a *App) paintQR(d *dc, l *layout) {
	// white plate: dark themes would otherwise sink the QR's contrast
	plate := colWhite
	if a.sess.Status == StatusPaired {
		// dimmed while in use — mirrors the plugin's 40%-opacity "in use" cue
		plate = uint32(0xcfd0c9)
	}
	d.roundRect(l.qrPanel, a.sc(8), plate, 0, false)

	// waiting: a soft accent ring that breathes, so the popup never looks frozen
	if a.sess.Status == StatusWaiting && a.animate {
		f := 0.35 + 0.35*a.pulse
		ring := mix(colBg, colAccentSft, f)
		g := l.qrPanel
		g.Left -= a.sc(3)
		g.Top -= a.sc(3)
		g.Right += a.sc(3)
		g.Bottom += a.sc(3)
		d.roundRect(g, a.sc(10), colBg, ring, true)
		d.roundRect(l.qrPanel, a.sc(8), plate, 0, false)
	}

	if a.qr != nil {
		pad := a.sc(8)
		blit(d, a.qr.hbm, l.qrPanel.Left+pad, l.qrPanel.Top+pad, a.qr.size, a.qr.size)
	}
}

// ── hit testing ─────────────────────────────────────────────────────────────

func (l *layout) hit(x, y int32) (int, bool) {
	for _, s := range l.spots {
		if s.r.Bottom == 0 && s.r.Top == 0 {
			continue
		}
		if x >= s.r.Left && x < s.r.Right && y >= s.r.Top && y < s.r.Bottom {
			return s.id, s.enabled
		}
	}
	return hsNone, false
}

// ── window geometry ─────────────────────────────────────────────────────────

// anchorRect is where the popup hangs from: the tray icon's own rectangle when
// the shell will tell us (Win7+), the cursor otherwise.
func (a *App) anchorRect() rect {
	var anchor rect
	id := notifyIconIdentifier{
		CbSize: uint32(unsafe.Sizeof(notifyIconIdentifier{})),
		HWnd:   a.hwnd,
		UID:    trayUID,
	}
	ok, _, _ := pShellNotifyIconRect.Call(uintptr(unsafe.Pointer(&id)),
		uintptr(unsafe.Pointer(&anchor)))
	if ok != 0 { // S_OK == 0; anything else means "no rect", fall back to cursor
		var pt point
		pGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
		anchor = rect{pt.X, pt.Y, pt.X, pt.Y}
	}
	return anchor
}

// workArea is the usable area of the monitor the tray icon lives on.
func (a *App) workArea(anchor rect) rect {
	mon, _, _ := pMonitorFromPoint.Call(uintptr(anchor.Left), uintptr(anchor.Top), monitorNearest)
	mi := monitorInfo{Size: uint32(unsafe.Sizeof(monitorInfo{}))}
	if mon != 0 {
		if r, _, _ := pGetMonitorInfo.Call(mon, uintptr(unsafe.Pointer(&mi))); r != 0 {
			return mi.RcWork
		}
	}
	return rect{0, 0, 1920, 1080}
}

// place positions the popup next to the tray icon, clamped to the work area of
// whichever monitor the icon sits on (multi-monitor + taskbar on any edge).
func (a *App) place(w, h int32) (int32, int32) {
	anchor := a.anchorRect()
	margin := a.sc(8)
	work := a.workArea(anchor)

	x := anchor.Right - w
	y := anchor.Top - h - margin
	if anchor.Top < work.Top+(work.height()/2) { // taskbar at the top
		y = anchor.Bottom + margin
	}
	if anchor.Left < work.Left+(work.width()/2) && anchor.Right-anchor.Left < w {
		// taskbar on the left edge
		if anchor.Right+w+margin < work.Right {
			x = anchor.Right + margin
		}
	}
	if x+w > work.Right-margin {
		x = work.Right - w - margin
	}
	if x < work.Left+margin {
		x = work.Left + margin
	}
	if y+h > work.Bottom-margin {
		y = work.Bottom - h - margin
	}
	if y < work.Top+margin {
		y = work.Top + margin
	}
	return x, y
}
