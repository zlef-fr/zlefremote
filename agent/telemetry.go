package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"runtime"
	"time"
)

// telemetryDefault is the compile-time default. Disable telemetry at BUILD time:
//
//	go build -ldflags "-X main.telemetryDefault=off"
//
// (the build.sh "notrack" option does this for you).
var telemetryDefault = "on"

const telemetryURL = "https://remote.zlef.fr/api/agent/ping"

// telemetryOff reports whether the anonymous ping is disabled, by any of:
//   - the --no-telemetry flag
//   - DO_NOT_TRACK=1  or  ZLEFREMOTE_NO_TELEMETRY (any value)
//   - a build with -ldflags "-X main.telemetryDefault=off"
func telemetryOff(flagDisabled bool) bool {
	return flagDisabled ||
		telemetryDefault == "off" ||
		os.Getenv("DO_NOT_TRACK") == "1" ||
		os.Getenv("ZLEFREMOTE_NO_TELEMETRY") != ""
}

// pingUsage fires a single anonymous startup ping (agent version, OS, arch and
// the chosen mode — nothing else). It is best-effort and fully fire-and-forget:
// it runs in a goroutine, never blocks startup, and silently ignores all errors.
func pingUsage(mode string, flagDisabled bool) {
	if telemetryOff(flagDisabled) {
		return
	}
	go func() {
		body, _ := json.Marshal(map[string]string{
			"event":   "start",
			"version": version,
			"os":      runtime.GOOS,
			"arch":    runtime.GOARCH,
			"mode":    mode,
		})
		req, err := http.NewRequest(http.MethodPost, telemetryURL, bytes.NewReader(body))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "zlefremote-agent/"+version)
		client := &http.Client{Timeout: 4 * time.Second}
		if resp, err := client.Do(req); err == nil {
			resp.Body.Close()
		}
	}()
}
