package core

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
)

// The exact same envelope as the agent (agent/crypto.go) and the phone client
// (public/app/js/crypto.js): AES-256-GCM, 12-byte IV, frame =
// base64url(iv) + "." + base64url(ciphertext). The 32-byte key never touches
// the relay — it arrives here inside the pairing URL's fragment, which the
// browser/agent never sends over the wire.
//
// crypto_test.go pins this against a frame sealed by the agent, so the three
// implementations can't drift apart silently.

var B64 = base64.RawURLEncoding

type Sealer struct{ aead cipher.AEAD }

func NewSealer(key []byte) (*Sealer, error) {
	if len(key) != 32 {
		return nil, errors.New("key must be 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Sealer{aead}, nil
}

func (s *Sealer) Seal(plaintext []byte) (string, error) {
	iv := make([]byte, 12)
	if _, err := rand.Read(iv); err != nil {
		return "", err
	}
	ct := s.aead.Seal(nil, iv, plaintext, nil)
	return B64.EncodeToString(iv) + "." + B64.EncodeToString(ct), nil
}

func (s *Sealer) Open(frame string) ([]byte, error) {
	dot := strings.IndexByte(frame, '.')
	if dot < 0 {
		return nil, errors.New("bad frame")
	}
	iv, err := B64.DecodeString(frame[:dot])
	if err != nil {
		return nil, err
	}
	ct, err := B64.DecodeString(frame[dot+1:])
	if err != nil {
		return nil, err
	}
	return s.aead.Open(nil, iv, ct, nil)
}

// SealerFromKeyB64 builds a sealer from the base64url key carried in a pairing
// URL fragment.
func SealerFromKeyB64(k string) (*Sealer, error) {
	raw, err := B64.DecodeString(k)
	if err != nil {
		return nil, errors.New("the pairing link's key is not valid base64url")
	}
	return NewSealer(raw)
}
