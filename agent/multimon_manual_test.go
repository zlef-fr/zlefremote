//go:build robotgo && linux

package main

// Manual multi-monitor harness (needs a live X server with 2+ Xinerama
// screens — see the 1.5.2 fix notes). Run:
//
//	Xvfb :98 -screen 0 800x600x24 -screen 1 640x480x24 +xinerama &
//	DISPLAY=:98 ZR_MM=1 go test -tags robotgo -run TestMultiMonManual -v
//
// Regression for the 1.5.1 multi-monitor bugs: display 0 captured the whole
// virtual desktop, display 1 crashed the agent with BadMatch (X_GetImage).

import (
	"bytes"
	"image"
	"image/jpeg"
	"os"
	"testing"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

// paintWindow maps an override-redirect solid-color window at the given root
// rect and returns a cleanup func.
func paintWindow(t *testing.T, x, y int16, w, h uint16, pixel uint32) func() {
	c, err := xgb.NewConn()
	if err != nil {
		t.Fatalf("xgb: %v", err)
	}
	setup := xproto.Setup(c)
	screen := setup.DefaultScreen(c)
	wid, _ := xproto.NewWindowId(c)
	xproto.CreateWindow(c, screen.RootDepth, wid, screen.Root,
		x, y, w, h, 0, xproto.WindowClassInputOutput, screen.RootVisual,
		xproto.CwBackPixel|xproto.CwOverrideRedirect,
		[]uint32{pixel, 1})
	xproto.MapWindow(c, wid)
	c.Sync()
	return func() { c.Close() }
}

func decode(t *testing.T, b []byte) image.Image {
	img, err := jpeg.Decode(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("jpeg: %v", err)
	}
	return img
}

// dominantRed reports whether the image center is red-ish (JPEG is lossy).
func centerIsRed(img image.Image) bool {
	b := img.Bounds()
	r, g, bl, _ := img.At(b.Min.X+b.Dx()/2, b.Min.Y+b.Dy()/2).RGBA()
	return r>>8 > 180 && g>>8 < 80 && bl>>8 < 80
}

func TestMultiMonManual(t *testing.T) {
	if os.Getenv("ZR_MM") == "" {
		t.Skip("set ZR_MM=1 with a 2-screen Xinerama DISPLAY to run")
	}
	s := rgScreen{}
	ds := s.Displays()
	t.Logf("displays: %+v", ds)
	if len(ds) < 2 {
		t.Fatalf("want >=2 displays, got %+v", ds)
	}

	// Red marker window covering display 1 (global coords from Displays).
	d1 := ds[1]
	cleanup := paintWindow(t, int16(d1.X), int16(d1.Y), uint16(d1.W), uint16(d1.H), 0xff0000)
	defer cleanup()

	// Display 1: must capture at its own native size, red center, no crash.
	b1, w1, h1, err := s.Capture(1, 100, 85)
	if err != nil {
		t.Fatalf("capture d1: %v", err)
	}
	if w1 != d1.W || h1 != d1.H {
		t.Fatalf("d1 dims: got %dx%d want %dx%d", w1, h1, d1.W, d1.H)
	}
	if !centerIsRed(decode(t, b1)) {
		t.Fatalf("d1 center not red — captured wrong region")
	}

	// Display 0: must be its own size only (not the merged desktop) and NOT red.
	d0 := ds[0]
	b0, w0, h0, err := s.Capture(0, 100, 85)
	if err != nil {
		t.Fatalf("capture d0: %v", err)
	}
	if w0 != d0.W || h0 != d0.H {
		t.Fatalf("d0 dims: got %dx%d want %dx%d (whole-desktop regression)", w0, h0, d0.W, d0.H)
	}
	if centerIsRed(decode(t, b0)) {
		t.Fatalf("d0 center red — captured display 1's region")
	}
}
