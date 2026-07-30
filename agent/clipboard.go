package main

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// Clipper reads and writes the host's clipboard so a connected client can share
// copy/paste with it. Like the Brightener it is exec-based and cgo-free (no
// robotgo build tag needed), because every OS ships a clipboard CLI:
// wl-copy/xclip/xsel on Linux, pbcopy/pbpaste on macOS, PowerShell on Windows.
// Availability is probed once at startup; when false the client hides its
// clipboard sync and the host never spawns a watcher.
type Clipper interface {
	Available() bool
	Read() (string, error)
	Write(s string) error
}

// Clipboard payloads ride the same 64 KB relay frame as everything else (text →
// JSON → sealed → base64 inflates ~1.8×), so refuse to sync anything bigger.
// 16 KB of text is a very long paste and still leaves plenty of headroom.
const maxClipBytes = 16 * 1024

// clipPollInterval is how often a watching session re-reads the host clipboard.
// Each read spawns a small CLI, so keep it lazy: only sessions that asked for
// `clipwatch` poll at all.
const clipPollInterval = 1200 * time.Millisecond

// noClipper is the fallback when no clipboard tool is installed.
type noClipper struct{}

func (noClipper) Available() bool       { return false }
func (noClipper) Read() (string, error) { return "", nil }
func (noClipper) Write(s string) error  { return nil }

// execClipper drives a read command and a write command (which takes the text on
// stdin). Both are resolved once at construction, so Available() is just "did we
// find a tool".
type execClipper struct {
	readCmd  []string
	writeCmd []string
}

func (c execClipper) Available() bool { return len(c.readCmd) > 0 && len(c.writeCmd) > 0 }

func (c execClipper) Read() (string, error) {
	if !c.Available() {
		return "", nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, c.readCmd[0], c.readCmd[1:]...).Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (c execClipper) Write(s string) error {
	if !c.Available() {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, c.writeCmd[0], c.writeCmd[1:]...)
	cmd.Stdin = strings.NewReader(s)
	return cmd.Run()
}

// have reports whether a binary exists on PATH (per-OS constructors use it to
// pick the first working tool).
func have(bin string) bool {
	_, err := exec.LookPath(bin)
	return err == nil
}
