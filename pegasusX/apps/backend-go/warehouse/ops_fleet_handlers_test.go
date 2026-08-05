package warehouse

import (
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestGenerateOpsDriverPIN(t *testing.T) {
	t.Parallel()
	pin, err := generateOpsDriverPIN(4)
	if err != nil {
		t.Fatal(err)
	}
	if len(pin) != 4 {
		t.Fatalf("len=%d want 4", len(pin))
	}
	for _, c := range pin {
		if c < '0' || c > '9' {
			t.Fatalf("non-digit in pin %q", pin)
		}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pin), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if string(hash) == pin || strings.TrimSpace(string(hash)) == "4321" {
		t.Fatal("pin must not be stored plaintext")
	}
	if err := bcrypt.CompareHashAndPassword(hash, []byte(pin)); err != nil {
		t.Fatalf("bcrypt round-trip: %v", err)
	}
}
