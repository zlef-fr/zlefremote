package main

import (
	"encoding/json"
	"sync"
	"testing"
	"time"
)

// fakes ────────────────────────────────────────────────────────────────────────

type fakeInjector struct {
	mu    sync.Mutex
	moves [][2]int // MoveAbs calls
}

func (f *fakeInjector) MoveRel(dx, dy int) {}
func (f *fakeInjector) MoveAbs(x, y int) {
	f.mu.Lock()
	f.moves = append(f.moves, [2]int{x, y})
	f.mu.Unlock()
}
func (f *fakeInjector) Click(b string, d bool)      {}
func (f *fakeInjector) Toggle(b string, down bool)  {}
func (f *fakeInjector) Scroll(dx, dy int)           {}
func (f *fakeInjector) KeyTap(k string, m []string) {}
func (f *fakeInjector) TypeStr(s string)            {}
func (f *fakeInjector) Media(k string)              {}
func (f *fakeInjector) ScreenSize() (int, int)      { return 1920, 1080 }
func (f *fakeInjector) HostInfo() (string, string)  { return "test-host", "linux" }
func (f *fakeInjector) lastMove() ([2]int, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.moves) == 0 {
		return [2]int{}, false
	}
	return f.moves[len(f.moves)-1], true
}

// two side-by-side monitors: 1920x1080 primary + 1280x1024 to its right
type fakeScreen struct {
	mu       sync.Mutex
	captured []int // display indices passed to Capture
}

func (f *fakeScreen) Available() bool { return true }
func (f *fakeScreen) Displays() []DisplayInfo {
	return []DisplayInfo{
		{X: 0, Y: 0, W: 1920, H: 1080},
		{X: 1920, Y: 0, W: 1280, H: 1024},
	}
}
func (f *fakeScreen) Capture(display, scalePct, quality int) ([]byte, int, int, error) {
	f.mu.Lock()
	f.captured = append(f.captured, display)
	f.mu.Unlock()
	return []byte{0xff, 0xd8, 0xff}, 10, 10, nil
}
func (f *fakeScreen) capturedDisplays() []int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]int(nil), f.captured...)
}

// two brightness-adjustable screens, like a laptop panel + external monitor
type fakeBright struct {
	mu    sync.Mutex
	avail bool
	cur   [2]int
	sets  [][2]int // every Set call as (display, pct), in order
	slow  time.Duration
}

func (f *fakeBright) Available() bool { return f.avail }
func (f *fakeBright) Screens() []BrightScreen {
	if !f.avail {
		return nil
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return []BrightScreen{{Name: "eDP-1", Pct: f.cur[0]}, {Name: "HDMI-1", Pct: f.cur[1]}}
}
func (f *fakeBright) Set(display, pct int) {
	if f.slow > 0 {
		time.Sleep(f.slow)
	}
	f.mu.Lock()
	for i := range f.cur {
		if display < 0 || display == i {
			f.cur[i] = pct
		}
	}
	f.sets = append(f.sets, [2]int{display, pct})
	f.mu.Unlock()
}
func (f *fakeBright) levels() [2]int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.cur
}
func (f *fakeBright) setCalls() [][2]int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([][2]int(nil), f.sets...)
}

// harness ──────────────────────────────────────────────────────────────────────

type harness struct {
	se     *Session
	sealer *Sealer
	inj    *fakeInjector
	scr    *fakeScreen
	br     *fakeBright
	mu     sync.Mutex
	out    []map[string]any // decrypted frames the agent pushed to the phone
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	key, _ := NewKey()
	sealer, err := NewSealer(key)
	if err != nil {
		t.Fatal(err)
	}
	h := &harness{sealer: sealer, inj: &fakeInjector{}, scr: &fakeScreen{}, br: &fakeBright{avail: true, cur: [2]int{70, 40}}}
	h.se = NewSession(sealer, h.inj, h.scr, h.br, func(sealed string) {
		pt, err := sealer.Open(sealed)
		if err != nil {
			t.Errorf("agent pushed an unopenable frame: %v", err)
			return
		}
		var m map[string]any
		if json.Unmarshal(pt, &m) != nil {
			t.Errorf("agent pushed non-JSON: %s", pt)
			return
		}
		h.mu.Lock()
		h.out = append(h.out, m)
		h.mu.Unlock()
	})
	return h
}

func (h *harness) sendCmd(t *testing.T, v any) {
	t.Helper()
	b, _ := json.Marshal(v)
	f, err := h.sealer.Seal(b)
	if err != nil {
		t.Fatal(err)
	}
	h.se.Handle(f)
}

func (h *harness) frames(typ string) []map[string]any {
	h.mu.Lock()
	defer h.mu.Unlock()
	var out []map[string]any
	for _, m := range h.out {
		if m["t"] == typ {
			out = append(out, m)
		}
	}
	return out
}

// tests ────────────────────────────────────────────────────────────────────────

func TestWelcomeCarriesScreens(t *testing.T) {
	h := newHarness(t)
	h.sendCmd(t, map[string]any{"t": "hello"})
	ws := h.frames("welcome")
	if len(ws) != 1 {
		t.Fatalf("want 1 welcome, got %d", len(ws))
	}
	screens, ok := ws[0]["screens"].([]any)
	if !ok || len(screens) != 2 {
		t.Fatalf("welcome.screens: want 2 entries, got %#v", ws[0]["screens"])
	}
	s1 := screens[1].(map[string]any)
	if s1["w"].(float64) != 1280 || s1["h"].(float64) != 1024 {
		t.Fatalf("screens[1]: want 1280x1024, got %#v", s1)
	}
}

func TestViewCapturesSelectedDisplayAndSwitchesLive(t *testing.T) {
	h := newHarness(t)
	h.sendCmd(t, map[string]any{"t": "view", "on": true, "fps": 20, "q": 40, "scale": 40, "d": 1})
	waitFor(t, func() bool { return len(h.scr.capturedDisplays()) >= 2 })
	for _, d := range h.scr.capturedDisplays() {
		if d != 1 {
			t.Fatalf("captured display %d, want 1", d)
		}
	}
	// live switch back to display 0 (view on while already streaming = retune)
	h.sendCmd(t, map[string]any{"t": "view", "on": true, "fps": 20, "q": 40, "scale": 40, "d": 0})
	waitFor(t, func() bool {
		caps := h.scr.capturedDisplays()
		return len(caps) > 0 && caps[len(caps)-1] == 0
	})
	h.se.Close()
}

func TestViewOutOfRangeDisplayFallsBackToPrimary(t *testing.T) {
	h := newHarness(t)
	h.sendCmd(t, map[string]any{"t": "view", "on": true, "fps": 20, "d": 7})
	waitFor(t, func() bool { return len(h.scr.capturedDisplays()) >= 1 })
	if d := h.scr.capturedDisplays()[0]; d != 0 {
		t.Fatalf("out-of-range display: captured %d, want 0", d)
	}
	h.se.Close()
}

func TestMoveAbsMapsIntoViewedDisplay(t *testing.T) {
	h := newHarness(t)
	// not streaming: mvabs maps onto display 0
	h.sendCmd(t, map[string]any{"t": "mvabs", "nx": 0.5, "ny": 0.5})
	if m, ok := h.inj.lastMove(); !ok || m != [2]int{960, 540} {
		t.Fatalf("display-0 mvabs: got %v, want [960 540]", m)
	}
	// start viewing display 1 → same normalized point lands in its global rect
	h.sendCmd(t, map[string]any{"t": "view", "on": true, "fps": 20, "d": 1})
	h.sendCmd(t, map[string]any{"t": "mvabs", "nx": 0.5, "ny": 0.5})
	if m, ok := h.inj.lastMove(); !ok || m != [2]int{1920 + 640, 512} {
		t.Fatalf("display-1 mvabs: got %v, want [2560 512]", m)
	}
	// corner clamp stays inside display 1
	h.sendCmd(t, map[string]any{"t": "mvabs", "nx": 1.0, "ny": 1.0})
	if m, _ := h.inj.lastMove(); m != [2]int{1920 + 1279, 1023} {
		t.Fatalf("display-1 corner: got %v, want [3199 1023]", m)
	}
	h.se.Close()
}

func TestWelcomeCarriesBrightness(t *testing.T) {
	h := newHarness(t)
	h.sendCmd(t, map[string]any{"t": "hello"})
	w := h.frames("welcome")[0]
	cap := w["cap"].(map[string]any)
	if cap["bright"] != true {
		t.Fatalf("cap.bright: want true, got %#v", cap["bright"])
	}
	if v, _ := w["bright"].(float64); v != 70 {
		t.Fatalf("welcome.bright: want 70, got %#v", w["bright"])
	}
	// per-screen list: two screens with names and current levels
	bs, _ := w["brights"].([]any)
	if len(bs) != 2 {
		t.Fatalf("welcome.brights: want 2 entries, got %#v", w["brights"])
	}
	s0 := bs[0].(map[string]any)
	s1 := bs[1].(map[string]any)
	if s0["name"] != "eDP-1" || s0["v"].(float64) != 70 {
		t.Fatalf("brights[0]: got %#v", s0)
	}
	if s1["name"] != "HDMI-1" || s1["v"].(float64) != 40 {
		t.Fatalf("brights[1]: got %#v", s1)
	}
}

func TestWelcomeHidesBrightnessWhenUnavailable(t *testing.T) {
	h := newHarness(t)
	h.br.avail = false
	h.sendCmd(t, map[string]any{"t": "hello"})
	w := h.frames("welcome")[0]
	if cap := w["cap"].(map[string]any); cap["bright"] != false {
		t.Fatalf("cap.bright: want false, got %#v", cap["bright"])
	}
	if _, present := w["bright"]; present {
		t.Fatalf("welcome.bright should be absent when unsupported")
	}
	// a bright command against an unavailable backend is ignored
	h.sendCmd(t, map[string]any{"t": "bright", "v": 50})
	time.Sleep(50 * time.Millisecond)
	if calls := h.br.setCalls(); len(calls) != 0 {
		t.Fatalf("Set called despite unavailable backend: %v", calls)
	}
}

func TestBrightSetsAndClamps(t *testing.T) {
	h := newHarness(t)
	// no bd (old client) → every screen
	h.sendCmd(t, map[string]any{"t": "bright", "v": 60})
	waitFor(t, func() bool { return h.br.levels() == [2]int{60, 60} })
	// floor: never black the screen out
	h.sendCmd(t, map[string]any{"t": "bright", "v": 0})
	waitFor(t, func() bool { return h.br.levels() == [2]int{brightMin, brightMin} })
	// ceiling
	h.sendCmd(t, map[string]any{"t": "bright", "v": 250})
	waitFor(t, func() bool { return h.br.levels() == [2]int{100, 100} })
}

func TestBrightPerScreen(t *testing.T) {
	h := newHarness(t)
	// target only the second screen; the first keeps its level
	h.sendCmd(t, map[string]any{"t": "bright", "v": 25, "bd": 1})
	waitFor(t, func() bool { return h.br.levels() == [2]int{70, 25} })
	// explicit -1 = all screens
	h.sendCmd(t, map[string]any{"t": "bright", "v": 80, "bd": -1})
	waitFor(t, func() bool { return h.br.levels() == [2]int{80, 80} })
	// out-of-range index falls back to all (same spirit as the view fallback)
	h.sendCmd(t, map[string]any{"t": "bright", "v": 33, "bd": 7})
	waitFor(t, func() bool { return h.br.levels() == [2]int{33, 33} })
}

func TestBrightLatestWinsUnderBurst(t *testing.T) {
	h := newHarness(t)
	h.br.slow = 20 * time.Millisecond // emulate a slow OS tool
	for v := 10; v <= 90; v += 5 {
		h.sendCmd(t, map[string]any{"t": "bright", "v": v})
	}
	waitFor(t, func() bool { return h.br.levels() == [2]int{90, 90} })
	// the worker must have skipped stale intermediate values, not queued all 17
	if calls := h.br.setCalls(); len(calls) >= 17 {
		t.Fatalf("burst not coalesced: %d Set calls (%v)", len(calls), calls)
	}
}

func TestBrightPerScreenBurstKeepsBothTargets(t *testing.T) {
	h := newHarness(t)
	h.br.slow = 20 * time.Millisecond
	// a per-screen tweak right after an all-set must not be dropped by coalescing
	h.sendCmd(t, map[string]any{"t": "bright", "v": 50, "bd": -1})
	h.sendCmd(t, map[string]any{"t": "bright", "v": 20, "bd": 0})
	waitFor(t, func() bool { return h.br.levels() == [2]int{20, 50} })
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not met within 2s")
}
