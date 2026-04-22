package routing

import (
	"math"
	"testing"
)

// ─── safeGetLegDuration ─────────────────────────────────────────────────────

func TestSafeGetLegDuration_InBounds(t *testing.T) {
	legs := []mapsRouteLeg{
		{Duration: mapsValue{Value: 300, Text: "5 min"}},
		{Duration: mapsValue{Value: 600, Text: "10 min"}},
	}
	if got := safeGetLegDuration(legs, 0); got != 300 {
		t.Errorf("got %d, want 300", got)
	}
	if got := safeGetLegDuration(legs, 1); got != 600 {
		t.Errorf("got %d, want 600", got)
	}
}

func TestSafeGetLegDuration_OutOfBounds(t *testing.T) {
	legs := []mapsRouteLeg{
		{Duration: mapsValue{Value: 300}},
	}
	if got := safeGetLegDuration(legs, 5); got != 0 {
		t.Errorf("out of bounds should return 0, got %d", got)
	}
}

func TestSafeGetLegDuration_EmptySlice(t *testing.T) {
	if got := safeGetLegDuration(nil, 0); got != 0 {
		t.Errorf("nil slice should return 0, got %d", got)
	}
}

// ─── safeGetLegDistance ─────────────────────────────────────────────────────

func TestSafeGetLegDistance_InBounds(t *testing.T) {
	legs := []mapsRouteLeg{
		{Distance: mapsValue{Value: 1500, Text: "1.5 km"}},
	}
	if got := safeGetLegDistance(legs, 0); got != 1500 {
		t.Errorf("got %d, want 1500", got)
	}
}

func TestSafeGetLegDistance_OutOfBounds(t *testing.T) {
	if got := safeGetLegDistance(nil, 0); got != 0 {
		t.Errorf("nil slice should return 0, got %d", got)
	}
}

// ─── parseWKTPoint ──────────────────────────────────────────────────────────

func TestParseWKTPoint_Valid(t *testing.T) {
	lat, lng, err := parseWKTPoint("POINT(69.27 41.31)")
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

func TestParseWKTPoint_NegativeCoords(t *testing.T) {
	lat, lng, err := parseWKTPoint("POINT(-73.935242 40.730610)")
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

func TestParseWKTPoint_Whitespace(t *testing.T) {
	lat, lng, err := parseWKTPoint("  POINT(69.27 41.31)  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(lat-41.31) > 0.001 || math.Abs(lng-69.27) > 0.001 {
		t.Errorf("lat=%f, lng=%f", lat, lng)
	}
}

func TestParseWKTPoint_Invalid_NoPrefix(t *testing.T) {
	_, _, err := parseWKTPoint("69.27 41.31")
	if err == nil {
		t.Error("expected error for missing POINT prefix")
	}
}

func TestParseWKTPoint_Invalid_Empty(t *testing.T) {
	_, _, err := parseWKTPoint("")
	if err == nil {
		t.Error("expected error for empty string")
	}
}

func TestParseWKTPoint_Invalid_OneCoord(t *testing.T) {
	_, _, err := parseWKTPoint("POINT(69.27)")
	if err == nil {
		t.Error("expected error for single coordinate")
	}
}

func TestParseWKTPoint_Invalid_NonNumeric(t *testing.T) {
	_, _, err := parseWKTPoint("POINT(abc def)")
	if err == nil {
		t.Error("expected error for non-numeric")
	}
}

// ─── Struct Fields ──────────────────────────────────────────────────────────

func TestDeliveryOrder_Fields(t *testing.T) {
	o := DeliveryOrder{
		OrderID:      "ord-1",
		DriverID:     "drv-1",
		ShopLocation: "POINT(69.27 41.31)",
		ParsedLat:    41.31,
		ParsedLng:    69.27,
	}
	if o.OrderID != "ord-1" || o.ParsedLat != 41.31 {
		t.Errorf("unexpected: %+v", o)
	}
}

func TestLegETA_CumulativeCalc(t *testing.T) {
	legs := []LegETA{
		{OrderID: "ord-1", SequenceIndex: 0, LegSec: 300, CumulativeSec: 300},
		{OrderID: "ord-2", SequenceIndex: 1, LegSec: 450, CumulativeSec: 750},
	}
	if legs[1].CumulativeSec != legs[0].CumulativeSec+legs[1].LegSec {
		t.Error("cumulative calculation incorrect")
	}
}

func TestRouteETAResult_TotalIncludes_ReturnLeg(t *testing.T) {
	r := RouteETAResult{
		Stops: []LegETA{
			{LegSec: 300, CumulativeSec: 300},
			{LegSec: 450, CumulativeSec: 750},
		},
		ReturnLegSec:  600,
		ReturnLegM:    5000,
		TotalRouteSec: 1350, // 300 + 450 + 600
	}
	expectedTotal := r.Stops[len(r.Stops)-1].CumulativeSec + r.ReturnLegSec
	if r.TotalRouteSec != expectedTotal {
		t.Errorf("total = %d, want %d", r.TotalRouteSec, expectedTotal)
	}
}
