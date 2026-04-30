package proximity

import (
	"math"
	"testing"
)

// ─── parseWKT ───────────────────────────────────────────────────────────────

func TestParseWKT_Valid(t *testing.T) {
	// POINT(lng lat) → returns (lat, lng)
	lat, lng, err := parseWKT("POINT(69.27 41.31)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(lat-41.31) > 0.001 {
		t.Errorf("lat = %f, want 41.31", lat)
	}
	if math.Abs(lng-69.27) > 0.001 {
		t.Errorf("lng = %f, want 69.27", lng)
	}
}

func TestParseWKT_NegativeCoordinates(t *testing.T) {
	lat, lng, err := parseWKT("POINT(-73.935242 40.730610)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(lat-40.730610) > 0.0001 {
		t.Errorf("lat = %f, want 40.730610", lat)
	}
	if math.Abs(lng-(-73.935242)) > 0.0001 {
		t.Errorf("lng = %f, want -73.935242", lng)
	}
}

func TestParseWKT_Invalid_Empty(t *testing.T) {
	_, _, err := parseWKT("")
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestParseWKT_Invalid_NoPrefix(t *testing.T) {
	_, _, err := parseWKT("69.27 41.31")
	if err == nil {
		t.Error("expected error for missing POINT prefix")
	}
}

func TestParseWKT_Invalid_OneCoord(t *testing.T) {
	_, _, err := parseWKT("POINT(69.27)")
	if err == nil {
		t.Error("expected error for single coordinate")
	}
}

func TestParseWKT_Invalid_ThreeCoords(t *testing.T) {
	_, _, err := parseWKT("POINT(69.27 41.31 100)")
	if err == nil {
		t.Error("expected error for three coordinates")
	}
}

func TestParseWKT_Invalid_NonNumeric(t *testing.T) {
	_, _, err := parseWKT("POINT(abc def)")
	if err == nil {
		t.Error("expected error for non-numeric coordinates")
	}
}

func TestParseWKT_Precision(t *testing.T) {
	lat, lng, err := parseWKT("POINT(69.123456789 41.987654321)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(lat-41.987654321) > 1e-9 {
		t.Errorf("precision lost: lat = %f", lat)
	}
	if math.Abs(lng-69.123456789) > 1e-9 {
		t.Errorf("precision lost: lng = %f", lng)
	}
}

// ─── Key Helpers ────────────────────────────────────────────────────────────

func TestDriverKey(t *testing.T) {
	if got := driverKey("drv-123"); got != "d:drv-123" {
		t.Errorf("got %q, want %q", got, "d:drv-123")
	}
}

func TestRetailerKey(t *testing.T) {
	if got := retailerKey("ret-456"); got != "r:ret-456" {
		t.Errorf("got %q, want %q", got, "r:ret-456")
	}
}

func TestDriverKey_Empty(t *testing.T) {
	if got := driverKey(""); got != "d:" {
		t.Errorf("got %q, want %q", got, "d:")
	}
}

// ─── Constants ──────────────────────────────────────────────────────────────

func TestBreachRadius_Is100Meters(t *testing.T) {
	if DefaultBreachRadius != 100.0 {
		t.Errorf("DefaultBreachRadius = %f, want 100.0", DefaultBreachRadius)
	}
}

// ─── Struct Fields ──────────────────────────────────────────────────────────

func TestDriverApproachingEvent_Fields(t *testing.T) {
	e := DriverApproachingEvent{
		OrderID:         "ord-1",
		SupplierID:      "sup-1",
		SupplierName:    "Nestle",
		RetailerID:      "ret-1",
		DeliveryToken:   "tok-abc",
		DriverLatitude:  41.31,
		DriverLongitude: 69.27,
	}
	if e.OrderID != "ord-1" || e.RetailerID != "ret-1" || e.DeliveryToken != "tok-abc" {
		t.Errorf("unexpected: %+v", e)
	}
}

func TestTransitOrder_Fields(t *testing.T) {
	o := transitOrder{
		OrderID:       "ord-1",
		RetailerID:    "ret-1",
		SupplierID:    "sup-1",
		SupplierName:  "Test Supplier",
		DeliveryToken: "tok-123",
		ShopLat:       41.31,
		ShopLng:       69.27,
	}
	if o.OrderID != "ord-1" || o.ShopLat != 41.31 {
		t.Errorf("unexpected: %+v", o)
	}
}
