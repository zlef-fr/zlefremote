package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
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

// serveEmbed writes one HTML file out of the agent's embedded client bundle.
func serveEmbed(w http.ResponseWriter, sub fs.FS, name string) {
	b, err := fs.ReadFile(sub, name)
	if err != nil {
		http.NotFound(w, &http.Request{})
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(b)
}

func runLAN(sealer *Sealer, inj Injector, scr Screener, br Brightener, clip Clipper, keyB64 string, port int) error {
	sub, err := fs.Sub(webFS, "web")
	if err != nil {
		return err
	}
	mux := http.NewServeMux()
	mux.Handle("/app/", http.StripPrefix("/app/", http.FileServer(http.FS(sub))))
	// the desktop remote — same key, same socket, a UI for a mouse and a real
	// keyboard. Its modules load from /app/js/, so both UIs share one transport.
	// (its own assets live one level down in the embed, hence a second sub-FS)
	deskFS, err := fs.Sub(sub, "desk")
	if err != nil {
		return err
	}
	mux.Handle("/desk/", http.StripPrefix("/desk/", http.FileServer(http.FS(deskFS))))
	mux.HandleFunc("/d", func(w http.ResponseWriter, r *http.Request) { serveEmbed(w, sub, "desk/index.html") })
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
		switch r.URL.Path {
		case "/":
			serveEmbed(w, sub, "index.html")
		case "/d/":
			serveEmbed(w, sub, "desk/index.html")
		default:
			http.NotFound(w, r)
		}
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
		se := NewSession(sealer, inj, scr, br, clip, func(payload string) {
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
	// HTTPS, not HTTP: browsers only expose crypto.subtle (the AES-256-GCM this
	// protocol runs on) in a secure context, and a LAN IP over plain HTTP is not
	// one. See lan_tls.go.
	cert, certErr := lanCertificate(ip)
	scheme := "https"
	if certErr != nil {
		// no certificate = no WebCrypto in the browser; say so instead of
		// serving a page that would fail with an unexplained "missing key".
		fmt.Fprintln(os.Stderr, "could not create the local certificate:", certErr)
		fmt.Fprintln(os.Stderr, "LAN mode needs HTTPS for the browser's crypto — use remote mode instead.")
		return certErr
	}
	url := fmt.Sprintf("%s://%s:%d/#k=%s", scheme, ip, port, keyB64)
	emit("mode", "lan")
	if !machineMode {
		fmt.Printf("\n  \033[1mLAN mode\033[0m — make sure your phone is on the same Wi-Fi.\n")
		fmt.Printf("  This computer serves its own certificate, so the browser warns once:\n")
		fmt.Printf("  tap \033[1mAdvanced → Proceed\033[0m. Your input stays end-to-end encrypted either way.\n\n")
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

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		TLSConfig:         &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12},
	}
	log.SetFlags(log.Ltime)
	// certs come from TLSConfig, hence the empty file arguments
	return srv.ListenAndServeTLS("", "")
}
