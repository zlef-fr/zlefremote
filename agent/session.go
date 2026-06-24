package main

import (
	"encoding/json"
	"log"
)

// Injector is the OS input backend. The real implementation (inject_robotgo.go,
// built with -tags robotgo) drives the mouse/keyboard; the default stub
// (inject_stub.go) logs actions so the agent compiles and the relay/handshake
// can be exercised anywhere.
type Injector interface {
	MoveRel(dx, dy int)
	Click(button string, double bool)
	Toggle(button string, down bool)
	Scroll(dx, dy int)
	KeyTap(key string, mods []string)
	TypeStr(s string)
	Media(k string)
	ScreenSize() (int, int)
	HostInfo() (name, os string)
}

type cmd struct {
	T      string   `json:"t"`
	DX     int      `json:"dx"`
	DY     int      `json:"dy"`
	X      int      `json:"x"`
	Y      int      `json:"y"`
	B      string   `json:"b"`
	Double bool     `json:"double"`
	K      string   `json:"k"`
	Mods   []string `json:"mods"`
	S      string   `json:"s"`
}

// Session decrypts client frames and dispatches them to the injector. One
// Session per connected phone.
type Session struct {
	sealer *Sealer
	inj    Injector
	paired bool
}

func NewSession(s *Sealer, inj Injector) *Session { return &Session{sealer: s, inj: inj} }

// Handle decrypts an incoming sealed frame, applies it, and returns a sealed
// reply frame to send back (empty string if there is nothing to reply).
func (se *Session) Handle(frame string) (reply string) {
	pt, err := se.sealer.Open(frame)
	if err != nil {
		// wrong key / tampered frame → ignore (this is also the auth gate)
		return ""
	}
	var c cmd
	if err := json.Unmarshal(pt, &c); err != nil {
		return ""
	}
	switch c.T {
	case "hello":
		se.paired = true
		w, h := se.inj.ScreenSize()
		name, os := se.inj.HostInfo()
		b, _ := json.Marshal(map[string]any{
			"t": "welcome", "name": name, "os": os,
			"screen": map[string]int{"w": w, "h": h},
		})
		out, _ := se.sealer.Seal(b)
		log.Printf("paired with a phone")
		return out
	case "mv":
		se.inj.MoveRel(c.DX, c.DY)
	case "click":
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
	}
	return ""
}

func norm(b string) string {
	if b == "" {
		return "left"
	}
	return b
}
