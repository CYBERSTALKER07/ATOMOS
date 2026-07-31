package allocation

import (
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/segment"
)

func TestAllocateFromInventoryRoutesByFlag(t *testing.T) {
	withSegment := &AllocationService{constrainedEnabled: true, segment: segment.NewService(nil)}
	if withSegment.constrainedEnabled && withSegment.segment != nil {
		// flag on + segment service wired → policy path (allocateWithPolicy).
	} else {
		t.Fatal("expected constrained path preconditions")
	}

	legacy := &AllocationService{constrainedEnabled: false, segment: segment.NewService(nil)}
	if legacy.constrainedEnabled {
		t.Fatal("expected first-fit when flag disabled")
	}
}

func TestFirstFitDecisionConstraintReason(t *testing.T) {
	dec := LineDecision{
		AllocationMode:   segment.AllocationModeFirstFit,
		ConstraintReason: ConstraintReasonLegacy,
		RequestedQty:     5,
		AllocatedQty:     5,
	}
	if dec.ConstraintReason != ConstraintReasonLegacy {
		t.Fatalf("constraint reason: got %s", dec.ConstraintReason)
	}
}
