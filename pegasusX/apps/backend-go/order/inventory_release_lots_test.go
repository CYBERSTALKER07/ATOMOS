package order

import (
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/stocklots"
)

func TestReleaseReservations_LotsEnabledRequiresOrderID(t *testing.T) {
	stocklots.SetLotsEnabled(true)
	t.Cleanup(func() { stocklots.SetLotsEnabled(false) })

	err := ReleaseReservationsForOrderInTxn(
		t.Context(), nil, "s1", "w1", "", OrderSource(""),
		[]LineItem{{SKU: "p1", Quantity: 1}},
	)
	if err == nil {
		t.Fatal("expected error when lots enabled and orderID empty")
	}
}
