package order

import (
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"testing"
	"time"

	"backend-go/cache"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// ─── HELPERS ──────────────────────────────────────────────────────────────────

// setupMiniRedis spins up an in-memory Redis and points cache.Client at it.
// cleanup restores cache.Client to nil.
func setupMiniRedis(t *testing.T) *miniredis.Miniredis {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run: %v", err)
	}
	cache.Client = redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		cache.Client.Close()
		cache.Client = nil
		mr.Close()
	})
	return mr
}

// ─── GenerateSecureToken ─────────────────────────────────────────────────────

func TestGenerateSecureToken_Length(t *testing.T) {
	tok := GenerateSecureToken()
	if len(tok) != 16 {
		t.Errorf("expected 16-char token, got %d chars: %q", len(tok), tok)
	}
}

func TestGenerateSecureToken_ValidHex(t *testing.T) {
	tok := GenerateSecureToken()
	if _, err := hex.DecodeString(tok); err != nil {
		t.Errorf("token %q is not valid hex: %v", tok, err)
	}
}

func TestGenerateSecureToken_Unique(t *testing.T) {
	seen := make(map[string]struct{}, 100)
	for i := 0; i < 100; i++ {
		tok := GenerateSecureToken()
		if _, dup := seen[tok]; dup {
			t.Fatalf("duplicate token on iteration %d: %s", i, tok)
		}
		seen[tok] = struct{}{}
	}
}

// ─── Redis TTL: RefreshDeliveryTokenTTL ──────────────────────────────────────

func TestRefreshDeliveryTokenTTL_SetsTTL(t *testing.T) {
	mr := setupMiniRedis(t)

	key := "delivery_token:ORD-001"
	mr.Set(key, "tok_abc")

	svc := &OrderService{} // nil Spanner is fine — Redis-only path
	svc.RefreshDeliveryTokenTTL(context.Background(), "ORD-001")

	ttl := mr.TTL(key)
	if ttl < 3*time.Hour || ttl > 5*time.Hour {
		t.Errorf("expected TTL ~4h, got %v", ttl)
	}
}

func TestRefreshDeliveryTokenTTL_NilClient(t *testing.T) {
	old := cache.Client
	cache.Client = nil
	defer func() { cache.Client = old }()

	svc := &OrderService{}
	// Must not panic
	svc.RefreshDeliveryTokenTTL(context.Background(), "ORD-999")
}

// ─── Redis TTL: InvalidateDeliveryToken ──────────────────────────────────────

func TestInvalidateDeliveryToken_DeletesKey(t *testing.T) {
	mr := setupMiniRedis(t)

	key := "delivery_token:ORD-002"
	mr.Set(key, "tok_xyz")

	svc := &OrderService{}
	svc.InvalidateDeliveryToken(context.Background(), "ORD-002")

	if mr.Exists(key) {
		t.Error("expected key to be deleted, but it still exists")
	}
}

func TestInvalidateDeliveryToken_NilClient(t *testing.T) {
	old := cache.Client
	cache.Client = nil
	defer func() { cache.Client = old }()

	svc := &OrderService{}
	// Must not panic
	svc.InvalidateDeliveryToken(context.Background(), "ORD-999")
}

func TestInvalidateDeliveryToken_NonExistentKey(t *testing.T) {
	setupMiniRedis(t)

	svc := &OrderService{}
	// Must not panic — DEL on missing key returns 0, no error
	svc.InvalidateDeliveryToken(context.Background(), "ORD-DOES-NOT-EXIST")
}

// ─── Token full lifecycle ────────────────────────────────────────────────────

func TestTokenLifecycle_SetRefreshInvalidate(t *testing.T) {
	mr := setupMiniRedis(t)

	key := "delivery_token:ORD-LC"
	mr.Set(key, "tok_lifecycle_123")
	// No TTL yet
	if mr.TTL(key) != 0 {
		t.Fatal("freshly SET key should have no TTL")
	}

	svc := &OrderService{}

	// Refresh → sets TTL to ~4h
	svc.RefreshDeliveryTokenTTL(context.Background(), "ORD-LC")
	ttl := mr.TTL(key)
	if ttl < 3*time.Hour || ttl > 5*time.Hour {
		t.Errorf("after Refresh, expected TTL ~4h, got %v", ttl)
	}

	// Value still intact
	val, err := mr.Get(key)
	if err != nil || val != "tok_lifecycle_123" {
		t.Errorf("value changed after Refresh: %q err=%v", val, err)
	}

	// Invalidate → key gone
	svc.InvalidateDeliveryToken(context.Background(), "ORD-LC")
	if mr.Exists(key) {
		t.Error("key should be gone after Invalidate")
	}
}

// ─── ValidateQRToken: Redis fast-path rejection ─────────────────────────────

func TestValidateQRToken_RedisFastPathReject(t *testing.T) {
	mr := setupMiniRedis(t)

	key := "delivery_token:ORD-QR"
	mr.Set(key, "correct_token_abc")

	// nil Spanner client → if Redis rejects, we never touch Spanner
	svc := &OrderService{Client: nil}
	_, err := svc.ValidateQRToken(context.Background(), "ORD-QR", "wrong_token_xyz")

	if err == nil {
		t.Fatal("expected error for wrong token, got nil")
	}
	expected := "INVALID QR TOKEN"
	if got := err.Error(); len(got) < len(expected) || got[:len(expected)] != expected {
		t.Errorf("expected error starting with %q, got %q", expected, got)
	}
}

// ─── Error types ─────────────────────────────────────────────────────────────

func TestErrStateConflict_Error(t *testing.T) {
	e := &ErrStateConflict{OrderID: "ORD-1", CurrentState: "COMPLETED", AttemptedOp: "cancel"}
	got := e.Error()
	if got != "state conflict on ORD-1: current state is COMPLETED, cannot cancel" {
		t.Errorf("unexpected: %s", got)
	}
}

func TestErrVersionConflict_Error(t *testing.T) {
	e := &ErrVersionConflict{OrderID: "ORD-2", ExpectedVersion: 3, ActualVersion: 5}
	got := e.Error()
	if got != "version conflict on ORD-2: expected v3, found v5 — refresh required" {
		t.Errorf("unexpected: %s", got)
	}
}

func TestErrFreezeLock_Error(t *testing.T) {
	ts := time.Date(2026, 4, 12, 14, 0, 0, 0, time.UTC)
	e := &ErrFreezeLock{OrderID: "ORD-3", LockedUntil: ts}
	got := e.Error()
	expected := fmt.Sprintf("order ORD-3 is locked for physical dispatch until %s", ts.Format(time.RFC3339))
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

// ─── Haversine (getDistance) ─────────────────────────────────────────────────

func TestGetDistance_SamePoint(t *testing.T) {
	d := getDistance(41.2995, 69.2401, 41.2995, 69.2401)
	if d > 1.0 {
		t.Errorf("same point should be ~0m, got %.2f", d)
	}
}

func TestGetDistance_KnownPair(t *testing.T) {
	// Tashkent ↔ Samarkand ≈ 262 km
	d := getDistance(41.2995, 69.2401, 39.6542, 66.9597)
	km := d / 1000.0
	if km < 250 || km > 280 {
		t.Errorf("Tashkent↔Samarkand ≈ 262km, got %.1f km", km)
	}
}

func TestGetDistance_Under100m(t *testing.T) {
	// ~0.0004° latitude ≈ 44 m
	d := getDistance(41.2995, 69.2401, 41.2999, 69.2401)
	if d > 100 {
		t.Errorf("expected <100m, got %.2f m", d)
	}
	if d < 30 {
		t.Errorf("expected >30m, got %.2f m", d)
	}
}

func TestGetDistance_Antipodal(t *testing.T) {
	// (0°,0°) ↔ (0°,180°) ≈ half Earth circumference ≈ 20015 km
	d := getDistance(0, 0, 0, 180)
	km := d / 1000.0
	halfCircum := math.Pi * 6371.0 // ≈ 20015 km
	if math.Abs(km-halfCircum) > 50 {
		t.Errorf("expected ~%.0f km, got %.0f km", halfCircum, km)
	}
}

// ─── parseWKTPoint ──────────────────────────────────────────────────────────

func TestParseWKTPoint_Valid(t *testing.T) {
	loc, err := parseWKTPoint("POINT(69.27 41.31)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(loc.Latitude-41.31) > 0.001 {
		t.Errorf("lat = %f, want 41.31", loc.Latitude)
	}
	if math.Abs(loc.Longitude-69.27) > 0.001 {
		t.Errorf("lng = %f, want 69.27", loc.Longitude)
	}
}

func TestParseWKTPoint_Negative(t *testing.T) {
	loc, err := parseWKTPoint("POINT(-73.93 40.73)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loc.Longitude >= 0 {
		t.Errorf("longitude should be negative, got %f", loc.Longitude)
	}
}

func TestParseWKTPoint_InvalidFormat(t *testing.T) {
	_, err := parseWKTPoint("POLYGON((0 0, 1 1, 1 0, 0 0))")
	if err == nil {
		t.Error("expected error for non-POINT WKT")
	}
}

func TestParseWKTPoint_Empty(t *testing.T) {
	_, err := parseWKTPoint("")
	if err == nil {
		t.Error("expected error for empty string")
	}
}

func TestParseWKTPoint_OneCoord(t *testing.T) {
	_, err := parseWKTPoint("POINT(69.27)")
	if err == nil {
		t.Error("expected error for single coordinate")
	}
}

// ─── normalizeCardGateway ───────────────────────────────────────────────────

func TestNormalizeCardGateway_Valid(t *testing.T) {
	cases := map[string]string{
		"CASH":      "CASH",
		"cash":      "CASH",
		"GLOBAL_PAY":      "GLOBAL_PAY",
		"global_pay":      "GLOBAL_PAY",
		"GLOBAL_PAY": "GLOBAL_PAY",
		"global_pay": "GLOBAL_PAY",
	}
	for input, want := range cases {
		t.Run(input, func(t *testing.T) {
			if got := normalizeCardGateway(input); got != want {
				t.Errorf("got %q, want %q", got, want)
			}
		})
	}
}

func TestNormalizeCardGateway_Invalid(t *testing.T) {
	invalids := []string{"VISA", "MASTERCARD", "UNKNOWN", "", "  "}
	for _, gw := range invalids {
		t.Run(gw, func(t *testing.T) {
			if got := normalizeCardGateway(gw); got != "" {
				t.Errorf("invalid gateway %q should return empty, got %q", gw, got)
			}
		})
	}
}

func TestNormalizeCardGateway_Whitespace(t *testing.T) {
	if got := normalizeCardGateway("  CASH  "); got != "CASH" {
		t.Errorf("got %q, want CASH (should trim)", got)
	}
}

// ─── GeneratePredictionId ───────────────────────────────────────────────────

func TestGeneratePredictionId_NotEmpty(t *testing.T) {
	id := GeneratePredictionId()
	if id == "" {
		t.Error("prediction ID should not be empty")
	}
}

func TestGeneratePredictionId_Unique(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := GeneratePredictionId()
		if ids[id] {
			t.Fatalf("duplicate prediction ID: %s", id)
		}
		ids[id] = true
	}
}
