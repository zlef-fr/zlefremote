package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// The agent can update itself in place from the relay's release manifest
// (GET https://<relay>/api/agent/version). Debian/Ubuntu users installed via
// the apt repo get updates through `apt upgrade` instead; this path is for the
// standalone binary / tarball install.

type releaseAsset struct {
	File   string `json:"file"`
	Sha256 string `json:"sha256"`
	URL    string `json:"url"`
}

type release struct {
	Version string                  `json:"version"`
	Assets  map[string]releaseAsset `json:"assets"`
}

func assetKey() string { return runtime.GOOS + "-" + runtime.GOARCH }

func fetchRelease(relayHost string) (*release, error) {
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Get("https://" + relayHost + "/api/agent/version")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("version endpoint: HTTP %d", resp.StatusCode)
	}
	var r release
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, err
	}
	return &r, nil
}

func notDigit(r rune) bool { return r < '0' || r > '9' }

// verCmp compares dotted numeric versions ("1.2.0" vs "1.10.0"): -1, 0, 1.
func verCmp(a, b string) int {
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	n := len(as)
	if len(bs) > n {
		n = len(bs)
	}
	for i := 0; i < n; i++ {
		var x, y int
		if i < len(as) {
			x, _ = strconv.Atoi(strings.TrimFunc(as[i], notDigit))
		}
		if i < len(bs) {
			y, _ = strconv.Atoi(strings.TrimFunc(bs[i], notDigit))
		}
		if x != y {
			if x < y {
				return -1
			}
			return 1
		}
	}
	return 0
}

// checkUpdateNotice prints a one-line hint to stderr if a newer agent exists.
// Best-effort and silent on any error; meant to run in a goroutine.
func checkUpdateNotice(relayHost string) {
	r, err := fetchRelease(relayHost)
	if err != nil || r == nil {
		return
	}
	if verCmp(version, r.Version) < 0 {
		fmt.Fprintf(os.Stderr,
			"\n  \033[33m↑ ZlefRemote %s is available\033[0m (you have %s) — run: zlefremote-agent -update\n",
			r.Version, version)
	}
}

// selfUpdate downloads the latest agent for this OS/arch, verifies its sha256,
// and atomically swaps the running binary. Returns nil if already current
// (unless force).
func selfUpdate(relayHost string, force bool) error {
	fmt.Printf("  checking %s for updates…\n", relayHost)
	r, err := fetchRelease(relayHost)
	if err != nil {
		return err
	}
	a, ok := r.Assets[assetKey()]
	if !ok {
		return fmt.Errorf("no published build for %s", assetKey())
	}
	if !force && verCmp(version, r.Version) >= 0 {
		fmt.Printf("  already up to date (%s). Use -update -force to reinstall.\n", version)
		return nil
	}
	fmt.Printf("  updating %s → %s …\n", version, r.Version)

	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if resolved, e := filepath.EvalSymlinks(exe); e == nil {
		exe = resolved
	}
	dir := filepath.Dir(exe)

	// stage beside the current binary so the final rename is atomic (same FS)
	tmp, err := os.CreateTemp(dir, ".zlefremote-update-*")
	if err != nil {
		return fmt.Errorf("cannot write to %s (try sudo): %w", dir, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Get(a.URL)
	if err != nil {
		tmp.Close()
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		tmp.Close()
		return fmt.Errorf("download: HTTP %d", resp.StatusCode)
	}

	h := sha256.New()
	if _, err := io.Copy(tmp, io.TeeReader(resp.Body, h)); err != nil {
		tmp.Close()
		return err
	}
	tmp.Close()

	if sum := hex.EncodeToString(h.Sum(nil)); !strings.EqualFold(sum, a.Sha256) {
		return fmt.Errorf("checksum mismatch (got %s…, want %s…)", sum[:12], a.Sha256[:12])
	}
	if err := os.Chmod(tmpName, 0o755); err != nil {
		return err
	}

	// keep a .bak so a failed swap can roll back (and Windows can rename a
	// running image out of the way)
	bak := exe + ".bak"
	_ = os.Remove(bak)
	if err := os.Rename(exe, bak); err != nil {
		return fmt.Errorf("cannot replace %s (try sudo): %w", exe, err)
	}
	if err := os.Rename(tmpName, exe); err != nil {
		_ = os.Rename(bak, exe) // roll back
		return err
	}
	_ = os.Remove(bak)
	fmt.Printf("  \033[32m✓ updated to %s\033[0m  (%s)\n  restart the agent to use it.\n", r.Version, exe)
	return nil
}
