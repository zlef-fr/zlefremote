package main

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"sync"
	"time"
)

// Injector is the OS input backend. The real implementation (inject_robotgo.go,
// built with -tags robotgo) drives the mouse/keyboard; the default stub
// (inject_stub.go) logs actions so the agent compiles and the relay/handshake
// can be exercised anywhere.
type Injector interface {
	MoveRel(dx, dy int)
	MoveAbs(x, y int)
	Click(button string, double bool)
	Toggle(button string, down bool)
	Scroll(dx, dy int)
	KeyTap(key string, mods []string)
	TypeStr(s string)
	Media(k string)
	ScreenSize() (int, int)
	HostInfo() (name, os string)
}

// Screener captures the desktop as JPEG frames for the live-view feature. The
// real implementation (screencap_robotgo.go, -tags robotgo) grabs the screen;
// the stub (screencap_stub.go) reports unavailable so the phone hides the tab.
type Screener interface {
	Available() bool
	// Displays enumerates monitors in global desktop coordinates. At least one
	// entry when Available; nil otherwise.
	Displays() []DisplayInfo
	// Capture grabs one display (index into Displays) scaled to scalePct (of
	// native size) and returns JPEG bytes at the given quality plus the
	// encoded width/height.
	Capture(display, scalePct, quality int) (jpeg []byte, w, h int, err error)
}

// DisplayInfo is one monitor's rectangle in the global desktop space.
type DisplayInfo struct {
	X, Y, W, H int
}

type cmd struct {
	T      string   `json:"t"`
	DX     int      `json:"dx"`
	DY     int      `json:"dy"`
	X      int      `json:"x"`
	Y      int      `json:"y"`
	NX     float64  `json:"nx"` // absolute pointer, normalized 0..1 of screen
	NY     float64  `json:"ny"`
	B      string   `json:"b"`
	Double bool     `json:"double"`
	K      string   `json:"k"`
	Mods   []string `json:"mods"`
	S      string   `json:"s"`
	On     bool     `json:"on"`    // view: start/stop live screen stream
	FPS    int      `json:"fps"`   // view: target frames per second
	Q      int      `json:"q"`     // view: JPEG quality
	Scale  int      `json:"scale"` // view: capture scale (percent of native)
	D      int      `json:"d"`     // view: display index (multi-monitor)
}

// A relayed frame must stay under the relay's 64 KB payload ceiling once the
// JPEG is base64'd, wrapped in JSON, sealed and base64'd again (~1.8× inflation).
// 32 KB of raw JPEG per chunk lands the final sealed frame around 58 KB.
const frameChunkMax = 32 * 1024

// Session decrypts client frames and dispatches them to the injector, and (for
// the live-view feature) pushes sealed screen frames back to one phone. One
// Session per connected phone; send() delivers a sealed frame to that phone.
type Session struct {
	sealer *Sealer
	inj    Injector
	scr    Screener
	send   func(sealed string) // push a sealed frame to this one client
	paired bool

	mu                  sync.Mutex
	streaming           bool
	stopCh              chan struct{}
	fps, quality, scale int
	display             int // which monitor is streamed / abs-pointer target
	frameID             uint32
}

func NewSession(s *Sealer, inj Injector, scr Screener, send func(string)) *Session {
	return &Session{sealer: s, inj: inj, scr: scr, send: send}
}

// sealSend marshals v to JSON, seals it, and pushes it to this phone.
func (se *Session) sealSend(v any) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	out, err := se.sealer.Seal(b)
	if err != nil {
		return
	}
	se.send(out)
}

// Handle decrypts an incoming sealed frame and applies it. Wrong key / tampered
// frames are silently ignored (this is also the auth gate).
func (se *Session) Handle(frame string) {
	pt, err := se.sealer.Open(frame)
	if err != nil {
		return
	}
	var c cmd
	if json.Unmarshal(pt, &c) != nil {
		return
	}
	switch c.T {
	case "hello":
		se.paired = true
		w, h := se.inj.ScreenSize()
		name, os := se.inj.HostInfo()
		screens := []map[string]int{}
		for _, d := range se.scr.Displays() {
			screens = append(screens, map[string]int{"w": d.W, "h": d.H})
		}
		se.sealSend(map[string]any{
			"t": "welcome", "name": name, "os": os,
			"screen":  map[string]int{"w": w, "h": h},
			"screens": screens,
			"cap":     map[string]bool{"screen": se.scr.Available()},
		})
		log.Printf("paired with a phone")
		emit("event", "paired")
	case "mv":
		se.inj.MoveRel(c.DX, c.DY)
	case "mvabs":
		se.moveAbs(c.NX, c.NY)
	case "click":
		se.inj.Click(norm(c.B), c.Double)
	case "clickabs":
		se.moveAbs(c.NX, c.NY)
		se.inj.Click(norm(c.B), c.Double)
	case "down":
		se.inj.Toggle(norm(c.B), true)
	case "up":
		se.inj.Toggle(norm(c.B), false)
	case "scroll":
		se.inj.Scroll(c.DX, c.DY)
	case "key":
		se.inj.KeyTap(c.K, c.Mods)
	case "text":
		se.inj.TypeStr(c.S)
	case "media":
		se.inj.Media(c.K)
	case "view":
		if c.On {
			se.startStream(c.FPS, c.Q, c.Scale, c.D)
		} else {
			se.stopStream()
		}
	}
}

// moveAbs maps a normalized 0..1 point onto the display currently being
// viewed (multi-monitor: each display has its own rect in the global desktop
// space). Falls back to the primary screen when no display info is available.
func (se *Session) moveAbs(nx, ny float64) {
	nx = clampF(nx)
	ny = clampF(ny)
	se.mu.Lock()
	di := se.display
	se.mu.Unlock()
	ox, oy, w, h := 0, 0, 0, 0
	if ds := se.scr.Displays(); len(ds) > 0 {
		if di < 0 || di >= len(ds) {
			di = 0
		}
		d := ds[di]
		ox, oy, w, h = d.X, d.Y, d.W, d.H
	} else {
		w, h = se.inj.ScreenSize()
	}
	x, y := int(nx*float64(w)), int(ny*float64(h))
	if x >= w {
		x = w - 1
	}
	if y >= h {
		y = h - 1
	}
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	se.inj.MoveAbs(ox+x, oy+y)
}

// startStream begins (or, if already running, retunes) the live screen
// stream. display switches monitors live — the loop re-reads it every frame.
func (se *Session) startStream(fps, q, scale, display int) {
	if !se.scr.Available() {
		se.sealSend(map[string]any{"t": "viewerr", "reason": "unsupported"})
		return
	}
	if n := len(se.scr.Displays()); display < 0 || display >= n {
		display = 0
	}
	se.mu.Lock()
	se.fps = clampInt(fps, 1, 20, 8)
	se.quality = clampInt(q, 20, 90, 55)
	se.scale = clampInt(scale, 20, 100, 75)
	se.display = display
	if !se.streaming {
		se.streaming = true
		se.stopCh = make(chan struct{})
		stop := se.stopCh
		se.mu.Unlock()
		go se.streamLoop(stop)
		return
	}
	se.mu.Unlock()
}

func (se *Session) stopStream() {
	se.mu.Lock()
	if se.streaming {
		se.streaming = false
		close(se.stopCh)
	}
	se.mu.Unlock()
}

// Close stops any live stream when the phone disconnects.
func (se *Session) Close() { se.stopStream() }

func (se *Session) streamLoop(stop chan struct{}) {
	emit("event", "view-start")
	if !machineMode {
		log.Printf("screen view: streaming to a phone")
	}
	defer func() {
		emit("event", "view-stop")
		if !machineMode {
			log.Printf("screen view: stopped")
		}
	}()
	for {
		if stopped(stop) {
			return
		}
		se.mu.Lock()
		fps, q, scale, display := se.fps, se.quality, se.scale, se.display
		se.mu.Unlock()

		start := time.Now()
		jb, w, h, err := se.scr.Capture(display, scale, q)
		if err != nil {
			se.sealSend(map[string]any{"t": "viewerr", "reason": "capture"})
			if sleepStop(stop, time.Second) {
				return
			}
			continue
		}
		if len(jb) > 0 {
			se.frameID++
			id := se.frameID
			n := (len(jb) + frameChunkMax - 1) / frameChunkMax
			for s := 0; s < n; s++ {
				lo := s * frameChunkMax
				hi := lo + frameChunkMax
				if hi > len(jb) {
					hi = len(jb)
				}
				se.sealSend(map[string]any{
					"t": "f", "i": id, "s": s, "n": n, "w": w, "h": h,
					"d": base64.RawURLEncoding.EncodeToString(jb[lo:hi]),
				})
				if stopped(stop) {
					return
				}
			}
		}

		interval := time.Second / time.Duration(fps)
		if el := time.Since(start); el < interval {
			if sleepStop(stop, interval-el) {
				return
			}
		}
	}
}

func stopped(stop chan struct{}) bool {
	select {
	case <-stop:
		return true
	default:
		return false
	}
}

func sleepStop(stop chan struct{}, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-stop:
		return true
	case <-t.C:
		return false
	}
}

func clampInt(v, lo, hi, def int) int {
	if v == 0 {
		return def
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func clampF(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func norm(b string) string {
	if b == "" {
		return "left"
	}
	return b
}
