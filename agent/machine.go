package main

import (
	"fmt"
	"os"
)

// machineMode, when enabled with -machine, makes the agent emit a small,
// stable, line-oriented protocol on stdout in addition to the human banner.
// Front-ends (e.g. the xfce4-panel plugin) parse these lines instead of
// scraping the ANSI-decorated human output. Every machine line is:
//
//	@zr <key>=<value>
//
// Emitted keys:
//
//	mode=lan|remote      chosen connection mode (once, at start)
//	url=<pairing-url>     the URL/QR payload the phone opens (contains #k=…)
//	qr=<png-path>         absolute path to the rendered QR PNG
//	status=waiting        agent is up and waiting for a phone
//	event=paired          a phone completed the handshake
//	event=disconnect      the relay dropped (remote mode auto-reconnects)
//	peer=join <id> <ip>   a phone connected (ip may be empty = unknown)
//	peer=leave <id>       a phone disconnected
//	clients=<n>           current connected-client count (after each change)
var machineMode bool

// emit writes one machine line if machine mode is on. It flushes immediately
// (stdout is line-buffered when piped) so the front-end sees events live.
func emit(key, val string) {
	if !machineMode {
		return
	}
	fmt.Printf("@zr %s=%s\n", key, val)
	os.Stdout.Sync()
}
