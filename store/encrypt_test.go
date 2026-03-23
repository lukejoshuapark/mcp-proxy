package store

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	tests := []struct {
		name      string
		plaintext []byte
	}{
		{"simple text", []byte("hello world")},
		{"json payload", []byte(`{"access_token":"secret","refresh_token":"also-secret"}`)},
		{"empty", []byte("")},
		{"binary data", func() []byte { b := make([]byte, 256); rand.Read(b); return b }()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := Encrypt(tt.plaintext, key)
			if err != nil {
				t.Fatalf("Encrypt failed: %v", err)
			}

			decrypted, err := Decrypt(encrypted, key)
			if err != nil {
				t.Fatalf("Decrypt failed: %v", err)
			}

			if !bytes.Equal(decrypted, tt.plaintext) {
				t.Errorf("round-trip mismatch: got %q, want %q", decrypted, tt.plaintext)
			}
		})
	}
}

func TestEncryptProducesDifferentCiphertexts(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	plaintext := []byte("same input")
	enc1, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatal(err)
	}
	enc2, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatal(err)
	}

	if enc1 == enc2 {
		t.Error("two encryptions of the same plaintext should produce different ciphertexts (random nonce)")
	}
}

func TestDecryptWithWrongKeyFails(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	for i := range key1 {
		key1[i] = byte(i)
		key2[i] = byte(i + 1)
	}

	encrypted, err := Encrypt([]byte("secret data"), key1)
	if err != nil {
		t.Fatal(err)
	}

	_, err = Decrypt(encrypted, key2)
	if err == nil {
		t.Error("decryption with wrong key should fail")
	}
}

func TestDecryptInvalidCiphertext(t *testing.T) {
	key := make([]byte, 32)

	_, err := Decrypt("not-valid-base64!!!", key)
	if err == nil {
		t.Error("decryption of invalid base64 should fail")
	}

	_, err = Decrypt("dG9vc2hvcnQ=", key) // "tooshort"
	if err == nil {
		t.Error("decryption of too-short ciphertext should fail")
	}
}

func TestEncryptInvalidKeyLength(t *testing.T) {
	shortKey := make([]byte, 16)
	_, err := Encrypt([]byte("data"), shortKey)
	if err != nil {
		t.Error("AES-128 key should still work with NewCipher")
	}

	badKey := make([]byte, 7)
	_, err = Encrypt([]byte("data"), badKey)
	if err == nil {
		t.Error("invalid key length should fail")
	}
}
