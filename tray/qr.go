//go:build windows

// qr.go — the pairing QR, rendered straight into a GDI bitmap.
//
// The agent also drops a PNG in %TEMP% and advertises it over `@zr qr=`, but we
// re-render from the URL instead: a QR scaled by GDI loses module alignment and
// gets harder to scan, whereas rasterising the module matrix at an integer
// module size is crisp at any DPI. The PNG stays the fallback.
package main

import (
	"image"
	"image/png"
	"os"
	"unsafe"

	qrcode "github.com/skip2/go-qrcode"
)

// qrBitmap is a ready-to-blit 32-bit top-down DIB.
type qrBitmap struct {
	hbm  uintptr
	size int32
}

func (q *qrBitmap) free() {
	if q != nil && q.hbm != 0 {
		pDeleteObject.Call(q.hbm)
		q.hbm = 0
	}
}

// makeQR renders url as a QR no larger than maxPx (and as close to it as an
// integer module size allows). Quiet zone included by go-qrcode's Bitmap().
func makeQR(url string, maxPx int32) *qrBitmap {
	q, err := qrcode.New(url, qrcode.Medium)
	if err != nil {
		return nil
	}
	bm := q.Bitmap() // [][]bool, true = dark, already padded with the quiet zone
	n := int32(len(bm))
	if n == 0 {
		return nil
	}
	scale := maxPx / n
	if scale < 1 {
		scale = 1
	}
	side := n * scale

	pix := make([]byte, side*side*4)
	for y := int32(0); y < side; y++ {
		row := bm[y/scale]
		for x := int32(0); x < side; x++ {
			o := (y*side + x) * 4
			v := byte(0xff)
			if row[x/scale] {
				v = 0x00
			}
			pix[o+0], pix[o+1], pix[o+2], pix[o+3] = v, v, v, 0xff
		}
	}
	hbm := dibFromBGRA(pix, side, side)
	if hbm == 0 {
		return nil
	}
	return &qrBitmap{hbm: hbm, size: side}
}

// makeQRFromPNG is the fallback path: decode the PNG the agent rendered.
func makeQRFromPNG(path string, maxPx int32) *qrBitmap {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return nil
	}
	b := img.Bounds()
	side := int32(b.Dx())
	if side <= 0 {
		return nil
	}
	// integer downscale so modules stay square
	step := int32(1)
	for side/(step+1) >= maxPx {
		step++
	}
	out := side / step
	pix := make([]byte, out*out*4)
	for y := int32(0); y < out; y++ {
		for x := int32(0); x < out; x++ {
			r, g, bl, _ := img.At(b.Min.X+int(x*step), b.Min.Y+int(y*step)).RGBA()
			o := (y*out + x) * 4
			pix[o+0], pix[o+1], pix[o+2], pix[o+3] = byte(bl>>8), byte(g>>8), byte(r>>8), 0xff
		}
	}
	hbm := dibFromBGRA(pix, out, out)
	if hbm == 0 {
		return nil
	}
	return &qrBitmap{hbm: hbm, size: out}
}

// dibFromBGRA uploads a top-down BGRA buffer into a GDI DIB section.
func dibFromBGRA(pix []byte, w, h int32) uintptr {
	bi := bitmapInfo{Header: bitmapInfoHeader{
		Size:        uint32(unsafe.Sizeof(bitmapInfoHeader{})),
		Width:       w,
		Height:      -h, // negative = top-down
		Planes:      1,
		BitCount:    32,
		Compression: biRGB,
	}}
	var bits unsafe.Pointer
	hbm, _, _ := pCreateDIBSection.Call(0, uintptr(unsafe.Pointer(&bi)), dibRGBCol,
		uintptr(unsafe.Pointer(&bits)), 0, 0)
	if hbm == 0 || bits == nil {
		return 0
	}
	dst := unsafe.Slice((*byte)(bits), len(pix))
	copy(dst, pix)
	return hbm
}

// blit draws a bitmap at (x,y) through a memory DC.
func blit(d *dc, hbm uintptr, x, y, w, h int32) {
	mem, _, _ := pCreateCompatibleDC.Call(d.h)
	if mem == 0 {
		return
	}
	old, _, _ := pSelectObject.Call(mem, hbm)
	pBitBlt.Call(d.h, uintptr(x), uintptr(y), uintptr(w), uintptr(h), mem, 0, 0, srcCopy)
	pSelectObject.Call(mem, old)
	pDeleteDC.Call(mem)
}

// rgbaToBGRA flips an image.RGBA into the byte order GDI DIBs want.
func rgbaToBGRA(img *image.RGBA) []byte {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	out := make([]byte, w*h*4)
	for y := 0; y < h; y++ {
		src := img.Pix[y*img.Stride : y*img.Stride+w*4]
		dst := out[y*w*4 : (y+1)*w*4]
		for x := 0; x < w; x++ {
			dst[x*4+0] = src[x*4+2] // B
			dst[x*4+1] = src[x*4+1] // G
			dst[x*4+2] = src[x*4+0] // R
			dst[x*4+3] = src[x*4+3] // A
		}
	}
	return out
}
