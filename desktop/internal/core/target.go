package core

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// Target is one machine this client can drive: where its WebSocket lives, which
// relay room to join (empty in LAN mode) and the E2EE key that proves we are
// allowed in.
//
// It is built from the very pairing link the agent prints/QR-encodes, so there
// is nothing new for the user to learn:
//
//	remote : https://remote.zlef.fr/r/AB12CD#k=<key>[&p=1]
//	LAN    : http://192.168.1.24:9783/#k=<key>
type Target struct {
	Label      string `json:"label"`
	WSURL      string `json:"ws"`
	Room       string `json:"room,omitempty"`
	KeyB64     string `json:"key"`
	Persistent bool   `json:"persistent,omitempty"`
	Relay      bool   `json:"relay"`
}

var roomPath = regexp.MustCompile(`^/r/([A-Za-z0-9]{4,8})/?$`)

// ParseTarget accepts a pairing URL (with its #k= fragment) and turns it into a
// Target. Whitespace and surrounding quotes are tolerated because links usually
// arrive through a chat message or a terminal copy.
func ParseTarget(raw string) (Target, error) {
	s := strings.TrimSpace(raw)
	s = strings.Trim(s, "\"'<>")
	if s == "" {
		return Target{}, errors.New("empty link")
	}
	if !strings.Contains(s, "://") {
		// bare host:port or host — assume a LAN agent over plain HTTP
		s = "http://" + s
	}
	u, err := url.Parse(s)
	if err != nil {
		return Target{}, fmt.Errorf("not a link: %w", err)
	}
	key, persistent := parseFragment(u.Fragment)
	if key == "" {
		return Target{}, errors.New("this link carries no key (#k=…) — copy the whole link the agent printed")
	}
	if _, err := SealerFromKeyB64(key); err != nil {
		return Target{}, err
	}

	t := Target{KeyB64: key, Persistent: persistent}
	scheme := "ws"
	if u.Scheme == "https" || u.Scheme == "wss" {
		scheme = "wss"
	}
	host := u.Host
	if host == "" {
		return Target{}, errors.New("this link has no host")
	}
	if m := roomPath.FindStringSubmatch(u.Path); m != nil {
		t.Room = strings.ToUpper(m[1])
		t.Relay = true
		t.Label = t.Room
	} else {
		t.Label = host
	}
	t.WSURL = fmt.Sprintf("%s://%s/ws", scheme, host)
	return t, nil
}

// parseFragment pulls the key and the persistent marker out of "k=…&p=1".
func parseFragment(frag string) (key string, persistent bool) {
	for _, part := range strings.FieldsFunc(frag, func(r rune) bool { return r == '&' || r == ';' }) {
		k, v, _ := strings.Cut(part, "=")
		switch k {
		case "k":
			key = v
		case "p":
			persistent = v == "1"
		}
	}
	return
}

// Describe is the one-line summary shown in the device list / window title.
func (t Target) Describe() string {
	if t.Relay {
		return "room " + t.Room
	}
	host := strings.TrimPrefix(strings.TrimPrefix(t.WSURL, "ws://"), "wss://")
	return strings.TrimSuffix(host, "/ws")
}

// Same reports whether two targets address the same machine+key (used to avoid
// saving a device twice).
func (t Target) Same(o Target) bool {
	return t.WSURL == o.WSURL && t.Room == o.Room && t.KeyB64 == o.KeyB64
}
