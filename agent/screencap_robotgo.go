//go:build robotgo

package main

import (
	"bytes"
	"image"
	"image/jpeg"
	"log"

	"github.com/go-vgo/robotgo"
	"github.com/vcaesar/screenshot"
	xdraw "golang.org/x/image/draw"
)

// Real screen-capture backend for the live-view feature. Built with -tags
// robotgo (same tag as real input) — it needs a display, which robotgo already
// requires, so view and control light up together. The stub build reports
// unavailable and the phone hides the Screen tab.
//
// Capture goes through vcaesar/screenshot's CaptureDisplay, NOT
// robotgo.CaptureImg: robotgo's X11 path hands the rect raw to XGetImage on
// the root window, but GetDisplayBounds coords are relative to the PRIMARY
// display's origin — on layouts where the primary isn't at the root origin
// (second monitor left of / above the primary) the rect lands outside the
// root and Xlib's default error handler kills the process with
// "BadMatch (X_GetImage)". CaptureDisplay uses the same coordinate system as
// its own bounds, intersects with the root and pads, so it can't BadMatch —
// and per-display capture also means display 0 is display 0, not the whole
// virtual desktop.
type rgScreen struct{}

func newScreener() Screener {
	log.Println("[screen] robotgo capture backend (live screen view available)")
	return rgScreen{}
}

func (rgScreen) Available() bool { return true }

// Displays enumerates monitors in GLOBAL desktop coordinates (what
// Injector.MoveAbs expects). Per-OS displayBounds() — on Linux raw Xinerama
// (see displays_robotgo_linux.go), elsewhere vcaesar/screenshot, which is
// already global there. Xinerama can be absent (headless Xvfb, odd servers)
// → fall back to a single display covering the primary screen so
// view/control keep working.
func (rgScreen) Displays() []DisplayInfo {
	if out := displayBounds(); len(out) > 0 {
		return out
	}
	w, h := robotgo.GetScreenSize()
	return []DisplayInfo{{X: 0, Y: 0, W: w, H: h}}
}

// Capture grabs one display (by index into Displays), downscales it to
// scalePct of its native size, and returns JPEG bytes at the given quality
// plus the encoded dims.
func (s rgScreen) Capture(display, scalePct, quality int) ([]byte, int, int, error) {
	var (
		src image.Image
		err error
	)
	if n := screenshot.NumActiveDisplays(); n > 0 {
		if display < 0 || display >= n {
			display = 0
		}
		src, err = screenshot.CaptureDisplay(display)
	} else {
		// No Xinerama (headless Xvfb): the historical whole-screen path.
		src, err = robotgo.CaptureImg()
	}
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
