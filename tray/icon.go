//go:build windows

// icon.go — the notification-area icon.
//
// The artwork is the same phone-radiating-a-signal mark the Xfce plugin ships,
// embedded as PNGs and turned into an HICON at runtime. Building the icon in Go
// (rather than through a resource + the icon theme) means one code path for
// every DPI *and* lets us tint a small status dot onto it, so the tray itself
// tells you whether a phone is connected.
package main

import (
	"bytes"
	"embed"
	"image"
	"image/png"
	"unsafe"
)

//go:embed assets/icon-*.png
var iconFS embed.FS

var iconSizes = []int{16, 24, 32, 48, 64}

// dotColor picks the status dot: none when idle, olive while waiting, grape
// once a phone is on the other end.
func dotColor(st Status) (uint32, bool) {
	switch st {
	case StatusStarting:
		return colAccentSft, true
	case StatusWaiting:
		return colAccentBr, true
	case StatusPaired:
		return colGrapeSoft, true
	}
	return 0, false
}

// makeIcon renders the mark at px pixels with an optional status dot.
func makeIcon(px int, st Status) uintptr {
	img := loadIconImage(px)
	if img == nil {
		return 0
	}
	if c, ok := dotColor(st); ok {
		drawDot(img, c)
	}
	return iconFromRGBA(img)
}

// loadIconImage decodes the smallest embedded PNG that is >= px and box-filters
// it down to exactly px (nearest-neighbour would fringe the thin signal arcs).
func loadIconImage(px int) *image.RGBA {
	src := pickSource(px)
	if src == nil {
		return nil
	}
	if src.Bounds().Dx() == px {
		return toRGBA(src)
	}
	return resizeBox(toRGBA(src), px)
}

func pickSource(px int) image.Image {
	best := -1
	for _, s := range iconSizes {
		if s >= px {
			best = s
			break
		}
	}
	if best < 0 {
		best = iconSizes[len(iconSizes)-1]
	}
	data, err := iconFS.ReadFile("assets/icon-" + itoa(best) + ".png")
	if err != nil {
		return nil
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil
	}
	return img
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [8]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

func toRGBA(src image.Image) *image.RGBA {
	if r, ok := src.(*image.RGBA); ok {
		return r
	}
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			r, g, bl, a := src.At(b.Min.X+x, b.Min.Y+y).RGBA()
			o := dst.PixOffset(x, y)
			dst.Pix[o+0] = byte(r >> 8)
			dst.Pix[o+1] = byte(g >> 8)
			dst.Pix[o+2] = byte(bl >> 8)
			dst.Pix[o+3] = byte(a >> 8)
		}
	}
	return dst
}

// resizeBox is a plain box filter — enough for 64→16 icon work, no deps.
func resizeBox(src *image.RGBA, px int) *image.RGBA {
	sw := src.Bounds().Dx()
	dst := image.NewRGBA(image.Rect(0, 0, px, px))
	for y := 0; y < px; y++ {
		y0, y1 := y*sw/px, (y+1)*sw/px
		if y1 <= y0 {
			y1 = y0 + 1
		}
		for x := 0; x < px; x++ {
			x0, x1 := x*sw/px, (x+1)*sw/px
			if x1 <= x0 {
				x1 = x0 + 1
			}
			var r, g, b, a, n uint32
			for sy := y0; sy < y1; sy++ {
				for sx := x0; sx < x1; sx++ {
					o := src.PixOffset(sx, sy)
					r += uint32(src.Pix[o+0])
					g += uint32(src.Pix[o+1])
					b += uint32(src.Pix[o+2])
					a += uint32(src.Pix[o+3])
					n++
				}
			}
			o := dst.PixOffset(x, y)
			dst.Pix[o+0] = byte(r / n)
			dst.Pix[o+1] = byte(g / n)
			dst.Pix[o+2] = byte(b / n)
			dst.Pix[o+3] = byte(a / n)
		}
	}
	return dst
}

// drawDot stamps a filled status dot in the bottom-right corner, ringed with
// the page background so it stays legible on any taskbar colour.
func drawDot(img *image.RGBA, col uint32) {
	w := img.Bounds().Dx()
	// Small enough that the mark still reads at 16 px — the dot is a status
	// cue, not the icon.
	r := float64(w) * 0.22
	cx, cy := float64(w)-r-1, float64(w)-r-1
	ring := r + float64(w)*0.05
	cr, cg, cb := byte(col>>16), byte(col>>8), byte(col)
	for y := 0; y < w; y++ {
		for x := 0; x < w; x++ {
			dx, dy := float64(x)+0.5-cx, float64(y)+0.5-cy
			d2 := dx*dx + dy*dy
			o := img.PixOffset(x, y)
			switch {
			case d2 <= r*r:
				img.Pix[o+0], img.Pix[o+1], img.Pix[o+2], img.Pix[o+3] = cr, cg, cb, 0xff
			case d2 <= ring*ring:
				img.Pix[o+0], img.Pix[o+1], img.Pix[o+2], img.Pix[o+3] = 0x06, 0x06, 0x0a, 0xff
			}
		}
	}
}

// iconFromRGBA builds an HICON from straight-alpha RGBA. GDI draws 32-bpp icons
// through AlphaBlend, so the colour bitmap must carry PREMULTIPLIED alpha —
// skipping that shows up as bright halos around the antialiased arcs.
func iconFromRGBA(img *image.RGBA) uintptr {
	w := int32(img.Bounds().Dx())
	h := int32(img.Bounds().Dy())
	pix := make([]byte, w*h*4)
	for i := int32(0); i < w*h; i++ {
		r := uint32(img.Pix[i*4+0])
		g := uint32(img.Pix[i*4+1])
		b := uint32(img.Pix[i*4+2])
		a := uint32(img.Pix[i*4+3])
		pix[i*4+0] = byte(b * a / 255)
		pix[i*4+1] = byte(g * a / 255)
		pix[i*4+2] = byte(r * a / 255)
		pix[i*4+3] = byte(a)
	}
	color := dibFromBGRA(pix, w, h)
	if color == 0 {
		return 0
	}
	// 1-bpp AND mask, all zeros: with a 32-bpp colour bitmap the per-pixel
	// alpha decides visibility, the mask just has to exist.
	maskBits := make([]byte, ((w+31)/32)*4*h)
	mask, _, _ := pCreateBitmap.Call(uintptr(w), uintptr(h), 1, 1,
		uintptr(unsafe.Pointer(&maskBits[0])))
	ii := iconInfo{FIcon: 1, HbmMask: mask, HbmColor: color}
	hicon, _, _ := pCreateIconIndirect.Call(uintptr(unsafe.Pointer(&ii)))
	pDeleteObject.Call(color)
	pDeleteObject.Call(mask)
	return hicon
}
