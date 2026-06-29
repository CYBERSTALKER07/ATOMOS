package supplier

import "testing"

func TestExceptionSeverity(t *testing.T) {
	if got := exceptionSeverity(1, 0, 0); got != "medium" {
		t.Fatalf("expected medium got %q", got)
	}
	if got := exceptionSeverity(2, 2, 1); got != "high" {
		t.Fatalf("expected high got %q", got)
	}
	if got := exceptionSeverity(0, 1, 0); got != "low" {
		t.Fatalf("expected low got %q", got)
	}
}

func TestExceptionMapBucketAggregation(t *testing.T) {
	buckets := map[string]*exceptionMapBucket{}
	add := func(cell, kind, orderID string) {
		b, ok := buckets[cell]
		if !ok {
			b = &exceptionMapBucket{h3Cell: cell}
			buckets[cell] = b
		}
		switch kind {
		case "shop_closed":
			b.shopClosed++
		case "delayed":
			b.delayed++
		case "manifest_gate":
			b.manifest++
		}
		if orderID != "" {
			b.orderIDs = append(b.orderIDs, orderID)
		}
	}
	add("8928308280fffff", "shop_closed", "ord-1")
	add("8928308280fffff", "delayed", "ord-2")
	add("8928308280fffff", "manifest_gate", "ord-3")

	cell := buckets["8928308280fffff"]
	if cell == nil || cell.shopClosed != 1 || cell.delayed != 1 || cell.manifest != 1 {
		t.Fatalf("unexpected bucket counts: %+v", cell)
	}
	sev := exceptionSeverity(cell.shopClosed, cell.delayed, cell.manifest)
	if sev != "high" {
		t.Fatalf("expected high severity got %q", sev)
	}
}
