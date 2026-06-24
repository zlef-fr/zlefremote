package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
)

// E2EE matching the browser client (public/app/js/crypto.js):
// AES-256-GCM, 12-byte IV, frame = base64url(iv) + "." + base64url(ciphertext).
// The 32-byte key is generated here and only ever leaves the process inside the
// QR code's URL fragment — never sent to the relay.

var b64 = base64.RawURLEncoding

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
	return b64.EncodeToString(iv) + "." + b64.EncodeToString(ct), nil
}

func (s *Sealer) Open(frame string) ([]byte, error) {
	dot := strings.IndexByte(frame, '.')
	if dot < 0 {
		return nil, errors.New("bad frame")
	}
	iv, err := b64.DecodeString(frame[:dot])
	if err != nil {
		return nil, err
	}
	ct, err := b64.DecodeString(frame[dot+1:])
	if err != nil {
		return nil, err
	}
	return s.aead.Open(nil, iv, ct, nil)
}

// NewKey returns a fresh random 32-byte key and its base64url encoding (for the
// QR fragment).
func NewKey() ([]byte, string) {
	k := make([]byte, 32)
	rand.Read(k)
	return k, b64.EncodeToString(k)
}
