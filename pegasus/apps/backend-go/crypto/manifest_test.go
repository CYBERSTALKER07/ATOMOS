package crypto

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestGenerateSHA256_KnownVector(t *testing.T) {
	// SHA-256 of "hello" is well-known
	expected := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	got := GenerateSHA256("hello")
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}

func TestGenerateSHA256_EmptyString(t *testing.T) {
	expected := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	got := GenerateSHA256("")
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}

func TestGenerateSHA256_Deterministic(t *testing.T) {
	a := GenerateSHA256("delivery-token-xyz")
	b := GenerateSHA256("delivery-token-xyz")
	if a != b {
		t.Error("same input should produce same hash")
	}
}

func TestGenerateSHA256_DifferentInputs(t *testing.T) {
	a := GenerateSHA256("token-a")
	b := GenerateSHA256("token-b")
	if a == b {
		t.Error("different inputs should produce different hashes")
	}
}

func TestGenerateSHA256_Format(t *testing.T) {
	got := GenerateSHA256("test")
	// Should be 64 hex chars (256 bits)
	if len(got) != 64 {
		t.Errorf("length = %d, want 64", len(got))
	}
	// Must be valid hex
	_, err := hex.DecodeString(got)
	if err != nil {
		t.Errorf("not valid hex: %v", err)
	}
}

func TestGenerateSHA256_MatchesStdlib(t *testing.T) {
	input := "verify-against-stdlib"
	sum := sha256.Sum256([]byte(input))
	expected := hex.EncodeToString(sum[:])
	got := GenerateSHA256(input)
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}

func TestRouteManifest_Struct(t *testing.T) {
	m := RouteManifest{
		DriverID:  "drv-1",
		Date:      "2026-04-12",
		ExpiresAt: 1776076800,
		Hashes:    map[string]string{"ORD-1": "abc123"},
	}
	if m.DriverID != "drv-1" || len(m.Hashes) != 1 {
		t.Errorf("unexpected: %+v", m)
	}
}
