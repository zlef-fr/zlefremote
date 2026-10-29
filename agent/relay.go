package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
)

// runRelay pairs the agent through remote.zlef.fr. The relay only ever sees a
// room code and opaque ciphertext — it cannot read any input.
//
// With persistent=true the agent asks the relay for a STABLE room derived from
// its saved key, so a remembered phone reconnects to the same address without a
// new QR. The pairing URL then carries &p=1, telling the phone this device is
// reconnectable and can be saved.
func runRelay(sealer *Sealer, inj Injector, scr Screener, br Brightener, clip Clipper, key []byte, keyB64, relayHost string, persistent bool) error {
	rooms := &roomKeeper{}
	if persistent {
		rooms.derived = deriveRoom(key)
	}
	for {
		err := relayOnce(sealer, inj, scr, br, clip, keyB64, relayHost, rooms)
		if errors.Is(err, errRoomTaken) {
			if rooms.taken(time.Now()) {
				// our own previous socket still holds the room on the relay; it
				// is reaped within a minute, so ask again shortly.
				time.Sleep(5 * time.Second)
				continue
			}
			if !machineMode {
				fmt.Printf("\n  saved-device address was busy — using a one-time room this session.\n")
			}
			continue
		}
		emit("event", "disconnect")
		if !machineMode {
			fmt.Printf("\n  relay disconnected (%v) — reconnecting in 3s…\n", err)
		}
		time.Sleep(3 * time.Second)
	}
}

// roomKeeper decides which room the agent asks the relay for on every connect.
//
// The room is the phone's only way back to this computer: a reconnect that
// lands in a new room strands every phone holding the old link, while the
// agent (and the panel showing it) look perfectly healthy. So once the relay
// has granted a room, every later connection asks for that same room — the
// key is unchanged, so the pairing link stays valid across relay restarts,
// network changes and suspend/resume.
type roomKeeper struct {
	derived string    // room derived from the remembered identity, "" if none
	held    string    // room the relay last granted this run
	takenAt time.Time // first room_taken while asking for held, zero otherwise
}

// reclaimWindow bounds how long we keep asking for a room the relay says is
// taken. Right after a reconnect the relay may still hold our previous, dead
// socket; its keepalive terminates that within 60 s, then the room is ours again.
const reclaimWindow = 2 * time.Minute

// want is the room to request ("" = let the relay pick one).
func (k *roomKeeper) want() string {
	if k.held != "" {
		return k.held
	}
	return k.derived
}

// persistentRoom reports whether room is the remembered identity's room — the
// only case where the pairing link may tell the phone it can save the device.
func (k *roomKeeper) persistentRoom(room string) bool {
	return k.derived != "" && room == k.derived
}

func (k *roomKeeper) granted(room string) {
	k.held = room
	k.takenAt = time.Time{}
}

// taken handles a room_taken answer and reports whether to ask for the same
// room again. A room we never held belongs to another live device (a derived
// room hash collision): give it up at once. A room we held is our own stale
// socket: retry until reclaimWindow runs out, then start over in a new room.
func (k *roomKeeper) taken(now time.Time) bool {
	if k.held != "" {
		if k.takenAt.IsZero() {
			k.takenAt = now
		}
		if now.Sub(k.takenAt) < reclaimWindow {
			return true
		}
	}
	k.derived, k.held, k.takenAt = "", "", time.Time{}
	return false
}

// Relay liveness. The relay answers {t:"ping"} with {t:"pong"}; when nothing at
// all has come back for relayDeadAfter the socket is dead even if TCP has not
// noticed (suspend/resume, a dropped Wi-Fi, a close frame lost at the edge), and
// the agent reconnects instead of waiting on a room that no longer exists.
const (
	relayPingEvery = 20 * time.Second
	relayDeadAfter = 50 * time.Second
	relayWriteWait = 10 * time.Second
)

var errRoomTaken = errors.New("room_taken")

func relayOnce(sealer *Sealer, inj Injector, scr Screener, br Brightener, clip Clipper, keyB64, relayHost string, rooms *roomKeeper) error {
	ctx := context.Background()
	wsURL := "wss://" + relayHost + "/ws"
	c, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		return err
	}
	defer c.CloseNow()
	c.SetReadLimit(128 * 1024)

	// One relay socket is shared by every phone's session plus the read loop, and
	// screen-view frames are pushed from per-session goroutines — serialize all
	// writes so concurrent Write calls never interleave.
	// A write that cannot complete within relayWriteWait means a dead socket:
	// the library then closes the connection and the read loop reconnects.
	var wmu sync.Mutex
	write := func(f frame) error {
		out, _ := json.Marshal(f)
		wmu.Lock()
		defer wmu.Unlock()
		wctx, cancel := context.WithTimeout(ctx, relayWriteWait)
		defer cancel()
		return c.Write(wctx, websocket.MessageText, out)
	}
	writeData := func(to int, payload string) {
		write(frame{T: "data", To: to, Payload: payload})
	}

	// register as a host, asking for the room we already hold this run (or the
	// remembered identity's room), so phones find us at the same address.
	if err := write(frame{T: "host", Room: rooms.want()}); err != nil {
		return err
	}

	var lastRx atomic.Int64
	lastRx.Store(time.Now().UnixNano())
	hbCtx, stopHeartbeat := context.WithCancel(ctx)
	defer stopHeartbeat()
	go func() {
		t := time.NewTicker(relayPingEvery)
		defer t.Stop()
		for {
			select {
			case <-hbCtx.Done():
				return
			case <-t.C:
				if time.Since(time.Unix(0, lastRx.Load())) > relayDeadAfter {
					c.CloseNow() // unblocks Read below → reconnect
					return
				}
				write(frame{T: "ping"})
			}
		}
	}()

	sessions := map[int]*Session{}
	roster := NewRoster()
	defer roster.Reset()
	defer func() {
		for _, se := range sessions {
			se.Close()
		}
	}()
	for {
		_, data, err := c.Read(ctx)
		if err != nil {
			return err
		}
		lastRx.Store(time.Now().UnixNano())
		var f frame
		if json.Unmarshal(data, &f) != nil {
			continue
		}
		switch f.T {
		case "hosted":
			// &p=1 marks this device as persistent so the phone offers to save
			// it. If the relay couldn't grant our derived room (collision), we
			// took a random one — drop the flag so the phone won't try to
			// silently re-derive an address that won't match.
			rooms.granted(f.Room)
			frag := "#k=" + keyB64
			if rooms.persistentRoom(f.Room) {
				frag += "&p=1"
			}
			url := fmt.Sprintf("https://%s/r/%s%s", relayHost, f.Room, frag)
			emit("mode", "remote")
			emit("room", f.Room)
			if !machineMode {
				fmt.Printf("\n  \033[1mRemote mode\033[0m — works from anywhere, end-to-end encrypted.\n")
				fmt.Printf("  Room code: \033[1;32m%s\033[0m\n\n", f.Room)
			}
			qrPath := printQR(url)
			emit("url", url)
			if qrPath != "" {
				emit("qr", qrPath)
			}
			emit("status", "waiting")
			if !machineMode {
				fmt.Printf("\n  Or open this on your phone:\n  \033[36m%s\033[0m\n\n", url)
				fmt.Printf("  Waiting for your phone…  ·  press Ctrl-C to stop\n\n")
			}
		case "peer":
			if f.Event == "join" {
				roster.Add(f.ID, f.IP)
			} else if f.Event == "leave" {
				if se := sessions[f.ID]; se != nil {
					se.Close()
				}
				delete(sessions, f.ID)
				roster.Remove(f.ID)
			}
		case "data":
			se := sessions[f.From]
			if se == nil {
				id := f.From
				se = NewSession(sealer, inj, scr, br, clip, func(payload string) { writeData(id, payload) })
				sessions[f.From] = se
			}
			se.Handle(f.Payload)
		case "error":
			if f.Error == "room_taken" {
				// our derived room is held by a different live device (a hash
				// collision — vanishingly rare). Signal runRelay to fall back to
				// a random room for the rest of this run.
				return errRoomTaken
			}
			return fmt.Errorf("relay error: %s", f.Error)
		case "closed":
			return fmt.Errorf("relay closed: %s", f.Error)
		}
	}
}
