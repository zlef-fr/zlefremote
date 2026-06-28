package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// Roster tracks the phones currently connected to this host (one entry per
// client). It is the single place that renders "who is connected" for both
// front-ends:
//
//   - the human CLI (a short connect/disconnect line + a live count), and
//   - the machine protocol consumed by the xfce4-panel plugin:
//       @zr peer=join <id> <ip>
//       @zr peer=leave <id>
//       @zr clients=<n>
//
// The IP is best-effort identifying metadata so the machine's owner can see who
// is controlling it. It may be empty ("nothing") when unknown — the relay
// forwards the phone's real IP, LAN mode uses the socket peer address.
type Roster struct {
	mu    sync.Mutex
	peers map[int]string // client id -> ip ("" if unknown)
}

func NewRoster() *Roster { return &Roster{peers: map[int]string{}} }

// Add records a newly connected client and reports the change.
func (r *Roster) Add(id int, ip string) {
	r.mu.Lock()
	r.peers[id] = ip
	n, list := r.snapshotLocked()
	r.mu.Unlock()
	emit("peer", fmt.Sprintf("join %d %s", id, ip))
	emit("clients", strconv.Itoa(n))
	if !machineMode {
		fmt.Printf("\n  \033[1;32m●\033[0m phone connected%s   ·   %s\n", who(ip), countLabel(n))
		printRoster(list)
	}
}

// Remove drops a disconnected client and reports the change.
func (r *Roster) Remove(id int) {
	r.mu.Lock()
	ip, had := r.peers[id]
	delete(r.peers, id)
	n, list := r.snapshotLocked()
	r.mu.Unlock()
	if !had {
		return
	}
	emit("peer", fmt.Sprintf("leave %d", id))
	emit("clients", strconv.Itoa(n))
	if !machineMode {
		fmt.Printf("\n  \033[1;31m○\033[0m phone disconnected%s   ·   %s\n", who(ip), countLabel(n))
		printRoster(list)
	}
}

// Reset clears everyone (e.g. the relay dropped) and reports an empty roster.
func (r *Roster) Reset() {
	r.mu.Lock()
	had := len(r.peers) > 0
	r.peers = map[int]string{}
	r.mu.Unlock()
	if had {
		emit("clients", "0")
	}
}

// Count returns the number of connected clients.
func (r *Roster) Count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.peers)
}

// snapshotLocked returns the current count and a sorted list of "ip" strings.
// Caller must hold r.mu.
func (r *Roster) snapshotLocked() (int, []string) {
	ids := make([]int, 0, len(r.peers))
	for id := range r.peers {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	list := make([]string, 0, len(ids))
	for _, id := range ids {
		ip := r.peers[id]
		if ip == "" {
			ip = "unknown"
		}
		list = append(list, ip)
	}
	return len(ids), list
}

func who(ip string) string {
	if ip == "" {
		return ""
	}
	return " from \033[36m" + ip + "\033[0m"
}

func countLabel(n int) string {
	if n == 1 {
		return "1 phone connected"
	}
	return strconv.Itoa(n) + " phones connected"
}

// printRoster lists the connected clients' IPs, one per line, when any.
func printRoster(list []string) {
	if len(list) == 0 {
		return
	}
	fmt.Printf("    connected: %s\n", strings.Join(list, ", "))
}
