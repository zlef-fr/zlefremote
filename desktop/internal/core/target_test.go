package core

import "testing"

const testKey = "AAcOFRwjKjE4P0ZNVFtiaXB3foWMk5qhqK-2vcTL0tk"

func TestParseRelayLink(t *testing.T) {
	tg, err := ParseTarget("https://remote.zlef.fr/r/AB12CD#k=" + testKey + "&p=1")
	if err != nil {
		t.Fatal(err)
	}
	if !tg.Relay || tg.Room != "AB12CD" {
		t.Fatalf("relay/room: %+v", tg)
	}
	if tg.WSURL != "wss://remote.zlef.fr/ws" {
		t.Fatalf("ws url: %q", tg.WSURL)
	}
	if !tg.Persistent {
		t.Fatal("p=1 should mark the device as saveable")
	}
}

func TestParseLanLink(t *testing.T) {
	tg, err := ParseTarget("  http://192.168.1.24:9783/#k=" + testKey + "  ")
	if err != nil {
		t.Fatal(err)
	}
	if tg.Relay || tg.Room != "" {
		t.Fatalf("LAN link parsed as relay: %+v", tg)
	}
	if tg.WSURL != "ws://192.168.1.24:9783/ws" {
		t.Fatalf("ws url: %q", tg.WSURL)
	}
	if tg.Describe() != "192.168.1.24:9783" {
		t.Fatalf("describe: %q", tg.Describe())
	}
}

func TestParseLowercaseRoomIsUppercased(t *testing.T) {
	tg, _ := ParseTarget("https://remote.zlef.fr/r/ab12cd#k=" + testKey)
	if tg.Room != "AB12CD" {
		t.Fatalf("room: %q", tg.Room)
	}
}

func TestParseRejectsLinkWithoutKey(t *testing.T) {
	if _, err := ParseTarget("https://remote.zlef.fr/r/AB12CD"); err == nil {
		t.Fatal("a keyless link was accepted — it could never decrypt anything")
	}
}

func TestParseRejectsShortKey(t *testing.T) {
	if _, err := ParseTarget("https://remote.zlef.fr/r/AB12CD#k=abcd"); err == nil {
		t.Fatal("a truncated key was accepted")
	}
}

func TestSameIgnoresLabel(t *testing.T) {
	a, _ := ParseTarget("https://remote.zlef.fr/r/AB12CD#k=" + testKey)
	b, _ := ParseTarget("https://remote.zlef.fr/r/AB12CD#k=" + testKey)
	b.Label = "renamed"
	if !a.Same(b) {
		t.Fatal("the same machine should match regardless of its label")
	}
}
