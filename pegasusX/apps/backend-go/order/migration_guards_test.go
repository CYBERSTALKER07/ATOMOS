package order

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestValidateStatusTransition_MigrationOpen(t *testing.T) {
	for _, next := range []Status{StatusPending, StatusScheduled, StatusCancelled} {
		if err := ValidateStatusTransition(string(StatusMigrationOpen), string(next), TransitionOpts{}); err != nil {
			t.Fatalf("expected MIGRATION_OPEN -> %s to be allowed, got %v", next, err)
		}
	}
	for _, forbidden := range []Status{StatusLoaded, StatusInTransit, StatusArrived, StatusCompleted, StatusAutoAccepted} {
		if err := ValidateStatusTransition(string(StatusMigrationOpen), string(forbidden), TransitionOpts{}); !errors.Is(err, ErrInvalidStatusTransition) {
			t.Fatalf("expected MIGRATION_OPEN -> %s to fail with ErrInvalidStatusTransition, got %v", forbidden, err)
		}
	}
}

func TestFiscalGuardsRejectMigratedAndHeldOrders(t *testing.T) {
	now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	admin := auth.Claims{Subject: "admin-1", Role: auth.RoleAdmin}

	cases := []struct {
		name   string
		mutate func(*Order)
	}{
		{
			name: "historical_migration_source",
			mutate: func(o *Order) {
				o.Source = OrderSourceMigration
				o.FiscalStatus = FiscalStatusHistoricalSkipped
				o.Status = StatusCompleted
			},
		},
		{
			name: "held_migration_open_status",
			mutate: func(o *Order) {
				o.Source = OrderSourceMigration
				o.Status = StatusMigrationOpen
			},
		},
		{
			name: "fiscal_status_historical_skipped",
			mutate: func(o *Order) {
				o.FiscalStatus = FiscalStatusHistoricalSkipped
				o.Status = StatusFiscalFailed
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			o := deliveryTestOrder(StatusAwaitingPayment)
			tc.mutate(&o)
			repo := &testRepo{found: true, order: o}
			svc := newTestService(repo, now)

			if _, err := svc.SubmitFiscalReceipt(context.Background(), driverClaims(), o.OrderID, "att-1"); !errors.Is(err, ErrFiscalAlreadySucceeded) {
				t.Fatalf("SubmitFiscalReceipt got %v want ErrFiscalAlreadySucceeded", err)
			}
			if _, err := svc.RetryFiscal(context.Background(), driverClaims(), o.OrderID); !errors.Is(err, ErrFiscalAlreadySucceeded) {
				t.Fatalf("RetryFiscal got %v want ErrFiscalAlreadySucceeded", err)
			}
			if _, err := svc.ForceCompleteOrder(context.Background(), admin, o.OrderID, ForceReasonOFDDown); !errors.Is(err, ErrFiscalAlreadySucceeded) {
				t.Fatalf("ForceCompleteOrder got %v want ErrFiscalAlreadySucceeded", err)
			}
			if repo.updateCalls != 0 {
				t.Fatalf("expected zero repository updates for excluded migration order, got %d", repo.updateCalls)
			}
		})
	}
}

func TestReleaseReservationsForOrderInTxn_MigrationSources(t *testing.T) {
	lines := []LineItem{{SKU: "sku-mig-1", Quantity: 3}}
	if err := ReleaseReservationsForOrderInTxn(t.Context(), nil, "sup-1", "wh-1", "ord-mig-1", OrderSourceMigration, lines); err != nil {
		t.Fatalf("OrderSourceMigration release must no-op cleanly, got %v", err)
	}
	if err := ReleaseReservationsForOrderInTxn(t.Context(), nil, "sup-1", "wh-1", "ord-mig-2", OrderSourceMigrationReleased, lines); err != nil {
		t.Fatalf("OrderSourceMigrationReleased nil-txn no-op failed: %v", err)
	}
}
