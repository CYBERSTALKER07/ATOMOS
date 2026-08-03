package order

import (
	"errors"
	"testing"
)

func TestApplyPartialOffloadLines_qtyMath(t *testing.T) {
	current := []LineItem{
		{SKU: "A", Quantity: 10, UnitPrice: 100},
		{SKU: "B", Quantity: 5, UnitPrice: 200},
	}
	updated, del, rem, err := ApplyPartialOffloadLines(current, []PartialOffloadLine{
		{OrderLineID: "A", DeliveredQty: 7, RemainingQty: 3, Reason: OffloadReasonShopRefused},
		{OrderLineID: "B", DeliveredQty: 5, RemainingQty: 0},
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	if del != 7*100+5*200 {
		t.Fatalf("delivered_minor=%d", del)
	}
	if rem != 3*100 {
		t.Fatalf("remaining_minor=%d", rem)
	}
	if updated[0].OffloadStatus != OffloadStatusPartial {
		t.Fatalf("A status=%s", updated[0].OffloadStatus)
	}
	if updated[1].OffloadStatus != OffloadStatusFull {
		t.Fatalf("B status=%s", updated[1].OffloadStatus)
	}
}

func TestApplyPartialOffloadLines_mismatch(t *testing.T) {
	current := []LineItem{{SKU: "A", Quantity: 10, UnitPrice: 100}}
	_, _, _, err := ApplyPartialOffloadLines(current, []PartialOffloadLine{
		{OrderLineID: "A", DeliveredQty: 4, RemainingQty: 4},
	}, true)
	if !errors.Is(err, ErrPartialQtyMismatch) {
		t.Fatalf("want ErrPartialQtyMismatch, got %v", err)
	}
}

func TestApplyPartialOffloadLines_unknownSKU(t *testing.T) {
	current := []LineItem{{SKU: "A", Quantity: 1, UnitPrice: 100}}
	_, _, _, err := ApplyPartialOffloadLines(current, []PartialOffloadLine{
		{OrderLineID: "Z", DeliveredQty: 1, RemainingQty: 0},
	}, true)
	if !errors.Is(err, ErrPartialUnknownSKU) {
		t.Fatalf("want ErrPartialUnknownSKU, got %v", err)
	}
}

func TestApplyPartialOffloadLines_noneDelivered(t *testing.T) {
	current := []LineItem{{SKU: "A", Quantity: 3, UnitPrice: 50}}
	updated, del, rem, err := ApplyPartialOffloadLines(current, []PartialOffloadLine{
		{OrderLineID: "A", DeliveredQty: 0, RemainingQty: 3, Reason: OffloadReasonMissing},
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	if del != 0 || rem != 150 {
		t.Fatalf("del=%d rem=%d", del, rem)
	}
	if updated[0].OffloadStatus != OffloadStatusNone {
		t.Fatalf("status=%s", updated[0].OffloadStatus)
	}
}
