package main

import (
	"bufio"
	"embed"
	"flag"
	"fmt"
	"os"
	"strings"
)

//go:embed all:web
var webFS embed.FS

const version = "1.1.0"

const banner = `
  ┌──────────────────────────────────────────┐
  │   ZlefRemote · agent                       │
  │   your phone is the trackpad               │
  └──────────────────────────────────────────┘
`

func main() {
	mode := flag.String("mode", "", "connection mode: lan | remote (prompted if empty)")
	port := flag.Int("port", 9783, "LAN mode listen port")
	relay := flag.String("relay", "remote.zlef.fr", "relay host for remote mode")
	noTelemetry := flag.Bool("no-telemetry", false, "disable the anonymous startup usage ping")
	remember := flag.Bool("remember", false, "persist this computer's encryption key locally so saved phones reconnect in one tap (room is derived from the stable key; default mints a fresh key per launch)")
	resetIdentity := flag.Bool("reset-identity", false, "with --remember: discard the stored key and mint a new one (rotates this computer's saved-device address)")
	machine := flag.Bool("machine", false, "emit machine-readable '@zr key=value' lines for front-ends (e.g. the xfce4-panel plugin)")
	update := flag.Bool("update", false, "update the agent in place to the latest release, then exit")
	force := flag.Bool("force", false, "with -update: reinstall even if already up to date")
	noUpdateCheck := flag.Bool("no-update-check", false, "don't check for a newer version on startup")
	ver := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	machineMode = *machine

	if *ver {
		fmt.Println("zlefremote-agent", version)
		return
	}

	if *update {
		if err := selfUpdate(*relay, *force); err != nil {
			fmt.Fprintln(os.Stderr, "update failed:", err)
			os.Exit(1)
		}
		return
	}

	// passive, best-effort "newer version available" hint (stderr only)
	if !*noUpdateCheck && !machineMode {
		go checkUpdateNotice(*relay)
	}

	if !machineMode {
		fmt.Print(banner)
	}

	inj := newInjector()
	name, goos := inj.HostInfo()
	emit("host", name)
	if !machineMode {
		fmt.Printf("  host: %s (%s)\n", name, goos)
	}

	// Identity: persistent (--remember) or ephemeral (default, fresh per launch).
	var key []byte
	var keyB64 string
	persistent := *remember
	if persistent {
		var ierr error
		key, keyB64, ierr = loadOrCreateIdentity(*resetIdentity)
		if ierr != nil {
			fmt.Fprintln(os.Stderr, "could not open the saved identity:", ierr)
			fmt.Fprintln(os.Stderr, "falling back to a fresh key for this session.")
			key, keyB64 = NewKey()
			persistent = false
		} else if !machineMode {
			fmt.Printf("  identity: remembered (saved phones reconnect in one tap)\n")
		}
	} else {
		key, keyB64 = NewKey()
	}
	emit("persistent", boolStr(persistent))

	sealer, err := NewSealer(key)
	if err != nil {
		fmt.Fprintln(os.Stderr, "crypto init failed:", err)
		os.Exit(1)
	}

	m := strings.ToLower(*mode)
	if m == "" {
		m = prompt()
	}

	switch m {
	case "lan", "l", "1":
		pingUsage("lan", *noTelemetry)
		if err := runLAN(sealer, inj, keyB64, *port); err != nil {
			fmt.Fprintln(os.Stderr, "lan mode error:", err)
			os.Exit(1)
		}
	case "remote", "r", "2":
		pingUsage("remote", *noTelemetry)
		if err := runRelay(sealer, inj, key, keyB64, *relay, persistent); err != nil {
			fmt.Fprintln(os.Stderr, "remote mode error:", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintln(os.Stderr, "unknown mode:", m)
		os.Exit(2)
	}
}

func boolStr(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

func prompt() string {
	fmt.Print(`
  How do you want to connect your phone?

    [1] Local network   — phone & PC on the same Wi-Fi (fastest, fully local)
    [2] Remote          — from anywhere, end-to-end encrypted via remote.zlef.fr
`)
	fmt.Print("  Choose 1 or 2: ")
	sc := bufio.NewScanner(os.Stdin)
	if sc.Scan() {
		return strings.TrimSpace(sc.Text())
	}
	return "1"
}
