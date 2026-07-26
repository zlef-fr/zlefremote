//go:build windows

// tray.go — the notification-area icon: tooltip, balloons and the right-click
// menu. The icon is the app's whole presence (there is no taskbar button and no
// window in Alt-Tab), so it also carries the session state as a coloured dot.
package main

import (
	"unsafe"
)

const trayUID = 0x5A52 // 'ZR'

// private window messages
const (
	wmTrayCallback = wmApp + 1
	wmAgentLine    = wmApp + 2
	wmAgentExit    = wmApp + 3
	wmAsyncDone    = wmApp + 4
	wmShowPopup    = wmApp + 5 // posted by a second instance
)

// menu command ids
const (
	cmdShow = 100 + iota
	cmdStartLan
	cmdStartRemote
	cmdStop
	cmdCopy
	cmdAutostart
	cmdUpdate
	cmdWebsite
	cmdQuit
)

// timer ids
const (
	timerFade  = 1
	timerPulse = 2
	timerCopy  = 3
)

func (a *App) trayAdd() {
	nid := a.trayData(nifMessage | nifIcon | nifTip | nifShowTip)
	pShellNotifyIcon.Call(nimAdd, uintptr(unsafe.Pointer(&nid)))
	// opt into the v4 callback shape (cursor coords in wParam)
	nid.UVersion = notifyIcoV4
	pShellNotifyIcon.Call(nimSetVersion, uintptr(unsafe.Pointer(&nid)))
}

func (a *App) trayUpdate() {
	old := a.hicon
	a.hicon = makeIcon(a.trayIconPx(), a.sess.Status)
	nid := a.trayData(nifMessage | nifIcon | nifTip | nifShowTip)
	pShellNotifyIcon.Call(nimModify, uintptr(unsafe.Pointer(&nid)))
	if old != 0 {
		pDestroyIcon.Call(old)
	}
}

func (a *App) trayRemove() {
	nid := a.trayData(0)
	pShellNotifyIcon.Call(nimDelete, uintptr(unsafe.Pointer(&nid)))
	if a.hicon != 0 {
		pDestroyIcon.Call(a.hicon)
		a.hicon = 0
	}
}

// trayBalloon raises a toast. Used for the two moments that matter while the
// popup is closed: a phone pairing, and that phone going away.
func (a *App) trayBalloon(title, body string) {
	nid := a.trayData(nifInfo | nifIcon)
	copyU16(nid.SzInfoTitle[:], title)
	copyU16(nid.SzInfo[:], body)
	nid.DwInfoFlags = niifUser // use our own icon in the toast
	pShellNotifyIcon.Call(nimModify, uintptr(unsafe.Pointer(&nid)))
}

func (a *App) trayData(flags uint32) notifyIconData {
	nid := notifyIconData{
		CbSize:           uint32(unsafe.Sizeof(notifyIconData{})),
		HWnd:             a.hwnd,
		UID:              trayUID,
		UFlags:           flags,
		UCallbackMessage: wmTrayCallback,
		HIcon:            a.hicon,
	}
	copyU16(nid.SzTip[:], a.tooltip())
	return nid
}

func (a *App) tooltip() string {
	switch a.sess.Status {
	case StatusStarting:
		return t("tip_starting")
	case StatusWaiting:
		return t("tip_waiting")
	case StatusPaired:
		return t("tip_paired")
	}
	return t("tip_idle")
}

// trayIconPx asks for the icon size the shell wants at the current DPI.
func (a *App) trayIconPx() int {
	px := 16 * int(a.dpi) / 96
	switch {
	case px <= 16:
		return 16
	case px <= 24:
		return 24
	case px <= 32:
		return 32
	case px <= 48:
		return 48
	}
	return 64
}

// ── context menu ────────────────────────────────────────────────────────────

func (a *App) showMenu() {
	menu, _, _ := pCreatePopupMenu.Call()
	if menu == 0 {
		return
	}
	defer pDestroyMenu.Call(menu)

	add := func(flags uintptr, id uintptr, label string) {
		pAppendMenu.Call(menu, flags, id, uintptr(unsafe.Pointer(u16(label))))
	}
	sep := func() { pAppendMenu.Call(menu, mfSeparator, 0, 0) }

	add(mfString, cmdShow, t("menu_show"))
	sep()
	if a.run.Running() {
		add(mfString, cmdStop, t("menu_stop"))
		if a.sess.URL != "" {
			add(mfString, cmdCopy, t("menu_copy"))
		}
	} else {
		f := uintptr(mfString)
		if a.run.Path == "" {
			f |= mfGrayed
		}
		add(f, cmdStartLan, t("menu_start_lan"))
		add(f, cmdStartRemote, t("menu_start_rem"))
	}
	sep()
	autoFlags := uintptr(mfString)
	if autostartEnabled() {
		autoFlags |= mfChecked
	}
	add(autoFlags, cmdAutostart, t("menu_autostart"))
	updFlags := uintptr(mfString)
	if a.run.Path == "" || a.run.Running() {
		updFlags |= mfGrayed
	}
	add(updFlags, cmdUpdate, t("menu_update"))
	add(mfString, cmdWebsite, t("menu_website"))
	sep()
	add(mfString, cmdQuit, t("menu_quit"))

	var pt point
	pGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	// required so the menu dismisses when the user clicks elsewhere
	pSetForegroundWindow.Call(a.hwnd)
	cmd, _, _ := pTrackPopupMenu.Call(menu,
		tpmRightButton|tpmReturnCmd|tpmLeftAlign|tpmBottomAlign,
		uintptr(pt.X), uintptr(pt.Y), 0, a.hwnd, 0)
	pPostMessage.Call(a.hwnd, 0 /*WM_NULL*/, 0, 0)
	if cmd != 0 {
		a.command(uint32(cmd))
	}
}
