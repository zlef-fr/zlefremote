package core

import (
	"bytes"
	"image"
	"image/draw"
	"image/jpeg"
	"sync"
	"time"
)

// Quality Presets. The agent clamps fps to 1..20, quality to 20..90 and scale
// to 20..100 (agent/session.go), so these stay inside its range. Scale is a
// percentage of the host's native resolution: dropping it is by far the
// cheapest way to buy frame rate on a slow link.
type Preset struct {
	Name    string
	FPS     int
	Quality int
	Scale   int
}

var Presets = []Preset{
	{Name: "fast", FPS: 20, Quality: 45, Scale: 50},
	{Name: "balanced", FPS: 14, Quality: 60, Scale: 75},
	{Name: "sharp", FPS: 10, Quality: 82, Scale: 100},
}

func PresetByName(n string) Preset {
	for _, p := range Presets {
		if p.Name == n {
			return p
		}
	}
	return Presets[1]
}

func NextPreset(n string) Preset {
	for i, p := range Presets {
		if p.Name == n {
			return Presets[(i+1)%len(Presets)]
		}
	}
	return Presets[1]
}

// Stream reassembles the chunked JPEG frames the agent pushes and decodes them
// off the UI goroutine.
//
// The agent splits every JPEG into ≤32 KB chunks so the sealed frame stays
// under the relay's 64 KB ceiling; chunks of one frame share an id `i` and
// carry their index `s` of `n`. A frame whose chunks are still incomplete when
// the next id arrives is dropped — on a lossy link a fresher frame is always
// worth more than a repaired stale one.
type Stream struct {
	mu    sync.Mutex
	id    float64
	parts [][]byte
	got   int
	w, h  int

	decodeCh chan []byte
	once     sync.Once

	frameMu sync.Mutex
	frame   *image.RGBA
	seq     uint64

	statMu     sync.Mutex
	frameTimes []time.Time // decode timestamps for the fps readout
	bytesIn    int64
	bytesAt    time.Time
	kbps       float64
	lastFrame  time.Time
}

func NewStream() *Stream {
	s := &Stream{decodeCh: make(chan []byte, 1), bytesAt: time.Now()}
	return s
}

func (s *Stream) start() {
	s.once.Do(func() {
		go func() {
			for jb := range s.decodeCh {
				img, err := jpeg.Decode(bytes.NewReader(jb))
				if err != nil {
					continue
				}
				b := img.Bounds()
				rgba, ok := img.(*image.RGBA)
				if !ok {
					// JPEG decodes to YCbCr; convert once into the layout
					// ebiten's WritePixels wants.
					rgba = image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
					draw.Draw(rgba, rgba.Bounds(), img, b.Min, draw.Src)
				}
				s.frameMu.Lock()
				s.frame = rgba
				s.seq++
				s.frameMu.Unlock()
				s.noteFrame()
			}
		}()
	})
}

// Feed takes one decrypted {"t":"f",…} frame from the host.
func (s *Stream) Feed(m map[string]any) {
	s.start()
	id, _ := m["i"].(float64)
	idx := int(Num(m["s"]))
	n := int(Num(m["n"]))
	w, h := int(Num(m["w"])), int(Num(m["h"]))
	data, _ := m["d"].(string)
	if n <= 0 || idx < 0 || idx >= n || data == "" {
		return
	}
	chunk, err := B64.DecodeString(data)
	if err != nil {
		return
	}

	s.mu.Lock()
	if id != s.id || len(s.parts) != n {
		s.id, s.parts, s.got = id, make([][]byte, n), 0
	}
	if s.parts[idx] == nil {
		s.parts[idx] = chunk
		s.got++
	}
	s.w, s.h = w, h
	complete := s.got == n
	var full []byte
	if complete {
		total := 0
		for _, p := range s.parts {
			total += len(p)
		}
		full = make([]byte, 0, total)
		for _, p := range s.parts {
			full = append(full, p...)
		}
		s.parts, s.got = nil, 0
	}
	s.mu.Unlock()

	s.noteBytes(int64(len(chunk)))
	if !complete {
		return
	}
	select {
	case s.decodeCh <- full:
	default:
		// the decoder is still busy: skip this frame rather than queue latency
	}
}

// Latest returns the newest decoded frame and a sequence number so the UI can
// tell whether it needs to re-upload the texture.
func (s *Stream) Latest() (*image.RGBA, uint64) {
	s.frameMu.Lock()
	defer s.frameMu.Unlock()
	return s.frame, s.seq
}

// Reset drops any half-assembled and decoded frame (used when the stream stops
// or the viewed monitor changes).
func (s *Stream) Reset() {
	s.mu.Lock()
	s.parts, s.got, s.id = nil, 0, 0
	s.mu.Unlock()
	s.frameMu.Lock()
	s.frame = nil
	s.frameMu.Unlock()
}

func (s *Stream) noteFrame() {
	now := time.Now()
	s.statMu.Lock()
	s.lastFrame = now
	s.frameTimes = append(s.frameTimes, now)
	cut := now.Add(-2 * time.Second)
	for len(s.frameTimes) > 0 && s.frameTimes[0].Before(cut) {
		s.frameTimes = s.frameTimes[1:]
	}
	s.statMu.Unlock()
}

func (s *Stream) noteBytes(n int64) {
	s.statMu.Lock()
	s.bytesIn += n
	if d := time.Since(s.bytesAt); d > time.Second {
		s.kbps = float64(s.bytesIn) / d.Seconds() / 1024
		s.bytesIn, s.bytesAt = 0, time.Now()
	}
	s.statMu.Unlock()
}

// Stats is what the HUD shows.
func (s *Stream) Stats() (fps float64, kbps float64, stale bool) {
	s.statMu.Lock()
	defer s.statMu.Unlock()
	if len(s.frameTimes) >= 2 {
		span := s.frameTimes[len(s.frameTimes)-1].Sub(s.frameTimes[0]).Seconds()
		if span > 0 {
			fps = float64(len(s.frameTimes)-1) / span
		}
	}
	return fps, s.kbps, !s.lastFrame.IsZero() && time.Since(s.lastFrame) > 2*time.Second
}

// Num reads a JSON number that landed in an any (always float64) without
// panicking on a missing/odd field.
func Num(v any) float64 {
	f, _ := v.(float64)
	return f
}
