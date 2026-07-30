package core

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
	"time"
)

// jpegFrame builds a JPEG the way the agent would, and splits it the way the
// agent's chunker does.
func jpegFrame(t *testing.T, w, h int, c color.RGBA) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func chunks(b []byte, n int) [][]byte {
	size := (len(b) + n - 1) / n
	var out [][]byte
	for i := 0; i < len(b); i += size {
		hi := i + size
		if hi > len(b) {
			hi = len(b)
		}
		out = append(out, b[i:hi])
	}
	return out
}

func frameMsg(id float64, idx, n, w, h int, chunk []byte) map[string]any {
	return map[string]any{
		"t": "f", "i": id, "s": float64(idx), "n": float64(n),
		"w": float64(w), "h": float64(h), "d": B64.EncodeToString(chunk),
	}
}

func waitFrame(t *testing.T, s *Stream) *image.RGBA {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if img, _ := s.Latest(); img != nil {
			return img
		}
		time.Sleep(10 * time.Millisecond)
	}
	return nil
}

func TestStreamReassemblesOutOfOrderChunks(t *testing.T) {
	s := NewStream()
	jb := jpegFrame(t, 64, 32, color.RGBA{200, 40, 60, 255})
	cs := chunks(jb, 3)
	// deliver 2, 0, 1 — the relay preserves order today, but a reassembler that
	// assumes it would be a landmine
	for _, i := range []int{2, 0, 1} {
		s.Feed(frameMsg(1, i, len(cs), 64, 32, cs[i]))
	}
	img := waitFrame(t, s)
	if img == nil {
		t.Fatal("no frame decoded")
	}
	if img.Bounds().Dx() != 64 || img.Bounds().Dy() != 32 {
		t.Fatalf("size: %v", img.Bounds())
	}
	r, g, b, _ := img.At(10, 10).RGBA()
	if r>>8 < 180 || g>>8 > 90 || b>>8 > 110 {
		t.Fatalf("pixel: %d %d %d — decoded image doesn't look like the source", r>>8, g>>8, b>>8)
	}
}

func TestStreamDropsIncompleteFrameWhenNextArrives(t *testing.T) {
	s := NewStream()
	old := jpegFrame(t, 32, 32, color.RGBA{10, 220, 10, 255})
	oldCs := chunks(old, 3)
	s.Feed(frameMsg(1, 0, 3, 32, 32, oldCs[0])) // frame 1 never completes

	fresh := jpegFrame(t, 48, 24, color.RGBA{20, 30, 240, 255})
	freshCs := chunks(fresh, 2)
	s.Feed(frameMsg(2, 0, 2, 48, 24, freshCs[0]))
	s.Feed(frameMsg(2, 1, 2, 48, 24, freshCs[1]))

	img := waitFrame(t, s)
	if img == nil {
		t.Fatal("no frame decoded")
	}
	if img.Bounds().Dx() != 48 {
		t.Fatalf("the stale half-frame was decoded instead of the fresh one: %v", img.Bounds())
	}
}

func TestStreamIgnoresGarbage(t *testing.T) {
	s := NewStream()
	s.Feed(map[string]any{"t": "f"})                                          // no chunk at all
	s.Feed(frameMsg(1, 5, 2, 10, 10, []byte("x")))                            // index past n
	s.Feed(map[string]any{"t": "f", "i": 1.0, "s": 0.0, "n": 1.0, "d": "!!"}) // not base64
	if img, _ := s.Latest(); img != nil {
		t.Fatal("garbage produced a frame")
	}
}

func TestPresetCycleIsStable(t *testing.T) {
	seen := map[string]bool{}
	p := PresetByName("balanced")
	for i := 0; i < len(Presets); i++ {
		seen[p.Name] = true
		p = NextPreset(p.Name)
	}
	if len(seen) != len(Presets) {
		t.Fatalf("cycling visited %d of %d Presets", len(seen), len(Presets))
	}
	if p.Name != "balanced" {
		t.Fatalf("cycle did not return to the start: %q", p.Name)
	}
	// every Preset must stay inside the ranges the agent clamps to
	for _, q := range Presets {
		if q.FPS < 1 || q.FPS > 20 || q.Quality < 20 || q.Quality > 90 || q.Scale < 20 || q.Scale > 100 {
			t.Fatalf("Preset %q is outside the agent's accepted range: %+v", q.Name, q)
		}
	}
}
