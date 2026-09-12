package factory

import "testing"

func TestDispatchableTransferState_IncludesLoadingBay(t *testing.T) {
	for _, state := range []string{TransferStateCreated, TransferStateApproved, TransferStateLoading, "loading"} {
		if !dispatchableTransferState(state) {
			t.Fatalf("%s should be dispatchable (unassigned loading-bay)", state)
		}
	}
	for _, state := range []string{TransferStateCancelled, "DISPATCHED", "IN_TRANSIT"} {
		if dispatchableTransferState(state) {
			t.Fatalf("%s must not be dispatchable", state)
		}
	}
}
