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

func TestEvaluateSettlementProximity_H3Match(t *testing.T) {
	orderLat, orderLng := 41.3111, 69.2797
	settlementCell := SettlementH3Cell(orderLat, orderLng)
	if settlementCell == "" {
		t.Fatal("expected non-empty settlement cell")
	}
	// Driver at same coords matches Res 9 cell
	method, _, ok := EvaluateSettlementProximity(orderLat, orderLng, orderLat, orderLng, settlementCell)
	if !ok || method != ProximityMethodH3 {
		t.Fatalf("expected H3 method, got ok=%v method=%s", ok, method)
	}
}

func TestValidateProximityVelocity(t *testing.T) {
	t1 := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	lat1, lng1 := 41.3111, 69.2797

	// 1. Valid driving speed: ~500 m in 60s ≈ 8.3 m/s (30 km/h)
	// 0.0045 deg lat ≈ 500m
	t2Valid := t1.Add(60 * time.Second)
	lat2Valid, lng2Valid := 41.3156, 69.2797
	if err := ValidateProximityVelocity(lat1, lng1, t1, lat2Valid, lng2Valid, t2Valid); err != nil {
		t.Fatalf("expected plausible velocity to pass, got: %v", err)
	}

	// 2. Impossible jump (GPS spoofing): 5 km in 10s ≈ 500 m/s (1800 km/h)
	// 0.045 deg lat ≈ 5 km
	t2Spoof := t1.Add(10 * time.Second)
	lat2Spoof, lng2Spoof := 41.3561, 69.2797
	err := ValidateProximityVelocity(lat1, lng1, t1, lat2Spoof, lng2Spoof, t2Spoof)
	if err == nil {
		t.Fatal("expected impossible velocity to be rejected")
	}
	if !errors.Is(err, ErrProximityVelocitySpoofed) {
		t.Fatalf("expected ErrProximityVelocitySpoofed, got: %v", err)
	}

	// 3. Sub-50m jitter at short interval passes without false positive
	t2Jitter := t1.Add(2 * time.Second)
	lat2Jitter, lng2Jitter := 41.3112, 69.2797 // ~11m away
	if err := ValidateProximityVelocity(lat1, lng1, t1, lat2Jitter, lng2Jitter, t2Jitter); err != nil {
		t.Fatalf("expected natural GPS jitter to pass, got: %v", err)
	}
}

