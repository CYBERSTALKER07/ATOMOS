package order

import (
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
)

func tashkentDate(y, m, d int) time.Time {
	return time.Date(y, time.Month(m), d, 12, 0, 0, 0, time.FixedZone("Asia/Tashkent", 5*3600))
}

func TestComputeDeliveryExpectation_standard(t *testing.T) {
	now := tashkentDate(2026, 6, 28)
	deliver := tashkentDate(2026, 6, 29)
	o := Order{
		Source:        OrderSourceManual,
		Status:        StatusPending,
		DeliverBefore: &deliver,
	}
	loc := proximity.TashkentLocation
	exp := ComputeDeliveryExpectation(now, loc, o)
	if exp.Kind != ExpectationKindStandard {
		t.Fatalf("kind=%s", exp.Kind)
	}
	if exp.Urgency != ExpectationUrgencyDueSoon {
		t.Fatalf("urgency=%s want due_soon", exp.Urgency)
	}
	if exp.TargetLabel == "" {
		t.Fatal("expected target label")
	}
}

func TestComputeDeliveryExpectation_express(t *testing.T) {
	now := tashkentDate(2026, 6, 28)
	deliver := tashkentDate(2026, 6, 29)
	o := Order{
		Source:           OrderSourceManual,
		Status:           StatusPending,
		DeliveryPriority: DeliveryPriorityExpress,
		DeliverBefore:    &deliver,
	}
	loc := proximity.TashkentLocation
	exp := ComputeDeliveryExpectation(now, loc, o)
	if exp.Kind != ExpectationKindExpress {
		t.Fatalf("kind=%s", exp.Kind)
	}
	if exp.TargetLabel == "" || exp.TargetDate == nil {
		t.Fatal("expected express target")
	}
}

func TestComputeDeliveryExpectation_preorder_scheduled_far(t *testing.T) {
	now := tashkentDate(2026, 6, 28)
	requested := tashkentDate(2026, 7, 15)
	o := Order{
		Source:                OrderSourceManualPreorder,
		Status:                StatusScheduled,
		RequestedDeliveryDate: &requested,
	}
	loc := proximity.TashkentLocation
	exp := ComputeDeliveryExpectation(now, loc, o)
	if exp.Kind != ExpectationKindScheduledPreorder {
		t.Fatalf("kind=%s", exp.Kind)
	}
	if exp.Urgency != ExpectationUrgencyScheduledFar {
		t.Fatalf("urgency=%s", exp.Urgency)
	}
}

func TestComputeDeliveryExpectation_proposal_pending(t *testing.T) {
	now := tashkentDate(2026, 6, 28)
	proposed := tashkentDate(2026, 7, 5)
	o := Order{
		Source:               OrderSourceManualPreorder,
		Status:               StatusScheduled,
		ConfirmationStatus:   ConfirmationStatusPendingWarehouse,
		ProposedDeliveryDate: &proposed,
	}
	loc := proximity.TashkentLocation
	exp := ComputeDeliveryExpectation(now, loc, o)
	if exp.Kind != ExpectationKindProposalPending {
		t.Fatalf("kind=%s", exp.Kind)
	}
	if exp.TargetLabel == "" {
		t.Fatal("expected proposal label")
	}
}

func TestComputeDeliveryExpectation_delayed_standard(t *testing.T) {
	now := tashkentDate(2026, 6, 30)
	deliver := tashkentDate(2026, 6, 28)
	o := Order{
		Source:        OrderSourceManual,
		Status:        StatusPending,
		DeliverBefore: &deliver,
	}
	loc := proximity.TashkentLocation
	exp := ComputeDeliveryExpectation(now, loc, o)
	if !exp.Delayed {
		t.Fatal("expected delayed")
	}
	if exp.Urgency != ExpectationUrgencyOverdue {
		t.Fatalf("urgency=%s", exp.Urgency)
	}
}

func TestComputeDeliveryExpectation_warehouse_delayed_status(t *testing.T) {
	now := tashkentDate(2026, 6, 28)
	deliver := tashkentDate(2026, 6, 30)
	o := Order{
		Source:        OrderSourceManual,
		Status:        StatusDelayed,
		DeliverBefore: &deliver,
	}
	loc := proximity.TashkentLocation
	exp := ComputeDeliveryExpectation(now, loc, o)
	if !exp.Delayed || exp.DelayReason == "" {
		t.Fatal("expected warehouse delay reason")
	}
}

func TestComputeDeliveryExpectation_completed_not_delayed(t *testing.T) {
	now := tashkentDate(2026, 7, 5)
	deliver := tashkentDate(2026, 6, 28)
	o := Order{
		Source:        OrderSourceManual,
		Status:        StatusCompleted,
		DeliverBefore: &deliver,
	}
	loc := proximity.TashkentLocation
	exp := ComputeDeliveryExpectation(now, loc, o)
	if exp.Delayed {
		t.Fatal("completed orders must not be delayed")
	}
}
