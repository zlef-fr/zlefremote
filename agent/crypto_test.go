package main

import "testing"

// The same pinned frame as desktop/crypto_test.go and the browser client's
// envelope: AES-256-GCM, 12-byte IV, base64url(iv) + "." + base64url(ct).
// If either side's crypto is ever "cleaned up", this fails before a release
// does.
const (
	vecKeyB64 = "AAcOFRwjKjE4P0ZNVFtiaXB3foWMk5qhqK-2vcTL0tk"
	vecFrame  = "emxlZnJlbW90ZTEy.FkjNAtUp5DuUEAPlMoo2OsJ68FXoMSn8YkBcgbZGrFx-3YwoVJa_uDHnCCk0Sr2naeLL"
	vecPlain  = `{"t":"key","k":"a","mods":["ctrl"]}`
)

func TestAgentOpensPinnedFrame(t *testing.T) {
	key, err := b64.DecodeString(vecKeyB64)
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewSealer(key)
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
