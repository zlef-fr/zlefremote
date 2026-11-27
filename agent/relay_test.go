package main

import (
	"testing"
	"time"
)

func TestRoomKeeperKeepsTheGrantedRoom(t *testing.T) {
	k := &roomKeeper{}
	if k.want() != "" {
		t.Fatalf("first connect should let the relay pick, asked for %q", k.want())
	}
	k.granted("ABC234")
	if k.want() != "ABC234" {
		t.Fatalf("reconnect must ask for the held room, asked for %q", k.want())
	}
	if k.persistentRoom("ABC234") {
		t.Fatal("a one-time room must not be advertised as saveable")
	}
}

func TestRoomKeeperRetriesItsOwnStaleRoom(t *testing.T) {
	k := &roomKeeper{}
	k.granted("ABC234")
	t0 := time.Now()
	if !k.taken(t0) || !k.taken(t0.Add(reclaimWindow-time.Second)) {
		t.Fatal("a room we held must be retried while the relay reaps our old socket")
	}
	if k.want() != "ABC234" {
		t.Fatalf("still asking for the held room, got %q", k.want())
	}
	if k.taken(t0.Add(reclaimWindow + time.Second)) {
		t.Fatal("past the reclaim window the agent must move on")
	}
	if k.want() != "" {
		t.Fatalf("after giving up, the relay picks; asked for %q", k.want())
	}
}

func TestRoomKeeperDerivedRoom(t *testing.T) {
	k := &roomKeeper{derived: "SAVED7"}
	if k.want() != "SAVED7" {
		t.Fatalf("first connect asks for the derived room, got %q", k.want())
	}
	// never held: another live device owns it → give it up at once
	if k.taken(time.Now()) {
		t.Fatal("a derived room we never held is a collision, not ours to wait for")
	}
	if k.want() != "" || k.persistentRoom("SAVED7") {
		t.Fatal("after a collision the room is one-time and not saveable")
	}

	k = &roomKeeper{derived: "SAVED7"}
	k.granted("SAVED7")
	if !k.persistentRoom("SAVED7") {
		t.Fatal("the granted derived room is saveable")
	}
	if !k.taken(time.Now()) {
		t.Fatal("after a reconnect, our own derived room is retried, not abandoned")
	}
	if !k.persistentRoom("SAVED7") {
		t.Fatal("retrying must not drop the saveable flag")
	}
}
