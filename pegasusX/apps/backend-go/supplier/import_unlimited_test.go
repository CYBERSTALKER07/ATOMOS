package supplier

import (
	"fmt"
	"strings"
	"testing"
)

func TestDiscoverImportDelimited_No10kRowCeiling(t *testing.T) {
	var b strings.Builder
	b.WriteString("warehouse_id,product_id,quantity_on_hand,product_name,price_minor\n")
	for i := 1; i <= 15000; i++ {
		fmt.Fprintf(&b, "WH-1,SKU-%05d,%d,Product %d,12000\n", i, i%500, i)
	}
	outcome, headers, rows, err := discoverImportDelimited([]byte(b.String()), ',')
	if err != nil {
		t.Fatalf("discoverImportDelimited: %v", err)
	}
	if len(headers) != 5 {
		t.Fatalf("expected 5 headers, got %d", len(headers))
	}
	if len(rows) != 15000 {
		t.Fatalf("expected all 15,000 rows without 10k truncation, got %d", len(rows))
	}
	if len(outcome.Mappings) != 5 {
		t.Fatalf("expected 5 mappings, got %d", len(outcome.Mappings))
	}
}
