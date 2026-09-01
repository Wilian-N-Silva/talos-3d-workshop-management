package storage

import (
	"crypto/sha256"
	"errors"
	"strings"
	"testing"
)

func TestParseObjectKeyAcceptsOpaqueTokens(t *testing.T) {
	values := []string{
		"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		"01K4Z4V2Y8N9B7H6M5Q3R2T1S0",
		"opaque-key_with-safe-characters",
	}

	for _, value := range values {
		t.Run(value, func(t *testing.T) {
			key, err := ParseObjectKey(value)
			if err != nil {
				t.Fatalf("ParseObjectKey() error = %v", err)
			}
			if !key.Valid() {
				t.Fatal("parsed key is invalid")
			}
			if got := key.String(); got != value {
				t.Fatalf("String() = %q, want %q", got, value)
			}
		})
	}
}

func TestParseObjectKeyRejectsPathAndEscapingSyntax(t *testing.T) {
	values := []string{
		"",
		".",
		"..",
		"../object",
		`..\object`,
		"objects/key",
		`C:\objects\key`,
		"%2e%2e%2fobject",
		"object name",
		"objeto-ç",
		strings.Repeat("a", maxObjectKeyLength+1),
	}

	for _, value := range values {
		t.Run(value, func(t *testing.T) {
			key, err := ParseObjectKey(value)
			if !errors.Is(err, ErrInvalidObjectKey) {
				t.Fatalf("ParseObjectKey() error = %v, want ErrInvalidObjectKey", err)
			}
			if key.Valid() {
				t.Fatal("rejected key is valid")
			}
		})
	}
}

func TestSHA256DigestString(t *testing.T) {
	digest := SHA256Digest(sha256.Sum256([]byte("talos")))
	const expected = "216016a8050f7df7d7302341465e5746c918aeea1e49c8cab4708e5a2159f383"

	if got := digest.String(); got != expected {
		t.Fatalf("String() = %q, want %q", got, expected)
	}
}
