package order

import "testing"

func TestWarehouseCoversRetailer_Hybrid(t *testing.T) {
	cells := CellsForCity(40.7128, -74.0060) // NYC
	if len(cells) == 0 {
		t.Fatal("city must produce H3 cells")
	}
	retailer := cells[0]
	if !WarehouseCoversRetailer("US", cells, "US", retailer) {
		t.Fatal("retailer in city disk must be covered")
	}
	if WarehouseCoversRetailer("US", cells, "US", "not-a-cell") {
		t.Fatal("outside cell must miss when a set exists")
	}
	if !WarehouseCoversRetailer("US", nil, "US", retailer) {
		t.Fatal("no cells + same country = whole country")
	}
	if WarehouseCoversRetailer("US", nil, "DE", retailer) {
		t.Fatal("no cells + different country must not cover")
	}
	if !WarehouseCoversRetailer("", nil, "US", retailer) {
		t.Fatal("empty warehouse country is unrestricted default")
	}
}

func TestCellInCoverage_ParentMatch(t *testing.T) {
	cells := CellsForCity(41.3111, 69.2797)
	if len(cells) == 0 {
		t.Fatal("tashkent cells")
	}
	if !CellInCoverage(cells[0], cells) {
		t.Fatal("exact cell must match")
	}
}
