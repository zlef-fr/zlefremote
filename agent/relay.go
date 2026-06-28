package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/coder/websocket"
)

// runRelay pairs the agent through remote.zlef.fr. The relay only ever sees a
// room code and opaque ciphertext — it cannot read any input.
func runRelay(sealer *Sealer, inj Injector, keyB64, relayHost string) error {
	for {
		err := relayOnce(sealer, inj, keyB64, relayHost)
		emit("event", "disconnect")
		if !machineMode {
			fmt.Printf("\n  relay disconnected (%v) — reconnecting in 3s…\n", err)
		}
		time.Sleep(3 * time.Second)
	}
}

func relayOnce(sealer *Sealer, inj Injector, keyB64, relayHost string) error {
	ctx := context.Background()
	wsURL := "wss://" + relayHost + "/ws"
	c, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		return err
	}
	defer c.CloseNow()
	c.SetReadLimit(128 * 1024)

	// register as a host
	hb, _ := json.Marshal(frame{T: "host"})
	if err := c.Write(ctx, websocket.MessageText, hb); err != nil {
		return err
	}

	sessions := map[int]*Session{}
	roster := NewRoster()
	defer roster.Reset()
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
			url := fmt.Sprintf("https://%s/r/%s#k=%s", relayHost, f.Room, keyB64)
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
				delete(sessions, f.ID)
				roster.Remove(f.ID)
			}
		case "data":
			se := sessions[f.From]
			if se == nil {
				se = NewSession(sealer, inj)
				sessions[f.From] = se
			}
			if reply := se.Handle(f.Payload); reply != "" {
				out, _ := json.Marshal(frame{T: "data", To: f.From, Payload: reply})
				c.Write(ctx, websocket.MessageText, out)
			}
		case "error":
			return fmt.Errorf("relay error: %s", f.Error)
		case "closed":
			return fmt.Errorf("relay closed: %s", f.Error)
		}
	}
}
