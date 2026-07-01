//go:build robotgo

package main

import (
	"bytes"
	"image"
	"image/jpeg"
	"log"

	"github.com/go-vgo/robotgo"
	xdraw "golang.org/x/image/draw"
)

// Real screen-capture backend for the live-view feature. Built with -tags
// robotgo (same tag as real input) — it needs a display, which robotgo already
// requires, so view and control light up together. The stub build reports
// unavailable and the phone hides the Screen tab.
type rgScreen struct{}

func newScreener() Screener {
	log.Println("[screen] robotgo capture backend (live screen view available)")
	return rgScreen{}
}

func (rgScreen) Available() bool { return true }

// Capture grabs the primary display, downscales it to scalePct of its native
// size, and returns JPEG bytes at the given quality plus the encoded dims.
func (rgScreen) Capture(scalePct, quality int) ([]byte, int, int, error) {
	src, err := robotgo.CaptureImg()
	if err != nil {
		return nil, 0, 0, err
	}
	b := src.Bounds()
	sw, sh := b.Dx(), b.Dy()

	dw, dh := sw*scalePct/100, sh*scalePct/100
	var out image.Image = src
	if scalePct < 100 && dw >= 1 && dh >= 1 {
		dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
		// ApproxBiLinear: good enough for a remote view, far cheaper than
		// CatmullRom at video-ish frame rates.
		xdraw.ApproxBiLinear.Scale(dst, dst.Bounds(), src, b, xdraw.Src, nil)
		out = dst
	} else {
		dw, dh = sw, sh
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, out, &jpeg.Options{Quality: quality}); err != nil {
		return nil, 0, 0, err
	}
	return buf.Bytes(), dw, dh, nil
}
