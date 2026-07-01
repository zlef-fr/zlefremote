package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// transport frame shared by LAN + relay: {"t":"data","payload":"<sealed>"} plus
// relay-only control verbs.
type frame struct {
	T       string `json:"t"`
	Payload string `json:"payload,omitempty"`
	Room    string `json:"room,omitempty"`
	ID      int    `json:"id,omitempty"`
	From    int    `json:"from,omitempty"`
	To      int    `json:"to,omitempty"`
	Event   string `json:"event,omitempty"`
	IP      string `json:"ip,omitempty"`
	Error   string `json:"error,omitempty"`
}

func lanIP() string {
	addrs, _ := net.InterfaceAddrs()
	for _, a := range addrs {
		if ipn, ok := a.(*net.IPNet); ok && !ipn.IP.IsLoopback() {
			if ip4 := ipn.IP.To4(); ip4 != nil {
				return ip4.String()
			}
		}
	}
	return "127.0.0.1"
}

// peerIP extracts the client IP from an inbound LAN request, stripping the
// ephemeral port. On a LAN this is the phone's local address.
func peerIP(r *http.Request) string {
	addr := r.RemoteAddr
	if host, _, err := net.SplitHostPort(addr); err == nil {
		return host
	}
	return addr
}

func runLAN(sealer *Sealer, inj Injector, scr Screener, keyB64 string, port int) error {
	sub, err := fs.Sub(webFS, "web")
	if err != nil {
		return err
	}
	mux := http.NewServeMux()
	mux.Handle("/app/", http.StripPrefix("/app/", http.FileServer(http.FS(sub))))
	// PWA manifest + service worker are referenced at the origin root; serve
	// them from the embed so the installable shell also works in LAN mode.
	mux.HandleFunc("/sw.js", func(w http.ResponseWriter, r *http.Request) {
		if b, err := fs.ReadFile(sub, "sw.js"); err == nil {
			w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
			w.Header().Set("Service-Worker-Allowed", "/")
			w.Write(b)
		} else {
			http.NotFound(w, r)
		}
	})
	mux.HandleFunc("/app.webmanifest", func(w http.ResponseWriter, r *http.Request) {
		if b, err := fs.ReadFile(sub, "app.webmanifest"); err == nil {
			w.Header().Set("Content-Type", "application/manifest+json; charset=utf-8")
			w.Write(b)
		} else {
			http.NotFound(w, r)
		}
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			b, _ := fs.ReadFile(sub, "index.html")
			w.Write(b)
			return
		}
		http.NotFound(w, r)
	})
	roster := NewRoster()
	var idMu sync.Mutex
	var nextID int
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		// Not TLS verification: this is the WS Origin check. On a LAN the host
		// is reached by IP/hostname from varied origins; the real access gate is
		// the AES-256-GCM key — a client without it cannot produce a frame that
		// decrypts, so it can never inject input.
		c, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			return
		}
		defer c.CloseNow()
		idMu.Lock()
		nextID++
		id := nextID
		idMu.Unlock()
		roster.Add(id, peerIP(r))
		defer roster.Remove(id)
		ctx := context.Background()
		// The screen-view stream pushes frames from a goroutine while the read
		// loop may also reply — serialize writes to this phone's socket.
		var wmu sync.Mutex
		se := NewSession(sealer, inj, scr, func(payload string) {
			out, _ := json.Marshal(frame{T: "data", Payload: payload})
			wmu.Lock()
			c.Write(ctx, websocket.MessageText, out)
			wmu.Unlock()
		})
		defer se.Close()
		for {
			_, data, err := c.Read(ctx)
			if err != nil {
				return
			}
			var f frame
			if json.Unmarshal(data, &f) != nil || f.T != "data" {
				continue
			}
			se.Handle(f.Payload)
		}
	})

	ip := lanIP()
	url := fmt.Sprintf("http://%s:%d/#k=%s", ip, port, keyB64)
	emit("mode", "lan")
	if !machineMode {
		fmt.Printf("\n  \033[1mLAN mode\033[0m — make sure your phone is on the same Wi-Fi.\n\n")
	}
	qrPath := printQR(url)
	emit("url", url)
	if qrPath != "" {
		emit("qr", qrPath)
	}
	emit("status", "waiting")
	if !machineMode {
		fmt.Printf("\n  Or open this on your phone:\n  \033[36m%s\033[0m\n\n", url)
		fmt.Printf("  Listening on %s:%d  ·  press Ctrl-C to stop\n\n", ip, port)
	}

	srv := &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.SetFlags(log.Ltime)
	return srv.ListenAndServe()
}
