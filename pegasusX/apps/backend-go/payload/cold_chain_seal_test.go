package payload

import (
	"errors"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/stocklots"
)

func TestAssertPhysicalSealGates_EnforcesColdChainBaseline(t *testing.T) {
	origPick := stocklots.PickWavesEnabled()
	origLoad := stocklots.LoadLedgerEnabled()
	origCold := stocklots.ColdChainEnabled()
	stocklots.SetPickWavesEnabled(false)
	stocklots.SetLoadLedgerEnabled(false)
	stocklots.SetColdChainEnabled(true)
	t.Cleanup(func() {
		stocklots.SetPickWavesEnabled(origPick)
		stocklots.SetLoadLedgerEnabled(origLoad)
		stocklots.SetColdChainEnabled(origCold)
	})

	svc := newPayloadTestService(&payloadRepoSpy{}, &payloadCacheBackendSpy{})
	_, err := svc.assertPhysicalSealGates(t.Context(), "man-seal-cold-check")
	if !errors.Is(err, stocklots.ErrColdChainBaselineRequired) {
		t.Fatalf("expected ErrColdChainBaselineRequired from assertPhysicalSealGates, got %v", err)
	}
}
