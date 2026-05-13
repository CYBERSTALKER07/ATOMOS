package order

import (
	"strings"
	"testing"
)

func TestActiveFulfillmentsSQL_IncludesCurrentAndLegacyAwaitingPaymentStates(t *testing.T) {
	if !strings.Contains(activeFulfillmentsSQL, "'AWAITING_PAYMENT'") {
		t.Fatalf("active fulfillments query missing AWAITING_PAYMENT state")
	}
	if !strings.Contains(activeFulfillmentsSQL, "'AWAITING_GLOBAL_PAYNT'") {
		t.Fatalf("active fulfillments query missing legacy AWAITING_GLOBAL_PAYNT state")
	}
}
