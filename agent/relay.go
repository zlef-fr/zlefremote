package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
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
func runRelay(sealer *Sealer, inj Injector, scr Screener, key []byte, keyB64, relayHost string, persistent bool) error {
	desiredRoom := ""
	if persistent {
		desiredRoom = deriveRoom(key)
	}
	for {
		err := relayOnce(sealer, inj, scr, keyB64, relayHost, desiredRoom, persistent)
		if errors.Is(err, errRoomTaken) {
			// stable room unavailable (collision) — degrade to a random room.
			desiredRoom, persistent = "", false
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

var errRoomTaken = errors.New("room_taken")

func relayOnce(sealer *Sealer, inj Injector, scr Screener, keyB64, relayHost, desiredRoom string, persistent bool) error {
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
	var wmu sync.Mutex
	writeData := func(to int, payload string) {
		out, _ := json.Marshal(frame{T: "data", To: to, Payload: payload})
		wmu.Lock()
		c.Write(ctx, websocket.MessageText, out)
		wmu.Unlock()
	}

	// register as a host. When we have a persistent identity, ask the relay for
	// our derived room so saved phones find us at the same address every time.
	hb, _ := json.Marshal(frame{T: "host", Room: desiredRoom})
	if err := c.Write(ctx, websocket.MessageText, hb); err != nil {
		return err
	}

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
			frag := "#k=" + keyB64
			if persistent && f.Room == desiredRoom {
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
				se = NewSession(sealer, inj, scr, func(payload string) { writeData(id, payload) })
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
