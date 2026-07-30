package core

import "testing"

// A frame sealed once and pinned here. The agent (agent/crypto_test.go), this
// client and the browser client all speak the same envelope; pinning the bytes
// in both Go modules means a change to either implementation fails a test
// instead of silently breaking pairing in the field.
const (
	vecKeyB64 = "AAcOFRwjKjE4P0ZNVFtiaXB3foWMk5qhqK-2vcTL0tk"
	vecFrame  = "emxlZnJlbW90ZTEy.FkjNAtUp5DuUEAPlMoo2OsJ68FXoMSn8YkBcgbZGrFx-3YwoVJa_uDHnCCk0Sr2naeLL"
	vecPlain  = `{"t":"key","k":"a","mods":["ctrl"]}`
)

func TestOpensPinnedFrame(t *testing.T) {
	s, err := SealerFromKeyB64(vecKeyB64)
	if err != nil {
		t.Fatal(err)
	}
	pt, err := s.Open(vecFrame)
	if err != nil {
		t.Fatalf("pinned frame did not open: %v", err)
	}
	if string(pt) != vecPlain {
		t.Fatalf("plaintext: got %q, want %q", pt, vecPlain)
	}
}

func TestSealRoundTrip(t *testing.T) {
	s, _ := SealerFromKeyB64(vecKeyB64)
	frame, err := s.Seal([]byte("héllo · wörld"))
	if err != nil {
		t.Fatal(err)
	}
	pt, err := s.Open(frame)
	if err != nil {
		t.Fatal(err)
	}
	if string(pt) != "héllo · wörld" {
		t.Fatalf("round trip: got %q", pt)
	}
}

func TestWrongKeyIsRejected(t *testing.T) {
	// the key is the only authentication gate: a frame sealed with another key
	// must not open, or anyone who found the room code could inject input.
	other, _ := SealerFromKeyB64("AQcOFRwjKjE4P0ZNVFtiaXB3foWMk5qhqK-2vcTL0tk")
	if _, err := other.Open(vecFrame); err == nil {
		t.Fatal("a frame opened with the wrong key")
	}
}

func TestBadKeyLength(t *testing.T) {
	if _, err := SealerFromKeyB64("c2hvcnQ"); err == nil {
		t.Fatal("a 5-byte key was accepted")
	}
}
