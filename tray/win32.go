//go:build windows

// win32.go — the thin syscall layer. Everything the tray needs from user32 /
// gdi32 / shell32 / kernel32, and nothing else. Pure syscalls (no cgo) so the
// whole app cross-compiles from Linux with CGO_ENABLED=0.
package main

import (
	"golang.org/x/sys/windows"
)

var (
	user32   = windows.NewLazySystemDLL("user32.dll")
	gdi32    = windows.NewLazySystemDLL("gdi32.dll")
	shell32  = windows.NewLazySystemDLL("shell32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")

	pRegisterClassEx      = user32.NewProc("RegisterClassExW")
	pCreateWindowEx       = user32.NewProc("CreateWindowExW")
	pDefWindowProc        = user32.NewProc("DefWindowProcW")
	pDestroyWindow        = user32.NewProc("DestroyWindow")
	pShowWindow           = user32.NewProc("ShowWindow")
	pUpdateWindow         = user32.NewProc("UpdateWindow")
	pGetMessage           = user32.NewProc("GetMessageW")
	pTranslateMessage     = user32.NewProc("TranslateMessage")
	pDispatchMessage      = user32.NewProc("DispatchMessageW")
	pPostQuitMessage      = user32.NewProc("PostQuitMessage")
	pPostMessage          = user32.NewProc("PostMessageW")
	pLoadCursor           = user32.NewProc("LoadCursorW")
	pSetCursor            = user32.NewProc("SetCursor")
	pSetForegroundWindow  = user32.NewProc("SetForegroundWindow")
	pGetCursorPos         = user32.NewProc("GetCursorPos")
	pCreatePopupMenu      = user32.NewProc("CreatePopupMenu")
	pAppendMenu           = user32.NewProc("AppendMenuW")
	pDestroyMenu          = user32.NewProc("DestroyMenu")
	pTrackPopupMenu       = user32.NewProc("TrackPopupMenu")
	pInvalidateRect       = user32.NewProc("InvalidateRect")
	pBeginPaint           = user32.NewProc("BeginPaint")
	pEndPaint             = user32.NewProc("EndPaint")
	pFillRect             = user32.NewProc("FillRect")
	pDrawText             = user32.NewProc("DrawTextW")
	pSetTimer             = user32.NewProc("SetTimer")
	pKillTimer            = user32.NewProc("KillTimer")
	pGetClientRect        = user32.NewProc("GetClientRect")
	pSetWindowPos         = user32.NewProc("SetWindowPos")
	pSetLayeredWindowAttr = user32.NewProc("SetLayeredWindowAttributes")
	pTrackMouseEvent      = user32.NewProc("TrackMouseEvent")
	pMonitorFromPoint     = user32.NewProc("MonitorFromPoint")
	pGetMonitorInfo       = user32.NewProc("GetMonitorInfoW")
	pSystemParametersInfo = user32.NewProc("SystemParametersInfoW")
	pGetDpiForWindow      = user32.NewProc("GetDpiForWindow")
	pSetDpiAwarenessCtx   = user32.NewProc("SetProcessDpiAwarenessContext")
	pSetProcessDPIAware   = user32.NewProc("SetProcessDPIAware")
	pCreateIconIndirect   = user32.NewProc("CreateIconIndirect")
	pDestroyIcon          = user32.NewProc("DestroyIcon")
	pRegisterWindowMsg    = user32.NewProc("RegisterWindowMessageW")
	pOpenClipboard        = user32.NewProc("OpenClipboard")
	pCloseClipboard       = user32.NewProc("CloseClipboard")
	pEmptyClipboard       = user32.NewProc("EmptyClipboard")
	pSetClipboardData     = user32.NewProc("SetClipboardData")
	pMessageBox           = user32.NewProc("MessageBoxW")
	pGetSystemMetrics     = user32.NewProc("GetSystemMetrics")

	pCreateSolidBrush   = gdi32.NewProc("CreateSolidBrush")
	pCreatePen          = gdi32.NewProc("CreatePen")
	pSelectObject       = gdi32.NewProc("SelectObject")
	pDeleteObject       = gdi32.NewProc("DeleteObject")
	pRoundRect          = gdi32.NewProc("RoundRect")
	pRectangle          = gdi32.NewProc("Rectangle")
	pEllipse            = gdi32.NewProc("Ellipse")
	pMoveToEx           = gdi32.NewProc("MoveToEx")
	pLineTo             = gdi32.NewProc("LineTo")
	pSetBkMode          = gdi32.NewProc("SetBkMode")
	pSetTextColor       = gdi32.NewProc("SetTextColor")
	pCreateFontIndirect = gdi32.NewProc("CreateFontIndirectW")
	pCreateCompatibleDC = gdi32.NewProc("CreateCompatibleDC")
	pCreateCompatibleBM = gdi32.NewProc("CreateCompatibleBitmap")
	pCreateDIBSection   = gdi32.NewProc("CreateDIBSection")
	pCreateBitmap       = gdi32.NewProc("CreateBitmap")
	pDeleteDC           = gdi32.NewProc("DeleteDC")
	pBitBlt             = gdi32.NewProc("BitBlt")
	pGetStockObject     = gdi32.NewProc("GetStockObject")
	pGetDeviceCaps      = gdi32.NewProc("GetDeviceCaps")

	pShellNotifyIcon     = shell32.NewProc("Shell_NotifyIconW")
	pShellNotifyIconRect = shell32.NewProc("Shell_NotifyIconGetRect")
	pShellExecute        = shell32.NewProc("ShellExecuteW")

	pGlobalAlloc   = kernel32.NewProc("GlobalAlloc")
	pRtlMoveMemory = kernel32.NewProc("RtlMoveMemory")
	pGlobalLock    = kernel32.NewProc("GlobalLock")
	pGlobalUnlock  = kernel32.NewProc("GlobalUnlock")
)

// ── constants ───────────────────────────────────────────────────────────────

const (
	wsPopup      = 0x80000000
	wsExToolWin  = 0x00000080
	wsExTopmost  = 0x00000008
	wsExLayered  = 0x00080000
	wsExNoActive = 0x08000000

	csDropShadow = 0x00020000
	csHRedraw    = 0x0002
	csVRedraw    = 0x0001

	wmDestroy      = 0x0002
	wmPaint        = 0x000F
	wmClose        = 0x0010
	wmEraseBkgnd   = 0x0014
	wmActivate     = 0x0006
	wmActivateApp  = 0x001C
	wmSetCursor    = 0x0020
	wmKeyDown      = 0x0100
	wmCommand      = 0x0111
	wmTimer        = 0x0113
	wmMouseMove    = 0x0200
	wmLButtonDown  = 0x0201
	wmLButtonUp    = 0x0202
	wmRButtonUp    = 0x0205
	wmMouseLeave   = 0x02A3
	wmDPIChanged   = 0x02E0
	wmApp          = 0x8000
	wmKillFocus    = 0x0008
	wmDisplayChang = 0x007E

	swHide   = 0
	swShow   = 5
	swShowNA = 8

	nimAdd        = 0x00000000
	nimModify     = 0x00000001
	nimDelete     = 0x00000002
	nimSetVersion = 0x00000004

	nifMessage  = 0x00000001
	nifIcon     = 0x00000002
	nifTip      = 0x00000004
	nifInfo     = 0x00000010
	nifShowTip  = 0x00000080
	niifNone    = 0x00000000
	niifUser    = 0x00000004
	notifyIcoV4 = 4

	tpmRightButton = 0x0002
	tpmReturnCmd   = 0x0100
	tpmLeftAlign   = 0x0000
	tpmBottomAlign = 0x0020

	mfString    = 0x00000000
	mfSeparator = 0x00000800
	mfChecked   = 0x00000008
	mfGrayed    = 0x00000001

	lwaAlpha = 0x00000002

	swpNoSize     = 0x0001
	swpNoMove     = 0x0002
	swpNoZOrder   = 0x0004
	swpNoActivate = 0x0010
	swpShowWindow = 0x0040
	hwndTopmost   = ^uintptr(0) // (HWND)-1

	idcArrow = 32512
	idcHand  = 32649

	dtLeft        = 0x00000000
	dtCenter      = 0x00000001
	dtVCenter     = 0x00000004
	dtSingleLine  = 0x00000020
	dtWordBreak   = 0x00000010
	dtCalcRect    = 0x00000400
	dtEndEllipsis = 0x00008000
	dtNoPrefix    = 0x00000800

	transparent = 1
	biRGB       = 0
	dibRGBCol   = 0
	srcCopy     = 0x00CC0020

	cfUnicodeText = 13
	gmemMoveable  = 0x0002

	monitorNearest = 0x00000002
	spiClientAnim  = 0x1042

	nullPen   = 8
	nullBrush = 5

	logPixelsX = 88

	dpiPerMonitorV2 = ^uintptr(3) // (DPI_AWARENESS_CONTEXT)-4

	mbIconInfo = 0x00000040
	mbIconWarn = 0x00000030

	smCXScreen = 0
	smCYScreen = 1
)

// ── structs ─────────────────────────────────────────────────────────────────

type point struct{ X, Y int32 }

type rect struct{ Left, Top, Right, Bottom int32 }

func (r rect) width() int32  { return r.Right - r.Left }
func (r rect) height() int32 { return r.Bottom - r.Top }

type msg struct {
	HWnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
	Private uint32
}

type wndClassEx struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
	IconSm     uintptr
}

type paintStruct struct {
	Hdc         uintptr
	Erase       int32
	RcPaint     rect
	Restore     int32
	IncUpdate   int32
	RgbReserved [32]byte
}

type notifyIconData struct {
	CbSize           uint32
	HWnd             uintptr
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            uintptr
	SzTip            [128]uint16
	DwState          uint32
	DwStateMask      uint32
	SzInfo           [256]uint16
	UVersion         uint32
	SzInfoTitle      [64]uint16
	DwInfoFlags      uint32
	GuidItem         windows.GUID
	HBalloonIcon     uintptr
}

type notifyIconIdentifier struct {
	CbSize   uint32
	HWnd     uintptr
	UID      uint32
	GuidItem windows.GUID
}

type trackMouseEvent struct {
	Size      uint32
	Flags     uint32
	HwndTrack uintptr
	HoverTime uint32
}

const tmeLeave = 0x00000002

type monitorInfo struct {
	Size      uint32
	RcMonitor rect
	RcWork    rect
	Flags     uint32
}

type logFont struct {
	Height         int32
	Width          int32
	Escapement     int32
	Orientation    int32
	Weight         int32
	Italic         byte
	Underline      byte
	StrikeOut      byte
	CharSet        byte
	OutPrecision   byte
	ClipPrecision  byte
	Quality        byte
	PitchAndFamily byte
	FaceName       [32]uint16
}

type bitmapInfoHeader struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

type bitmapInfo struct {
	Header bitmapInfoHeader
	Colors [1]uint32
}

type iconInfo struct {
	FIcon    int32
	XHotspot uint32
	YHotspot uint32
	HbmMask  uintptr
	HbmColor uintptr
}

// ── helpers ─────────────────────────────────────────────────────────────────

func u16(s string) *uint16 {
	p, err := windows.UTF16PtrFromString(s)
	if err != nil {
		p, _ = windows.UTF16PtrFromString("")
	}
	return p
}

// copyU16 fills a fixed-size UTF-16 field (szTip, szInfo…), always
// NUL-terminated even when the string has to be cut.
func copyU16(dst []uint16, s string) {
	src := windows.StringToUTF16(s) // NUL-terminated
	if len(src) > len(dst) {
		src = src[:len(dst)]
		src[len(src)-1] = 0
	}
	copy(dst, src)
}

func loWord(v uintptr) int32 { return int32(int16(v & 0xFFFF)) }
func hiWord(v uintptr) int32 { return int32(int16((v >> 16) & 0xFFFF)) }

// rgb packs a Go-side color into GDI's COLORREF (0x00BBGGRR).
func rgb(r, g, b byte) uintptr { return uintptr(r) | uintptr(g)<<8 | uintptr(b)<<16 }

func hexColor(h uint32) uintptr {
	r := byte(h >> 16)
	g := byte(h >> 8)
	b := byte(h)
	return rgb(r, g, b)
}

// mix blends two 0xRRGGBB colors (f = 0 → a, 1 → b).
func mix(a, b uint32, f float64) uint32 {
	ch := func(sh uint) uint32 {
		x := float64((a>>sh)&0xFF)*(1-f) + float64((b>>sh)&0xFF)*f
		return uint32(x + 0.5)
	}
	return ch(16)<<16 | ch(8)<<8 | ch(0)
}
