package core

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// Conn is the client half of the ZlefRemote transport. Both modes speak the
// same {"t":"data","payload":<sealed>} envelope:
//
//	relay  — dial wss://<relay>/ws, {"t":"join","room":…}, wait for "joined"
//	LAN    — dial ws://<agent>/ws, the agent is already listening
//
// Everything above this layer only ever sees plaintext maps; sealing/opening
// happens here, so a wrong key simply yields no traffic (that is also the
// authentication gate — see agent/session.go).
type Conn struct {
	target Target
	sealer *Sealer

	events chan Event

	mu     sync.Mutex
	ws     *websocket.Conn
	closed bool
	cancel context.CancelFunc
}

// ClientVersion is stamped into the handshake's user agent; main sets it at
// startup so this package stays free of build metadata.
var ClientVersion = "dev"

// Event is what the UI loop consumes. Kind is one of:
// "state" (Text = connecting|linked|paired|reconnecting|closed), "cmd" (Cmd is
// a decrypted frame from the host) or "error".
type Event struct {
	Kind string
	Text string
	Cmd  map[string]any
}

const (
	StateConnecting   = "connecting"
	StateLinked       = "linked"
	StatePaired       = "paired"
	StateReconnecting = "reconnecting"
	StateClosed       = "closed"
)

func NewConn(t Target) (*Conn, error) {
	s, err := SealerFromKeyB64(t.KeyB64)
	if err != nil {
		return nil, err
	}
	return &Conn{target: t, sealer: s, events: make(chan Event, 256)}, nil
}

func (c *Conn) Events() <-chan Event { return c.events }

// Start dials and keeps the session alive: a dropped socket is retried with a
// short backoff (the relay also drops idle rooms, and laptops sleep), until
// Close is called.
func (c *Conn) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	c.mu.Lock()
	c.cancel = cancel
	c.mu.Unlock()
	go func() {
		defer close(c.events)
		attempt := 0
		for {
			if c.isClosed() {
				return
			}
			err := c.runOnce(ctx)
			if c.isClosed() {
				return
			}
			attempt++
			c.emit(Event{Kind: "state", Text: StateReconnecting})
			if err != nil {
				c.emit(Event{Kind: "error", Text: err.Error()})
			}
			// backoff 1s → 5s: fast enough for a Wi-Fi blip, gentle on the relay
			wait := time.Duration(min(attempt, 5)) * time.Second
			select {
			case <-ctx.Done():
				return
			case <-time.After(wait):
			}
		}
	}()
}

func (c *Conn) runOnce(ctx context.Context) error {
	c.emit(Event{Kind: "state", Text: StateConnecting})
	dialCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	ws, _, err := websocket.Dial(dialCtx, c.target.WSURL, nil)
	if err != nil {
		return err
	}
	// screen frames are chunked to ~58 KB by the agent; give the read side room
	ws.SetReadLimit(256 * 1024)
	defer ws.CloseNow()

	c.mu.Lock()
	c.ws = ws
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		c.ws = nil
		c.mu.Unlock()
	}()

	if c.target.Relay {
		if err := c.writeRaw(ctx, map[string]any{"t": "join", "room": c.target.Room}); err != nil {
			return err
		}
	} else {
		c.linked(ctx)
	}

	// transport keepalive (the relay answers with a pong; a LAN agent ignores it)
	stopPing := make(chan struct{})
	defer close(stopPing)
	go func() {
		t := time.NewTicker(25 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-stopPing:
				return
			case <-t.C:
				_ = c.writeRaw(ctx, map[string]any{"t": "ping"})
			}
		}
	}()

	for {
		_, data, err := ws.Read(ctx)
		if err != nil {
			return err
		}
		var msg struct {
			T       string `json:"t"`
			Payload string `json:"payload"`
			Reason  string `json:"reason"`
			Error   string `json:"error"`
		}
		if json.Unmarshal(data, &msg) != nil {
			continue
		}
		switch msg.T {
		case "joined":
			c.linked(ctx)
		case "data":
			pt, err := c.sealer.Open(msg.Payload)
			if err != nil {
				continue // not ours / tampered — ignore, exactly like the phone
			}
			var m map[string]any
			if json.Unmarshal(pt, &m) != nil {
				continue
			}
			c.emit(Event{Kind: "cmd", Cmd: m})
		case "closed":
			c.emit(Event{Kind: "state", Text: StateClosed})
			c.emit(Event{Kind: "error", Text: msg.Reason})
			return nil
		case "error":
			c.emit(Event{Kind: "error", Text: msg.Error})
			return nil
		}
	}
}

// linked announces the transport is up and performs the handshake; the agent
// answers with a welcome frame carrying its name, screens and capabilities.
func (c *Conn) linked(ctx context.Context) {
	c.emit(Event{Kind: "state", Text: StateLinked})
	_ = c.Send(map[string]any{"t": "hello", "v": 1, "ua": "zlefremote-desktop/" + ClientVersion})
}

// Send seals and writes one command to the host. Safe from any goroutine.
func (c *Conn) Send(v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	sealed, err := c.sealer.Seal(b)
	if err != nil {
		return err
	}
	return c.writeRaw(context.Background(), map[string]any{"t": "data", "payload": sealed})
}

func (c *Conn) writeRaw(ctx context.Context, v any) error {
	c.mu.Lock()
	ws := c.ws
	c.mu.Unlock()
	if ws == nil {
		return nil // not connected yet: dropping input is correct, queuing is not
	}
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	wctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return ws.Write(wctx, websocket.MessageText, b)
}

func (c *Conn) Close() {
	c.mu.Lock()
	c.closed = true
	ws, cancel := c.ws, c.cancel
	c.mu.Unlock()
	if ws != nil {
		_ = ws.Close(websocket.StatusNormalClosure, "bye")
	}
	if cancel != nil {
		cancel()
	}
}

func (c *Conn) isClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

func (c *Conn) emit(e Event) {
	select {
	case c.events <- e:
	default: // the UI is behind; dropping a stale event beats stalling the reader
	}
}
