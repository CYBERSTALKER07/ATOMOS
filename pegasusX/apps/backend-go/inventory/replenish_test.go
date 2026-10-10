package inventory

import (
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/stocklots"
)

func TestCreditSupplierInventoryV2InTxn_FailClosedWhenLotsEnabled(t *testing.T) {
	stocklots.SetLotsEnabled(true)
	t.Cleanup(func() { stocklots.SetLotsEnabled(false) })

	err := CreditSupplierInventoryV2InTxn(t.Context(), nil, "s1", "w1", "p1", 5)
	if err == nil {
		t.Fatal("expected fail-closed error when WMS_LOTS_ENABLED")
	}
	if err != ErrLotsEnabledDirectV2 {
		t.Fatalf("got %v want %v", err, ErrLotsEnabledDirectV2)
	}
}

func TestCreditSupplierInventoryV2InTxn_AllowsZeroQtyWhenLotsEnabled(t *testing.T) {
	// Zero qty returns before lots check would matter for no-op — still fail-closed first.
	stocklots.SetLotsEnabled(true)
	t.Cleanup(func() { stocklots.SetLotsEnabled(false) })
	err := CreditSupplierInventoryV2InTxn(t.Context(), nil, "s1", "w1", "p1", 0)
	if err != ErrLotsEnabledDirectV2 {
		t.Fatalf("got %v want %v", err, ErrLotsEnabledDirectV2)
	}
}
