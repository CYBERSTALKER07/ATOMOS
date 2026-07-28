package claims

import "testing"

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
