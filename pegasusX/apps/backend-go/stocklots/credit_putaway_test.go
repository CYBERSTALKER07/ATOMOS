package stocklots

import (
	"testing"
)

func TestCreditViaDefaultPutaway_Defaults(t *testing.T) {
	if DefaultRecvLocationID != "recv-default" {
		t.Fatalf("recv default = %q", DefaultRecvLocationID)
	}
	if DefaultReturnsLocationID != "returns-default" {
		t.Fatalf("returns default = %q", DefaultReturnsLocationID)
	}
}

func TestCreditViaDefaultPutawayInTxn_Validation(t *testing.T) {
	SetLotsEnabled(true)
	t.Cleanup(func() { SetLotsEnabled(false) })

	_, err := CreditViaDefaultPutawayInTxn(t.Context(), nil, "", "w1", "p1", "", "", 1)
	if err == nil {
		t.Fatal("expected validation error for empty supplier")
	}
	res, err := CreditViaDefaultPutawayInTxn(t.Context(), nil, "s1", "w1", "p1", "", "", 0)
	if err != nil {
		t.Fatalf("zero qty: %v", err)
	}
	if res != nil {
		t.Fatalf("zero qty should return nil result, got %+v", res)
	}
}
