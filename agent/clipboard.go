package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"time"
	"unicode/utf8"
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

// ImageClipper is the optional half of a Clipper that can also put a picture on
// the host clipboard (a screenshot copied on the client, pasted over there).
// Text tools don't always carry images — xsel and pbcopy can't — so it is probed
// on its own and advertised to clients as `cap.clipimg`.
type ImageClipper interface {
	ImageAvailable() bool
	WriteImage(png []byte) error
}

// Images are far bigger than any paste of text, so they travel in chunks
// ({t:'climg'}) and are reassembled here. 12 MB of PNG is a full-resolution 4K
// screenshot with room to spare; past that the client says so up front.
const maxClipImageBytes = 12 << 20

// errNoImage: this backend has no way to put a picture on the clipboard.
var errNoImage = errors.New("clipboard backend cannot carry images")

func canWriteImage(c Clipper) bool {
	ic, ok := c.(ImageClipper)
	return ok && c.Available() && ic.ImageAvailable()
}

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
	// extra environment for both commands, appended to the agent's own. Used
	// on Wayland, where wl-copy/wl-paste need a WAYLAND_DISPLAY the agent's
	// process may never have inherited.
	env []string
	// image puts a PNG on the clipboard; nil = this backend can't (xsel).
	image func(c execClipper, png []byte) error
}

// command builds one clipboard invocation with the backend's extra environment.
func (c execClipper) command(ctx context.Context, argv []string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	if len(c.env) > 0 {
		cmd.Env = append(os.Environ(), c.env...)
	}
	return cmd
}

func (c execClipper) Available() bool { return len(c.readCmd) > 0 && len(c.writeCmd) > 0 }

func (c execClipper) Read() (string, error) {
	if !c.Available() {
		return "", nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	out, err := c.command(ctx, c.readCmd).Output()
	if err != nil {
		return "", err
	}
	// a picture is not text. When the clipboard holds only an image, some
	// readers hand its bytes over anyway — xclip does whenever xclip itself owns
	// the selection, which is exactly the case right after we put a pasted image
	// there — and those bytes would echo back to the client as "copied text".
	if !utf8.Valid(out) || bytes.IndexByte(out, 0) >= 0 {
		return "", nil
	}
	return string(out), nil
}

func (c execClipper) Write(s string) error {
	if !c.Available() {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := c.command(ctx, c.writeCmd)
	cmd.Stdin = strings.NewReader(s)
	return cmd.Run()
}

func (c execClipper) ImageAvailable() bool { return c.Available() && c.image != nil }

func (c execClipper) WriteImage(png []byte) error {
	if !c.ImageAvailable() {
		return errNoImage
	}
	return c.image(c, png)
}

// pngOnStdin is the image writer for tools that take the picture on stdin and a
// MIME type on the command line (wl-copy, xclip).
func pngOnStdin(argv ...string) func(execClipper, []byte) error {
	return func(c execClipper, png []byte) error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		cmd := c.command(ctx, argv)
		cmd.Stdin = bytes.NewReader(png)
		return cmd.Run()
	}
}

// pngViaFile is the image writer for tools that only read a picture from disk
// (osascript on macOS, PowerShell on Windows): the PNG goes to a temp file whose
// path reaches the tool through ZR_CLIP_PNG — an environment variable, so no
// path ever has to survive a round of shell or AppleScript quoting.
func pngViaFile(argv ...string) func(execClipper, []byte) error {
	return func(c execClipper, png []byte) error {
		f, err := os.CreateTemp("", "zlefremote-clip-*.png")
		if err != nil {
			return err
		}
		path := f.Name()
		defer os.Remove(path)
		if _, err := f.Write(png); err != nil {
			f.Close()
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		cmd := c.command(ctx, argv)
		cmd.Env = append(cmd.Environ(), "ZR_CLIP_PNG="+path)
		if out, err := cmd.CombinedOutput(); err != nil {
			if msg := strings.TrimSpace(string(out)); msg != "" {
				return errors.New(err.Error() + ": " + msg)
			}
			return err
		}
		return nil
	}
}

// have reports whether a binary exists on PATH (per-OS constructors use it to
// pick the first working tool).
func have(bin string) bool {
	_, err := exec.LookPath(bin)
	return err == nil
}
