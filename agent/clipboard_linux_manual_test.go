//go:build linux

package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"testing"
	"time"
)

// Manual check of the real Linux image writer: needs a live display (DISPLAY,
// e.g. Xvfb, or WAYLAND_DISPLAY) and xclip or wl-clipboard. Skipped unless
// ZR_CLIP_IMAGE is set:
//
//	ZR_CLIP_IMAGE=1 xvfb-run -a go test -run TestClipboardImageManual .
//
// It reads the picture back as image/png, the way a paste target would.
func TestClipboardImageManual(t *testing.T) {
	if os.Getenv("ZR_CLIP_IMAGE") == "" {
		t.Skip("set ZR_CLIP_IMAGE (and DISPLAY) to run")
	}
	c := newClipper()
	if !canWriteImage(c) {
		t.Fatal("no image-capable clipboard backend")
	}
	png := testPNG(300 * 1024)
	if err := c.(ImageClipper).WriteImage(png); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	argv := []string{"xclip", "-selection", "clipboard", "-t", "image/png", "-o"}
	if _, ok := waylandSocket(); ok {
		argv = []string{"wl-paste", "--type", "image/png"}
	}
	out, err := exec.CommandContext(ctx, argv[0], argv[1:]...).Output()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, png) {
		t.Fatalf("read back %d bytes, want the %d-byte original", len(out), len(png))
	}
	// and the text reader must not hand those bytes over as "text"
	if s, err := c.Read(); err == nil && s != "" {
		t.Fatalf("text read of an image clipboard returned %d bytes of text", len(s))
	}
}
