//go:build windows

// theme.go — zlef design identity (da.zlef.fr tokens) expressed in GDI, plus
// the small drawing helpers the popup paints itself with. Dark only; ZLEF has
// no light theme, so the popup keeps its own surface instead of following the
// Windows light/dark setting.
package main

import "unsafe"

// Tokens (subset of da.zlef.fr/tokens.css). Typed so every drawing helper
// takes the same colour type.
const (
	colBg        uint32 = 0x0e0e13 // --zl-surface-1 (the popup body)
	colSurface2  uint32 = 0x15151c // --zl-surface-2 (inputs, nested panels)
	colSurface3  uint32 = 0x1d1d25 // --zl-surface-3 (hover)
	colText      uint32 = 0xe9eae2 // --zl-text
	colTextSoft  uint32 = 0xb6b7ad // --zl-text-soft
	colTextMuted uint32 = 0x7d7e76 // --zl-text-muted
	colTextFaint uint32 = 0x54554f // --zl-text-faint
	colLine      uint32 = 0x24242b // --zl-line over surface-1
	colAccent    uint32 = 0x3e4618 // olive leaf
	colAccentMid uint32 = 0x59642a
	colAccentSft uint32 = 0x9dae50
	colAccentBr  uint32 = 0xbdce74
	colOnAccent  uint32 = 0xeef0e2
	colGrapeSoft uint32 = 0xb095b3
	colError     uint32 = 0xd88b8b // --zl-danger-soft (inline failures)
	colWhite     uint32 = 0xffffff
)

// dc wraps a device context with the GDI object bookkeeping GDI won't do.
type dc struct {
	h     uintptr
	stock []uintptr // objects to delete on release
}

func (d *dc) del(obj uintptr) uintptr {
	if obj != 0 {
		d.stock = append(d.stock, obj)
	}
	return obj
}

func (d *dc) release() {
	for _, o := range d.stock {
		pDeleteObject.Call(o)
	}
	d.stock = nil
}

func (d *dc) brush(c uint32) uintptr {
	h, _, _ := pCreateSolidBrush.Call(hexColor(c))
	return d.del(h)
}

func (d *dc) pen(c uint32, w int32) uintptr {
	h, _, _ := pCreatePen.Call(0 /*PS_SOLID*/, uintptr(w), hexColor(c))
	return d.del(h)
}

func (d *dc) fill(r rect, c uint32) {
	b := d.brush(c)
	pFillRect.Call(d.h, uintptr(unsafe.Pointer(&r)), b)
}

// roundRect paints a filled (and optionally outlined) rounded rectangle.
func (d *dc) roundRect(r rect, radius int32, fillCol uint32, borderCol uint32, hasBorder bool) {
	var pen uintptr
	if hasBorder {
		pen = d.pen(borderCol, 1)
	} else {
		pen, _, _ = pGetStockObject.Call(nullPen)
	}
	old1, _, _ := pSelectObject.Call(d.h, pen)
	old2, _, _ := pSelectObject.Call(d.h, d.brush(fillCol))
	pRoundRect.Call(d.h, uintptr(r.Left), uintptr(r.Top), uintptr(r.Right), uintptr(r.Bottom),
		uintptr(radius*2), uintptr(radius*2))
	pSelectObject.Call(d.h, old1)
	pSelectObject.Call(d.h, old2)
}

func (d *dc) ellipse(r rect, fillCol uint32, borderCol uint32, hasBorder bool, penW int32) {
	var pen uintptr
	if hasBorder {
		pen = d.pen(borderCol, penW)
	} else {
		pen, _, _ = pGetStockObject.Call(nullPen)
	}
	old1, _, _ := pSelectObject.Call(d.h, pen)
	old2, _, _ := pSelectObject.Call(d.h, d.brush(fillCol))
	pEllipse.Call(d.h, uintptr(r.Left), uintptr(r.Top), uintptr(r.Right), uintptr(r.Bottom))
	pSelectObject.Call(d.h, old1)
	pSelectObject.Call(d.h, old2)
}

func (d *dc) line(x1, y1, x2, y2 int32, c uint32, w int32) {
	old, _, _ := pSelectObject.Call(d.h, d.pen(c, w))
	pMoveToEx.Call(d.h, uintptr(x1), uintptr(y1), 0)
	pLineTo.Call(d.h, uintptr(x2), uintptr(y2))
	pSelectObject.Call(d.h, old)
}

// text draws a string inside r with the given font/colour/flags and returns the
// height it used (useful for stacking wrapped paragraphs).
func (d *dc) text(s string, r rect, font uintptr, col uint32, flags uint32) int32 {
	if s == "" {
		return 0
	}
	old, _, _ := pSelectObject.Call(d.h, font)
	pSetBkMode.Call(d.h, transparent)
	pSetTextColor.Call(d.h, hexColor(col))
	w := u16(s)
	h, _, _ := pDrawText.Call(d.h, uintptr(unsafe.Pointer(w)), ^uintptr(0),
		uintptr(unsafe.Pointer(&r)), uintptr(flags))
	pSelectObject.Call(d.h, old)
	return int32(h)
}

// measure returns the height s needs when wrapped into width w.
func (d *dc) measure(s string, font uintptr, w int32, flags uint32) int32 {
	if s == "" {
		return 0
	}
	r := rect{0, 0, w, 0}
	old, _, _ := pSelectObject.Call(d.h, font)
	txt := u16(s)
	pDrawText.Call(d.h, uintptr(unsafe.Pointer(txt)), ^uintptr(0),
		uintptr(unsafe.Pointer(&r)), uintptr(flags|dtCalcRect))
	pSelectObject.Call(d.h, old)
	return r.Bottom - r.Top
}

// fonts bundles the type ramp at one DPI.
type fonts struct {
	title  uintptr // 15 bold
	body   uintptr // 11.5
	bodyB  uintptr // 11.5 bold
	small  uintptr // 10
	mono   uintptr // 10 (url)
	button uintptr // 12 semibold
}

func makeFont(pt float64, weight int32, dpi int32, face string) uintptr {
	lf := logFont{
		Height:         -int32(pt*float64(dpi)/72.0 + 0.5),
		Weight:         weight,
		CharSet:        1, // DEFAULT_CHARSET
		Quality:        5, // CLEARTYPE_QUALITY
		PitchAndFamily: 0,
	}
	copyU16(lf.FaceName[:], face)
	h, _, _ := pCreateFontIndirect.Call(uintptr(unsafe.Pointer(&lf)))
	return h
}

func newFonts(dpi int32) *fonts {
	const ui = "Segoe UI"
	return &fonts{
		title:  makeFont(14.5, 700, dpi, ui),
		body:   makeFont(11.0, 400, dpi, ui),
		bodyB:  makeFont(11.0, 700, dpi, ui),
		small:  makeFont(9.5, 400, dpi, ui),
		mono:   makeFont(9.0, 400, dpi, "Consolas"),
		button: makeFont(11.5, 600, dpi, ui),
	}
}

func (f *fonts) free() {
	for _, h := range []uintptr{f.title, f.body, f.bodyB, f.small, f.mono, f.button} {
		if h != 0 {
			pDeleteObject.Call(h)
		}
	}
}
