package main

import "testing"

func feed(s *Session, lines ...string) {
	for _, l := range lines {
		s.Handle(l)
	}
}

func TestPairingFlow(t *testing.T) {
	s := NewSession()
	paired := 0
	s.OnPaired = func() { paired++ }

	feed(s,
		"  ┌──────────────────────────────────────────┐", // banner noise is ignored
		"@zr mode=remote",
		"@zr url=https://remote.zlef.fr/r/ABC123#k=secret",
		"@zr qr=C:\\Users\\z\\AppData\\Local\\Temp\\zlefremote-qr.png",
		"@zr status=waiting",
	)
	if s.Status != StatusWaiting {
		t.Fatalf("status = %v, want waiting", s.Status)
	}
	if s.URL != "https://remote.zlef.fr/r/ABC123#k=secret" {
		t.Fatalf("url = %q", s.URL)
	}
	if s.QRPath == "" {
		t.Fatal("qr path not captured")
	}

	feed(s, "@zr event=paired")
	if s.Status != StatusPaired || paired != 1 {
		t.Fatalf("paired: status=%v cb=%d", s.Status, paired)
	}
}

func TestPeerRoster(t *testing.T) {
	s := NewSession()
	feed(s,
		"@zr peer=join 2 192.168.1.42",
		"@zr peer=join 1 10.0.0.7",
		"@zr clients=2",
	)
	got := s.PeerList()
	if len(got) != 2 {
		t.Fatalf("peers = %v", got)
	}
	// ordered by id so the roster doesn't reshuffle between repaints
	if got[0].ID != 1 || got[0].IP != "10.0.0.7" || got[1].ID != 2 {
		t.Fatalf("unordered roster: %v", got)
	}

	feed(s, "@zr peer=leave 1")
	if len(s.PeerList()) != 1 {
		t.Fatalf("leave ignored: %v", s.PeerList())
	}

	// an ip-less join (relay could not resolve the peer address)
	feed(s, "@zr peer=join 5")
	if ip, ok := s.Peers[5]; !ok || ip != "" {
		t.Fatalf("ip-less join: %q %v", ip, ok)
	}

	// clients=0 is authoritative: the relay dropped, roster resets
	feed(s, "@zr clients=0")
	if len(s.Peers) != 0 {
		t.Fatalf("roster not reset: %v", s.Peers)
	}
}

func TestDisconnectReturnsToWaiting(t *testing.T) {
	s := NewSession()
	bye := 0
	s.OnDisconnect = func() { bye++ }
	feed(s, "@zr status=waiting", "@zr peer=join 1 1.2.3.4", "@zr event=paired",
		"@zr event=disconnect")
	if s.Status != StatusWaiting {
		t.Fatalf("status = %v, want waiting after disconnect", s.Status)
	}
	if len(s.Peers) != 0 || bye != 1 {
		t.Fatalf("peers=%v cb=%d", s.Peers, bye)
	}
}

func TestPersistentIdentity(t *testing.T) {
	s := NewSession()
	feed(s, "@zr persistent=1")
	if !s.Persistent {
		t.Fatal("persistent not set")
	}
	s.Reset()
	if s.Persistent || s.URL != "" || s.Status != StatusIdle {
		t.Fatalf("reset left state behind: %+v", s)
	}
}

func TestMalformedLinesAreIgnored(t *testing.T) {
	s := NewSession()
	feed(s, "@zr", "@zr novalue", "not ours", "@zr peer=join notanumber",
		"@zr clients=abc", "")
	if s.Status != StatusIdle || len(s.Peers) != 0 {
		t.Fatalf("garbage changed state: %+v", s)
	}
}

func TestLineSplitterHandlesPartialChunks(t *testing.T) {
	var sp LineSplitter
	var got []string
	emit := func(l string) { got = append(got, l) }

	sp.Feed([]byte("@zr status=wai"), emit)
	if len(got) != 0 {
		t.Fatalf("emitted a partial line: %v", got)
	}
	sp.Feed([]byte("ting\r\n@zr event=pai"), emit)
	sp.Feed([]byte("red\n"), emit)

	want := []string{"@zr status=waiting", "@zr event=paired"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("line %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestLangFallback(t *testing.T) {
	setLang("fr-FR")
	if t2 := t2Copy(); t2 != "Copier le lien" {
		t.Fatalf("fr copy = %q", t2)
	}
	setLang("en-GB")
	if t2 := t2Copy(); t2 != "Copy link" {
		t.Fatalf("en copy = %q", t2)
	}
	setLang("de-DE") // unknown → English fallback
	if t2 := t2Copy(); t2 != "Copy link" {
		t.Fatalf("fallback copy = %q", t2)
	}
	// every English key must have a French twin (and vice versa)
	for k := range enStrings {
		if _, ok := frStrings[k]; !ok {
			t.Errorf("missing fr translation for %q", k)
		}
	}
	for k := range frStrings {
		if _, ok := enStrings[k]; !ok {
			t.Errorf("stray fr key %q", k)
		}
	}
}

func t2Copy() string { return t("copy") }
