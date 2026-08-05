package partner

import (
	"testing"
	"time"
)

func TestGenerateAndVerifyAPIKey(t *testing.T) {
	plain, prefix, hash, err := GenerateAPIKey()
	if err != nil {
		t.Fatal(err)
	}
	if prefix == "" || !VerifyAPIKey(plain, hash) {
		t.Fatalf("verify failed prefix=%q", prefix)
	}
	got, ok := ParseBearerKey(plain)
	if !ok || got != prefix {
		t.Fatalf("parse got %q ok=%v", got, ok)
	}
	if VerifyAPIKey(plain+"x", hash) {
		t.Fatal("expected mismatch")
	}
}

func TestSignAndVerifyPayload(t *testing.T) {
	secret := "whsec_test"
	body := []byte(`{"type":"ORDER_CREATED"}`)
	ts := time.Now().Unix()
	sig := SignPayload(secret, ts, body)
	if !VerifySignature(secret, ts, body, sig) {
		t.Fatal("verify failed")
	}
	if VerifySignature(secret, ts+1, body, sig) {
		t.Fatal("expected fail on timestamp change")
	}
}

func TestHasScope(t *testing.T) {
	if !HasScope([]string{ScopeOrdersRead, ScopeCatalogRead}, ScopeOrdersRead) {
		t.Fatal("expected scope")
	}
	if HasScope([]string{ScopeOrdersRead}, ScopeOrdersWrite) {
		t.Fatal("unexpected scope")
	}
	if !HasScope([]string{"*"}, ScopeWebhooksManage) {
		t.Fatal("wildcard")
	}
}
