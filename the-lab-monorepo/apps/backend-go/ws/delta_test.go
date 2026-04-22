package ws

import (
	"testing"
)

func TestNewDelta(t *testing.T) {
	d := NewDelta(DeltaOrderUpdate, "ORD-1", map[string]interface{}{
		"status":    "DISPATCHED",
		"driver_id": "D-9",
	})
	if d.T != DeltaOrderUpdate {
		t.Errorf("T = %q, want %q", d.T, DeltaOrderUpdate)
	}
	if d.I != "ORD-1" {
		t.Errorf("I = %q, want ORD-1", d.I)
	}
	if d.TS <= 0 {
		t.Error("TS should be > 0")
	}
	if d.D["status"] != "DISPATCHED" {
		t.Errorf("D[status] = %v, want DISPATCHED", d.D["status"])
	}
}

func TestCompressDelta(t *testing.T) {
	input := map[string]interface{}{
		"status":           "LOADED",
		"driver_id":        "D-1",
		"order_id":         "ORD-99",
		"warehouse_id":     "WH-5",
		"volumetric_units": 42.5,
		"unknown_field":    "keep-me",
	}

	compressed := CompressDelta(input)

	// V.O.I.D. dictionary: known keys should be shortened
	cases := []struct {
		shortKey string
		wantVal  interface{}
	}{
		{"s", "LOADED"},
		{"d", "D-1"},
		{"o", "ORD-99"},
		{"w", "WH-5"},
		{"v", 42.5},
		{"unknown_field", "keep-me"}, // Unknown keys pass through
	}

	for _, tc := range cases {
		got, ok := compressed[tc.shortKey]
		if !ok {
			t.Errorf("missing short key %q", tc.shortKey)
			continue
		}
		if got != tc.wantVal {
			t.Errorf("compressed[%q] = %v, want %v", tc.shortKey, got, tc.wantVal)
		}
	}

	// Verify long keys are NOT present
	for _, longKey := range []string{"status", "driver_id", "order_id", "warehouse_id", "volumetric_units"} {
		if _, ok := compressed[longKey]; ok {
			t.Errorf("long key %q should not be in compressed output", longKey)
		}
	}
}

func TestCompressDelta_Empty(t *testing.T) {
	compressed := CompressDelta(map[string]interface{}{})
	if len(compressed) != 0 {
		t.Errorf("expected empty map, got %d entries", len(compressed))
	}
}

func TestCompressDelta_LocationArray(t *testing.T) {
	// The V.O.I.D. dictionary maps "location" → "l" for [lat, lng] arrays
	input := map[string]interface{}{
		"location": []float64{41.311, 69.279},
	}
	compressed := CompressDelta(input)
	loc, ok := compressed["l"]
	if !ok {
		t.Fatal("missing short key 'l' for location")
	}
	arr, ok := loc.([]float64)
	if !ok {
		t.Fatal("location value should be []float64")
	}
	if len(arr) != 2 || arr[0] != 41.311 || arr[1] != 69.279 {
		t.Errorf("location = %v, want [41.311, 69.279]", arr)
	}
}

func TestDeltaEventTypes(t *testing.T) {
	// Verify all delta types are short (max 7 chars for wire efficiency)
	types := []string{
		DeltaOrderUpdate, DeltaDriverUpdate, DeltaFleetGPS,
		DeltaWarehouseLoad, DeltaPaymentUpdate, DeltaRouteUpdate,
		DeltaNegotiation, DeltaCreditUpdate,
	}
	for _, dt := range types {
		if len(dt) > 7 {
			t.Errorf("DeltaType %q is %d chars, should be ≤7 for wire efficiency", dt, len(dt))
		}
	}
}

func TestShortKeyMap_VOIDDictionary(t *testing.T) {
	// Verify the 8 canonical V.O.I.D. dictionary entries exist
	expected := map[string]string{
		"status":           "s",
		"location":         "l",
		"volumetric_units": "v",
		"id":               "i",
		"order_id":         "o",
		"driver_id":        "d",
		"warehouse_id":     "w",
		"updated_at":       "at",
	}
	for longKey, wantShort := range expected {
		got, ok := ShortKeyMap[longKey]
		if !ok {
			t.Errorf("V.O.I.D. key %q missing from ShortKeyMap", longKey)
			continue
		}
		if got != wantShort {
			t.Errorf("ShortKeyMap[%q] = %q, want %q", longKey, got, wantShort)
		}
	}

	// Verify no duplicate short keys (collision check, excluding intentional aliases)
	seen := make(map[string]string) // shortKey → longKey
	for longKey, shortKey := range ShortKeyMap {
		if existing, ok := seen[shortKey]; ok {
			// "state" and "status" both map to "s" — intentional alias
			// "volume_units" and "volumetric_units" both map to "v" — intentional alias
			isAlias := (longKey == "state" || existing == "state") ||
				(longKey == "volume_units" || existing == "volume_units")
			if !isAlias {
				t.Errorf("short key collision: %q maps to both %q and %q", shortKey, existing, longKey)
			}
		}
		seen[shortKey] = longKey
	}
}
