package order

import (
	"errors"
	"testing"
	"time"
)

func TestEvaluateSettlementProximity_geofence100(t *testing.T) {
	// ~50 m north of origin-ish Tashkent coords
	orderLat, orderLng := 41.3111, 69.2797
	// ~0.00045 deg lat ≈ 50 m
	method, dist, ok := EvaluateSettlementProximity(41.31155, 69.2797, orderLat, orderLng, "")
	if !ok {
		t.Fatalf("expected unlock, dist=%.1f method=%s", dist, method)
	}
	if method != ProximityMethodGeofence100 {
		t.Fatalf("method=%s want GEOFENCE_100M", method)
	}
	if dist > SettlementProximityRadiusM {
		t.Fatalf("dist=%.1f > 100", dist)
	}
}

func TestEvaluateSettlementProximity_tooFar(t *testing.T) {
	orderLat, orderLng := 41.3111, 69.2797
	// ~1 km away
	method, dist, ok := EvaluateSettlementProximity(41.3200, 69.2797, orderLat, orderLng, "")
	if ok {
		t.Fatalf("expected lock, method=%s dist=%.1f", method, dist)
	}
	if dist < SettlementProximityRadiusM {
		t.Fatalf("dist=%.1f unexpectedly inside fence", dist)
	}
	_ = method
}

func TestEvaluateSettlementProximity_missingCoords(t *testing.T) {
	_, _, ok := EvaluateSettlementProximity(0, 0, 41.3, 69.2, "")
	if ok {
		t.Fatal("zero driver coords must fail")
	}
}

func TestValidateTelemetryFreshness(t *testing.T) {
	now := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	if err := ValidateTelemetryFreshness("", now, SettlementTelemetryMaxAge); err != nil {
		t.Fatalf("empty ts ok: %v", err)
	}
	fresh := now.Add(-30 * time.Second).Format(time.RFC3339Nano)
	if err := ValidateTelemetryFreshness(fresh, now, SettlementTelemetryMaxAge); err != nil {
		t.Fatalf("fresh: %v", err)
	}
	stale := now.Add(-10 * time.Minute).Format(time.RFC3339Nano)
	if err := ValidateTelemetryFreshness(stale, now, SettlementTelemetryMaxAge); err == nil {
		t.Fatal("stale must fail")
	} else if !errors.Is(err, ErrProximityTelemetryStale) {
		t.Fatalf("want ErrProximityTelemetryStale, got %v", err)
	}
}
