package supplier

import (
	"fmt"
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════════════
// CASE D: MASSIVE SPLIT — Scale & Correctness Tests
//
// Verifies the ManifestSplitter handles 2,000 line items across 50 categories,
// splitting into chunks of ≤ 25 stops (Rule of 25) without dropping a single SKU.
// ═══════════════════════════════════════════════════════════════════════════════

// makeGeoOrders generates n GeoOrders with unique IDs and varying volumes.
func makeGeoOrders(n int) []GeoOrder {
	orders := make([]GeoOrder, n)
	for i := 0; i < n; i++ {
		orders[i] = GeoOrder{
			OrderID:      fmt.Sprintf("ORD-%05d", i),
			RetailerID:   fmt.Sprintf("RET-%03d", i%200),
			RetailerName: fmt.Sprintf("Shop-%d", i%200),
			Amount:       int64(1000 + i%500),
			Lat:          41.30 + float64(i%50)*0.01,
			Lng:          69.24 + float64(i%50)*0.01,
			Volume:       float64(1 + i%5), // 1.0 to 5.0 VU
		}
	}
	return orders
}

// TestSplitManifest_2000Orders_RuleOf25 is the core "Massive Split" test.
// 2,000 orders must be split into ceil(2000/25) = 80 chunks.
func TestSplitManifest_2000Orders_RuleOf25(t *testing.T) {
	orders := makeGeoOrders(2000)
	group := SplitManifest("driver-massive", "truck-massive", orders, 25)

	// Total orders preserved — no SKU dropped
	if group.TotalOrders != 2000 {
		t.Errorf("TotalOrders = %d, want 2000", group.TotalOrders)
	}

	// Correct number of chunks
	expectedChunks := (2000 + 25 - 1) / 25 // ceil(2000/25) = 80
	if len(group.Chunks) != expectedChunks {
		t.Fatalf("chunks = %d, want %d", len(group.Chunks), expectedChunks)
	}

	// Every chunk has ≤ 25 orders
	totalAcrossChunks := 0
	for i, chunk := range group.Chunks {
		if len(chunk.Orders) > 25 {
			t.Errorf("chunk %d has %d orders, max 25", i, len(chunk.Orders))
		}
		totalAcrossChunks += len(chunk.Orders)
	}

	// Cross-check: sum across all chunks == original count
	if totalAcrossChunks != 2000 {
		t.Errorf("sum across chunks = %d, want 2000", totalAcrossChunks)
	}

	// RouteIDs are unique
	routeSet := make(map[string]bool, len(group.Chunks))
	for _, chunk := range group.Chunks {
		if routeSet[chunk.RouteID] {
			t.Errorf("duplicate RouteID: %s", chunk.RouteID)
		}
		routeSet[chunk.RouteID] = true
	}

	// Suffixes are sequential alphabetical (A, B, C, ..., Z, AA, AB, ...)
	for i, chunk := range group.Chunks {
		expected := alphaIndex(i)
		if chunk.Suffix != expected {
			t.Errorf("chunk %d: suffix = %q, want %q", i, chunk.Suffix, expected)
		}
	}

	// VolumeVU per chunk is correctly summed
	for i, chunk := range group.Chunks {
		var expectedVol float64
		for _, o := range chunk.Orders {
			expectedVol += o.Volume
		}
		if chunk.VolumeVU != expectedVol {
			t.Errorf("chunk %d: VolumeVU = %f, want %f", i, chunk.VolumeVU, expectedVol)
		}
	}
}

// TestSplitManifest_NoSplit verifies a small order set produces a single chunk
// with no suffix.
func TestSplitManifest_NoSplit(t *testing.T) {
	orders := makeGeoOrders(10)
	group := SplitManifest("driver-small", "truck-small", orders, 25)

	if len(group.Chunks) != 1 {
		t.Fatalf("chunks = %d, want 1", len(group.Chunks))
	}
	if group.Chunks[0].Suffix != "" {
		t.Errorf("single chunk should have empty suffix, got %q", group.Chunks[0].Suffix)
	}
	if len(group.Chunks[0].Orders) != 10 {
		t.Errorf("chunk has %d orders, want 10", len(group.Chunks[0].Orders))
	}
}

// TestSplitManifest_DefaultMaxStops verifies maxStops=0 defaults to 25.
func TestSplitManifest_DefaultMaxStops(t *testing.T) {
	orders := makeGeoOrders(60)
	group := SplitManifest("driver-default", "truck-default", orders, 0)

	expectedChunks := (60 + 25 - 1) / 25 // 3
	if len(group.Chunks) != expectedChunks {
		t.Fatalf("chunks = %d, want %d (maxStops=0 should default to 25)", len(group.Chunks), expectedChunks)
	}
	for _, chunk := range group.Chunks {
		if len(chunk.Orders) > 25 {
			t.Errorf("chunk with %d orders exceeds default max of 25", len(chunk.Orders))
		}
	}
}

// TestSplitManifest_ExactBoundary verifies no off-by-one when orders == maxStops.
func TestSplitManifest_ExactBoundary(t *testing.T) {
	orders := makeGeoOrders(25)
	group := SplitManifest("driver-exact", "truck-exact", orders, 25)

	if len(group.Chunks) != 1 {
		t.Errorf("exactly 25 orders should produce 1 chunk, got %d", len(group.Chunks))
	}
}

// TestSplitManifest_26Orders verifies the smallest split case.
func TestSplitManifest_26Orders(t *testing.T) {
	orders := makeGeoOrders(26)
	group := SplitManifest("driver-26", "truck-26", orders, 25)

	if len(group.Chunks) != 2 {
		t.Fatalf("26 orders should produce 2 chunks, got %d", len(group.Chunks))
	}
	if len(group.Chunks[0].Orders) != 25 {
		t.Errorf("first chunk: %d orders, want 25", len(group.Chunks[0].Orders))
	}
	if len(group.Chunks[1].Orders) != 1 {
		t.Errorf("second chunk: %d orders, want 1", len(group.Chunks[1].Orders))
	}
	if group.Chunks[0].Suffix != "A" || group.Chunks[1].Suffix != "B" {
		t.Errorf("suffixes: %q, %q — want A, B", group.Chunks[0].Suffix, group.Chunks[1].Suffix)
	}
}

// TestSplitManifest_OrderPreservation verifies input order is preserved in chunks.
func TestSplitManifest_OrderPreservation(t *testing.T) {
	orders := makeGeoOrders(100)
	group := SplitManifest("driver-preserve", "truck-preserve", orders, 25)

	idx := 0
	for _, chunk := range group.Chunks {
		for _, o := range chunk.Orders {
			if o.OrderID != orders[idx].OrderID {
				t.Fatalf("order at position %d: got %s, want %s — order not preserved",
					idx, o.OrderID, orders[idx].OrderID)
			}
			idx++
		}
	}
}

// TestAlphaIndex verifies the base-26 encoding used for chunk suffixes.
func TestAlphaIndex(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{0, "A"},
		{1, "B"},
		{25, "Z"},
		{26, "AA"},
		{27, "AB"},
		{51, "AZ"},
		{52, "BA"},
		{701, "ZZ"},
		{702, "AAA"},
	}
	for _, tt := range tests {
		got := alphaIndex(tt.input)
		if got != tt.want {
			t.Errorf("alphaIndex(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// TestSplitManifest_50Categories verifies category diversity doesn't affect splitting.
// The splitter is category-agnostic — it splits by count only.
func TestSplitManifest_50Categories(t *testing.T) {
	orders := makeGeoOrders(2000)
	// Simulate 50 categories by varying Volume (which acts as weight proxy)
	for i := range orders {
		orders[i].Volume = float64(1+(i%50)) * 0.5 // 0.5 to 25.0 VU
	}

	group := SplitManifest("driver-cat", "truck-cat", orders, 25)

	// All orders accounted for
	total := 0
	for _, chunk := range group.Chunks {
		total += len(chunk.Orders)
	}
	if total != 2000 {
		t.Errorf("total across chunks = %d, want 2000", total)
	}
}
