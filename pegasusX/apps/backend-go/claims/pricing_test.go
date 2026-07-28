package claims

import (
	"math"
	"testing"
)

func TestAggregateClaimLines_MergesDuplicateSKUs(t *testing.T) {
	got := AggregateClaimLines([]ClaimLine{
		{SKU: "a", Quantity: 2, Reason: "DAMAGED"},
		{SKU: "a", Quantity: 3, Reason: "MISSING"},
		{SKU: "b", Quantity: 1},
	})
	if len(got) != 2 {
		t.Fatalf("len=%d want 2", len(got))
	}
	if got[0].SKU != "a" || got[0].Quantity != 5 {
		t.Fatalf("sku a = %+v", got[0])
	}
}

func TestPriceClaimLines_FromOrderUnitPrices(t *testing.T) {
	priced, total, err := PriceClaimLines(
		[]OrderLine{
			{SKU: "sku-a", Quantity: 10, UnitPriceMinor: 1500},
			{SKU: "sku-b", Quantity: 2, UnitPriceMinor: 5000},
		},
		[]ClaimLine{
			{SKU: "sku-a", Quantity: 3, Reason: "DAMAGED"},
			{SKU: "sku-b", Quantity: 1, Reason: "MISSING"},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	want := int64(3*1500 + 1*5000)
	if total != want {
		t.Fatalf("total=%d want %d", total, want)
	}
	if priced[0].UnitPriceMinor != 1500 || priced[0].AmountMinor != 4500 {
		t.Fatalf("line0=%+v", priced[0])
	}
}

func TestPriceClaimLines_RejectsSplitBypassOfQtyCap(t *testing.T) {
	// Two rows totaling 6 on an order of 5 must fail after aggregation.
	_, _, err := PriceClaimLines(
		[]OrderLine{{SKU: "sku-a", Quantity: 5, UnitPriceMinor: 100}},
		[]ClaimLine{
			{SKU: "sku-a", Quantity: 3},
			{SKU: "sku-a", Quantity: 3},
		},
	)
	if err == nil {
		t.Fatal("expected over-qty after aggregate")
	}
}

func TestPriceClaimLines_WithPriorClaims(t *testing.T) {
	_, _, err := PriceClaimLinesWithPrior(
		[]OrderLine{{SKU: "sku-a", Quantity: 5, UnitPriceMinor: 100}},
		[]ClaimLine{{SKU: "sku-a", Quantity: 3}},
		map[string]int64{"sku-a": 3}, // 3 already claimed → only 2 left
	)
	if err == nil {
		t.Fatal("expected remaining cap failure")
	}
	priced, total, err := PriceClaimLinesWithPrior(
		[]OrderLine{{SKU: "sku-a", Quantity: 5, UnitPriceMinor: 100}},
		[]ClaimLine{{SKU: "sku-a", Quantity: 2}},
		map[string]int64{"sku-a": 3},
	)
	if err != nil {
		t.Fatal(err)
	}
	if total != 200 || priced[0].Quantity != 2 {
		t.Fatalf("got total=%d lines=%+v", total, priced)
	}
}

func TestPriceClaimLines_WeightedAverageUnitPrice(t *testing.T) {
	// 2 @ 100 + 2 @ 300 → avg 200
	priced, total, err := PriceClaimLines(
		[]OrderLine{
			{SKU: "sku-a", Quantity: 2, UnitPriceMinor: 100},
			{SKU: "sku-a", Quantity: 2, UnitPriceMinor: 300},
		},
		[]ClaimLine{{SKU: "sku-a", Quantity: 2}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if priced[0].UnitPriceMinor != 200 {
		t.Fatalf("unit=%d want 200", priced[0].UnitPriceMinor)
	}
	if total != 400 {
		t.Fatalf("total=%d", total)
	}
}

func TestPriceClaimLines_RejectsUnknownSKU(t *testing.T) {
	_, _, err := PriceClaimLines(
		[]OrderLine{{SKU: "sku-a", Quantity: 1, UnitPriceMinor: 100}},
		[]ClaimLine{{SKU: "nope", Quantity: 1}},
	)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPriceClaimLines_RejectsOverQty(t *testing.T) {
	_, _, err := PriceClaimLines(
		[]OrderLine{{SKU: "sku-a", Quantity: 2, UnitPriceMinor: 100}},
		[]ClaimLine{{SKU: "sku-a", Quantity: 5}},
	)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMulInt64_Overflow(t *testing.T) {
	_, err := mulInt64(math.MaxInt64, 2)
	if err == nil {
		t.Fatal("expected overflow")
	}
	got, err := mulInt64(100, 200)
	if err != nil || got != 20000 {
		t.Fatalf("got %d %v", got, err)
	}
}

func TestClaimedQtyBySKU_IgnoresRejected(t *testing.T) {
	prior := []Claim{
		{ClaimID: "c1", Status: StatusResolved, LineItems: []ClaimLine{{SKU: "a", Quantity: 2}}},
		{ClaimID: "c2", Status: StatusRejected, LineItems: []ClaimLine{{SKU: "a", Quantity: 9}}},
		{ClaimID: "c3", Status: StatusOpen, LineItems: []ClaimLine{{SKU: "a", Quantity: 1}}},
	}
	got := ClaimedQtyBySKU(prior, "")
	if got["a"] != 3 {
		t.Fatalf("claimed=%d want 3", got["a"])
	}
	got = ClaimedQtyBySKU(prior, "c3")
	if got["a"] != 2 {
		t.Fatalf("exclude open: claimed=%d want 2", got["a"])
	}
}

func TestCapAmount(t *testing.T) {
	if CapAmount(500, 400) != 400 {
		t.Fatal("cap")
	}
	if CapAmount(100, 0) != 100 {
		t.Fatal("no cap")
	}
}
