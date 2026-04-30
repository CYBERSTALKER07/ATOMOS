package auth

import (
	"os"
	"strings"
	"testing"
)

// TestMain seeds Init() with ephemeral, test-only credentials so the package
// tests can exercise sign/verify paths. These values never leave the test
// binary and are unrelated to the production signing key.
func TestMain(m *testing.M) {
	Init("test-only-signing-key-do-not-use-in-prod", "test-only-internal-key")
	os.Exit(m.Run())
}

// ─── Fail-closed contract tests ─────────────────────────────────────────────
// These prove that auth.Init panics on empty credentials regardless of
// environment. The panic fires before any global mutation, so running these
// leaves JWTSecret / internalAPIKey (seeded by TestMain) intact for every
// other test in the package.

func TestInit_PanicsOnEmptyJWTSecret(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("auth.Init(\"\", non-empty) must panic, did not")
		}
		msg, _ := r.(string)
		if !strings.Contains(msg, "JWT_SECRET") {
			t.Errorf("panic message = %q, want substring %q", msg, "JWT_SECRET")
		}
	}()
	Init("", "non-empty-internal-key")
}

func TestInit_PanicsOnEmptyInternalAPIKey(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("auth.Init(non-empty, \"\") must panic, did not")
		}
		msg, _ := r.(string)
		if !strings.Contains(msg, "INTERNAL_API_KEY") {
			t.Errorf("panic message = %q, want substring %q", msg, "INTERNAL_API_KEY")
		}
	}()
	Init("non-empty-jwt", "")
}

func TestInit_PanicsOnBothEmpty(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("auth.Init(\"\", \"\") must panic, did not")
		}
	}()
	Init("", "")
}
