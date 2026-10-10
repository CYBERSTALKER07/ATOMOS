package catalog

import (
	"context"
	"testing"
)

func TestStockEnricher_NilSafe(t *testing.T) {
	t.Parallel()
	var e *StockEnricher
	if got := e.Enrich(context.Background(), "ret-1", []Product{{ProductID: "p1", SupplierID: "s1"}}); len(got) != 0 {
		t.Fatalf("%#v", got)
	}
}

func TestResolvePolicy(t *testing.T) {
	t.Parallel()
	if got := resolvePolicy("REJECT", "ACCEPT_BACKORDER"); got != "ACCEPT_BACKORDER" {
		t.Fatalf("%s", got)
	}
	if got := resolvePolicy("ACCEPT_BACKORDER", ""); got != "ACCEPT_BACKORDER" {
		t.Fatalf("%s", got)
	}
	if got := resolvePolicy("", ""); got != "REJECT" {
		t.Fatalf("%s", got)
	}
}
