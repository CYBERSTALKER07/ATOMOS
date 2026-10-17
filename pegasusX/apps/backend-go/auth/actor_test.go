package auth

import (
	"context"
	"testing"
)

func TestActorLabelPreference(t *testing.T) {
	if got := ActorLabel(Claims{Subject: "u1", PhoneNumber: "+1", Role: RolePlatformAdmin}); got != "u1" {
		t.Fatalf("subject first: %q", got)
	}
	if got := ActorLabel(Claims{PhoneNumber: "+998", Role: RolePlatformAdmin}); got != "+998" {
		t.Fatalf("phone fallback: %q", got)
	}
	if got := ActorLabel(Claims{RetailerUserID: "ru-1", Role: RoleRetailer}); got != "ru-1" {
		t.Fatalf("retailer user fallback: %q", got)
	}
	if got := ActorLabel(Claims{Role: RolePlatformAdmin}); got != string(RolePlatformAdmin) {
		t.Fatalf("role fallback: %q", got)
	}
	if got := ActorLabel(Claims{}); got != "unknown" {
		t.Fatalf("empty: %q", got)
	}
}

func TestActorFromContext(t *testing.T) {
	if got := ActorFromContext(context.Background()); got != "unknown" {
		t.Fatalf("no claims: %q", got)
	}
	ctx := WithClaims(context.Background(), Claims{Subject: "admin-a", Role: RolePlatformAdmin})
	if got := ActorFromContext(ctx); got != "admin-a" {
		t.Fatalf("with claims: %q", got)
	}
}
