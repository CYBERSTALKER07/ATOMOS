// Package spannerrouter provides geographic Spanner routing based on H3 cells.
package spannerrouter

import (
	"testing"

	h3 "github.com/uber/h3-go/v4"
)

// cellAt converts a lat/lng to a res-7 H3 cell string. Helper for tests only.
func cellAt(lat, lng float64) string {
	c, err := h3.LatLngToCell(h3.LatLng{Lat: lat, Lng: lng}, 7)
	if err != nil {
		return ""
	}
	return h3.CellToString(c)
}

// TestCellToRegion verifies that major V.O.I.D. deployment cities resolve
// to their expected region, and that unknown/empty inputs fall back correctly.
func TestCellToRegion(t *testing.T) {
	cases := []struct {
		name string
		lat  float64
		lng  float64
		want string
	}{
		{"Tashkent", 41.2, 69.2, "asia"},
		{"Samarkand", 39.7, 66.9, "asia"},
		{"Almaty", 43.3, 76.9, "asia"},
		{"Bishkek", 42.9, 74.6, "asia"},
		{"Dushanbe", 38.5, 68.8, "asia"},
		{"Ashgabat", 37.9, 58.4, "asia"},

		{"Paris", 48.8, 2.3, "eu"},
		{"Berlin", 52.5, 13.4, "eu"},
		{"Warsaw", 52.2, 21.0, "eu"},

		{"New York", 40.7, -74.0, "us"},
		{"São Paulo", -23.5, -46.6, "us"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cell := cellAt(tc.lat, tc.lng)
			got := cellToRegion(cell)
			if got != tc.want {
				t.Errorf("cellToRegion(%s @ %.1f,%.1f) = %q, want %q",
					cell, tc.lat, tc.lng, got, tc.want)
			}
		})
	}
}

// TestCellToRegion_Fallback verifies graceful fallback for edge cases.
func TestCellToRegion_Fallback(t *testing.T) {
	cases := []struct {
		name string
		cell string
	}{
		{"empty", ""},
		{"too short", "87abc"},
		{"invalid hex", "xxxxxxxyyyyyyy"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := cellToRegion(tc.cell)
			if got != "" {
				t.Errorf("cellToRegion(%q) = %q, want empty string (primary fallback)", tc.cell, got)
			}
		})
	}
}

// TestRouterFor_SingleRegion verifies that single-region mode always returns
// the same client regardless of cell input.
func TestRouterFor_SingleRegion(t *testing.T) {
	r := NewSingleRegion(nil) // nil client is fine for routing-logic tests

	for _, cell := range []string{
		cellAt(41.2, 69.2), // Tashkent
		cellAt(48.8, 2.3),  // Paris
		"",                 // empty
	} {
		if got := r.For(cell); got != nil {
			t.Errorf("single-region For(%q): expected nil client, got %v", cell, got)
		}
	}
	if got := r.Primary(); got != nil {
		t.Errorf("single-region Primary(): expected nil client, got %v", got)
	}
}
