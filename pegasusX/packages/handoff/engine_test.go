package handoff

import "testing"

func TestEngineMintOnLoaded(t *testing.T) {
	engine := New(Config{LegacyOrderIDFallback: true, Mint: func() string { return "tok-abc" }})
	token := ""
	engine.ApplyTransition(&token, "PENDING", "LOADED", "", "")
	if token != "tok-abc" {
		t.Fatalf("token=%q want tok-abc", token)
	}
}

func TestEngineLegacyPublicToken(t *testing.T) {
	engine := New(Config{LegacyOrderIDFallback: true})
	got := engine.PublicToken("ord-1", "", "IN_TRANSIT")
	if got != "ord-1" {
		t.Fatalf("public=%q want ord-1", got)
	}
}

func TestEnginePersistedTokenPreferred(t *testing.T) {
	engine := New(Config{LegacyOrderIDFallback: true})
	got := engine.PublicToken("ord-1", "secret", "IN_TRANSIT")
	if got != "secret" {
		t.Fatalf("public=%q want secret", got)
	}
}

func TestEngineValidateRejectsWrongToken(t *testing.T) {
	engine := New(Config{LegacyOrderIDFallback: false})
	if err := engine.Validate("ord-1", "secret", "wrong"); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestEngineClearOnCompleted(t *testing.T) {
	engine := New(Config{LegacyOrderIDFallback: true, Mint: func() string { return "tok-new" }})
	token := "tok-old"
	engine.ApplyTransition(&token, "ARRIVED", "COMPLETED", "d1", "d1")
	if token != "" {
		t.Fatalf("token=%q want cleared", token)
	}
}

func TestEngineRotateOnReassign(t *testing.T) {
	engine := New(Config{LegacyOrderIDFallback: true, Mint: func() string { return "tok-new" }})
	token := "tok-old"
	engine.ApplyTransition(&token, "IN_TRANSIT", "IN_TRANSIT", "d1", "d2")
	if token != "tok-new" {
		t.Fatalf("token=%q want tok-new", token)
	}
}

func TestEngine_EnforceCryptographicTokenWhenFallbackDisabled(t *testing.T) {
	t.Parallel()

	// 1. With LegacyOrderIDFallback = false, PublicToken MUST NOT return order_id when token is empty
	engineStrict := New(Config{LegacyOrderIDFallback: false})
	if pub := engineStrict.PublicToken("ord-1234", "", "IN_TRANSIT"); pub != "" {
		t.Fatalf("expected empty public token when unminted in strict mode, got %q", pub)
	}

	// 2. Validate MUST reject order_id matching presentation when persisted token is empty
	if err := engineStrict.Validate("ord-1234", "", "ord-1234"); err == nil {
		t.Fatal("expected strict mode to reject presentation of order_id when token is unset")
	}

	// 3. Validate MUST accept exact cryptographic token
	if err := engineStrict.Validate("ord-1234", "crypto-token-xyz", "crypto-token-xyz"); err != nil {
		t.Fatalf("unexpected validation error with valid token: %v", err)
	}
}

