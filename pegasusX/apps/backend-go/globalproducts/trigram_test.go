package globalproducts

import (
	"testing"
)

func TestTrigramSimilarity(t *testing.T) {
	// Exact matches
	if sim := TrigramSimilarity("Coca Cola 1L", "Coca Cola 1L"); sim < 0.99 {
		t.Fatalf("expected exact match sim ~ 1.0, got %f", sim)
	}

	// Minor typo / variation
	simMinor := TrigramSimilarity("Coca Cola 1L", "Coca-Cola 1L")
	if simMinor < 0.7 {
		t.Fatalf("expected high similarity for minor punctuation variation, got %f", simMinor)
	}

	// Completely different
	simDiff := TrigramSimilarity("Coca Cola", "Pepsi Max")
	if simDiff > 0.3 {
		t.Fatalf("expected low similarity for completely different products, got %f", simDiff)
	}
}
