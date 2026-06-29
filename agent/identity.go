package main

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"strings"
)

// Persistent device identity (opt-in, --remember).
//
// By default the agent mints a fresh 256-bit key on every launch and the relay
// hands it a random room code — nothing is stored, and a session is unreachable
// once it ends. With --remember the key is persisted locally (0600) so the same
// computer keeps a STABLE identity across restarts, and the relay room is
// DERIVED from that key. A saved phone holding only the key can then re-derive
// the exact room and reconnect in one tap, with no QR rescan.
//
// The derivation is one-way (sha256), so the room leaks nothing about the key,
// and only someone who already holds the key (i.e. the legitimate device) can
// compute it. This file and public/app/js/crypto.js MUST derive identically.

// roomAlphabet matches lib/rooms.js (no 0/O/1/I ambiguity), 32 symbols → 5 bits.
const roomAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// roomDomain namespaces the hash so the room can never collide with any other
// use of the key. Changing it rotates every persistent device's room.
const roomDomain = "zlefremote-room-v1\x00"

// deriveRoom maps a 32-byte key to its stable 6-character relay room code.
func deriveRoom(key []byte) string {
	h := sha256.Sum256(append([]byte(roomDomain), key...))
	var b strings.Builder
	b.Grow(6)
	for i := 0; i < 6; i++ {
		b.WriteByte(roomAlphabet[h[i]&31]) // 32 symbols → low 5 bits, uniform
	}
	return b.String()
}

// identityPath returns <user-config>/zlefremote/identity, creating the dir.
func identityPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	d := filepath.Join(dir, "zlefremote")
	if err := os.MkdirAll(d, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(d, "identity"), nil
}

// loadOrCreateIdentity returns the persistent key (raw + base64url), reading it
// from disk or minting and saving a fresh one on first use. With reset=true any
// existing key is discarded and replaced (rotates the device's stable room).
func loadOrCreateIdentity(reset bool) (key []byte, keyB64 string, err error) {
	p, err := identityPath()
	if err != nil {
		return nil, "", err
	}
	if !reset {
		if raw, rerr := os.ReadFile(p); rerr == nil {
			s := strings.TrimSpace(string(raw))
			if k, derr := b64.DecodeString(s); derr == nil && len(k) == 32 {
				return k, s, nil
			}
		}
	}
	k, b := NewKey()
	if err := os.WriteFile(p, []byte(b), 0o600); err != nil {
		return nil, "", err
	}
	return k, b, nil
}
