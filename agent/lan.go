package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
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

func runLAN(sealer *Sealer, inj Injector, keyB64 string, port int) error {
	sub, err := fs.Sub(webFS, "web")
	if err != nil {
		return err
	}
	mux := http.NewServeMux()
	mux.Handle("/app/", http.StripPrefix("/app/", http.FileServer(http.FS(sub))))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			b, _ := fs.ReadFile(sub, "index.html")
			w.Write(b)
			return
		}
		http.NotFound(w, r)
	})
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
		se := NewSession(sealer, inj)
		ctx := context.Background()
		for {
			_, data, err := c.Read(ctx)
			if err != nil {
				return
			}
			var f frame
			if json.Unmarshal(data, &f) != nil || f.T != "data" {
				continue
			}
			if reply := se.Handle(f.Payload); reply != "" {
				out, _ := json.Marshal(frame{T: "data", Payload: reply})
				c.Write(ctx, websocket.MessageText, out)
			}
		}
	})

	ip := lanIP()
	url := fmt.Sprintf("http://%s:%d/#k=%s", ip, port, keyB64)
	fmt.Printf("\n  \033[1mLAN mode\033[0m — make sure your phone is on the same Wi-Fi.\n\n")
	printQR(url)
	fmt.Printf("\n  Or open this on your phone:\n  \033[36m%s\033[0m\n\n", url)
	fmt.Printf("  Listening on %s:%d  ·  press Ctrl-C to stop\n\n", ip, port)

	srv := &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.SetFlags(log.Ltime)
	return srv.ListenAndServe()
}
