// Package main — ZlefRemote tray app.
//
// protocol.go parses the agent's machine protocol. The agent, run with
// `-machine`, prints one `@zr key=value` line per event on stdout (see
// agent/machine.go). This file turns that stream into a Session state that the
// UI renders. It is deliberately OS-independent so it can be unit-tested on any
// platform (the rest of the app is Win32-only).
package main

import (
	"sort"
	"strconv"
	"strings"
)

// Status mirrors the xfce plugin's ZrStatus.
type Status int

const (
	StatusIdle Status = iota
	StatusStarting
	StatusWaiting
	StatusPaired
)

// Peer is one connected phone.
type Peer struct {
	ID int
	IP string
}

// Session holds everything the UI needs to draw. It is mutated only from the UI
// thread (lines are handed over one at a time), so it needs no locking.
type Session struct {
	Status     Status
	URL        string // pairing URL (contains the #k= secret)
	QRPath     string // PNG the agent rendered, used as a fallback source
	Persistent bool   // agent reported a remembered identity
	Peers      map[int]string

	// Changed is set by Handle when anything the UI shows moved.
	Changed bool
	// Events the UI may want to react to beyond a repaint.
	OnPaired     func()
	OnDisconnect func()
}

func NewSession() *Session {
	return &Session{Peers: map[int]string{}}
}

// Reset clears everything that only makes sense while the agent runs.
func (s *Session) Reset() {
	s.Status = StatusIdle
	s.URL = ""
	s.QRPath = ""
	s.Persistent = false
	s.Peers = map[int]string{}
	s.Changed = true
}

// PeerList returns the connected phones ordered by id, so the roster doesn't
// reshuffle on every repaint (Go map iteration is randomised).
func (s *Session) PeerList() []Peer {
	out := make([]Peer, 0, len(s.Peers))
	for id, ip := range s.Peers {
		out = append(out, Peer{ID: id, IP: ip})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Handle consumes one line of agent stdout. Non-protocol lines are ignored.
func (s *Session) Handle(line string) {
	line = strings.TrimRight(line, "\r\n")
	if !strings.HasPrefix(line, "@zr ") {
		return
	}
	kv := line[len("@zr "):]
	eq := strings.IndexByte(kv, '=')
	if eq < 0 {
		return
	}
	key, val := kv[:eq], kv[eq+1:]

	switch key {
	case "url":
		s.URL = val
		s.Changed = true
	case "qr":
		s.QRPath = val
		s.Changed = true
	case "status":
		if val == "waiting" {
			s.setStatus(StatusWaiting)
		}
	case "persistent":
		s.Persistent = val == "1"
		s.Changed = true
	case "peer":
		s.handlePeer(val)
	case "clients":
		// authoritative count; 0 means the roster was reset (relay drop)
		if n, err := strconv.Atoi(val); err == nil && n == 0 {
			s.Peers = map[int]string{}
			s.Changed = true
		}
	case "event":
		switch val {
		case "paired":
			s.setStatus(StatusPaired)
			if s.OnPaired != nil {
				s.OnPaired()
			}
		case "disconnect":
			s.Peers = map[int]string{}
			s.setStatus(StatusWaiting)
			s.Changed = true
			if s.OnDisconnect != nil {
				s.OnDisconnect()
			}
		}
	}
}

// handlePeer parses "join <id> [ip]" / "leave <id>".
func (s *Session) handlePeer(val string) {
	switch {
	case strings.HasPrefix(val, "join "):
		f := strings.Fields(val[len("join "):])
		if len(f) == 0 {
			return
		}
		id, err := strconv.Atoi(f[0])
		if err != nil {
			return
		}
		ip := ""
		if len(f) > 1 {
			ip = f[1]
		}
		s.Peers[id] = ip
		s.Changed = true
	case strings.HasPrefix(val, "leave "):
		if id, err := strconv.Atoi(strings.TrimSpace(val[len("leave "):])); err == nil {
			delete(s.Peers, id)
			s.Changed = true
		}
	}
}

func (s *Session) setStatus(st Status) {
	if s.Status != st {
		s.Status = st
		s.Changed = true
	}
}

// LineSplitter accumulates arbitrary stdout chunks and yields complete lines.
// (bufio.Scanner would do, but the reader lives in a goroutine that must never
// block the UI on a partial line, and this keeps the seam testable.)
type LineSplitter struct{ buf strings.Builder }

func (l *LineSplitter) Feed(chunk []byte, emit func(string)) {
	l.buf.Write(chunk)
	s := l.buf.String()
	for {
		i := strings.IndexByte(s, '\n')
		if i < 0 {
			break
		}
		emit(strings.TrimRight(s[:i], "\r"))
		s = s[i+1:]
	}
	l.buf.Reset()
	l.buf.WriteString(s)
}
