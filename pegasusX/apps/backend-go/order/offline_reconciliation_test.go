package order

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestOfflineReconciliation_ActiveOrderDelivery(t *testing.T) {
	ctx := auth.WithTenant(context.Background(), auth.TenantContext{
		SupplierID: "sup-recon-1",
		Source:     "offline_recon_test",
	})

	repo := &testRepo{
		found: true,
		order: Order{
			OrderID:    "ord-active-1",
			SupplierID: "sup-recon-1",
			RetailerID: "ret-1",
			Status:     StatusArrived,
			TotalMinor: 50000,
			Currency:   "UZS",
		},
	}
	svc := NewService(ServiceConfig{
		Repo:         repo,
		SupplierID:   "sup-recon-1",
		SupplierName: "Recon Supplier",
		Currency:     "UZS",
		Log:          slog.Default(),
	})

	res, err := svc.ReconcileOfflineDelivery(ctx, OfflineDeliverySyncRequest{
		OrderID:        "ord-active-1",
		DriverID:       "drv-101",
		DeliveredAt:    time.Now().UTC(),
		ProofSignature: "data:image/png;base64,sig",
		Notes:          "Delivered at backdoor",
	})
	if err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}

	if res.Resolution != ResolutionNormalDelivery {
		t.Errorf("expected NORMAL_DELIVERY, got %s", res.Resolution)
	}
	if res.FinalStatus != StatusDeliveredOnCredit {
		t.Errorf("expected DELIVERED_ON_CREDIT, got %s", res.FinalStatus)
	}
	if repo.captured.Status != StatusDeliveredOnCredit {
		t.Errorf("expected repo captured status DELIVERED_ON_CREDIT, got %s", repo.captured.Status)
	}
}

func TestOfflineReconciliation_PhysicalCustodySupremacy(t *testing.T) {
	ctx := auth.WithTenant(context.Background(), auth.TenantContext{
		SupplierID: "sup-recon-1",
		Source:     "offline_recon_test",
	})

	// Order was cancelled online while driver was in offline zone!
	repo := &testRepo{
		found: true,
		order: Order{
			OrderID:    "ord-conflict-1",
			SupplierID: "sup-recon-1",
			RetailerID: "ret-1",
			Status:     StatusCancelled,
			TotalMinor: 75000,
			Currency:   "UZS",
		},
	}
	svc := NewService(ServiceConfig{
		Repo:         repo,
		SupplierID:   "sup-recon-1",
		SupplierName: "Recon Supplier",
		Currency:     "UZS",
		Log:          slog.Default(),
	})

	res, err := svc.ReconcileOfflineDelivery(ctx, OfflineDeliverySyncRequest{
		OrderID:        "ord-conflict-1",
		DriverID:       "drv-hero",
		DeliveredAt:    time.Now().UTC().Add(-10 * time.Minute),
		ProofSignature: "signature_bytes",
		Notes:          "Handed over to shop owner",
	})
	if err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}

	// Invariant assertion: Physical delivery won over online cancel
	if res.Resolution != ResolutionPhysicalCustodyWon {
		t.Errorf("expected PHYSICAL_CUSTODY_WON, got %s", res.Resolution)
	}
	if !res.Disputed {
		t.Error("expected Disputed to be true")
	}
	if res.FinalStatus != StatusDeliveredOnCredit {
		t.Errorf("expected DELIVERED_ON_CREDIT, got %s", res.FinalStatus)
	}
	if repo.captured.Status != StatusDeliveredOnCredit {
		t.Errorf("expected repo captured status DELIVERED_ON_CREDIT, got %s", repo.captured.Status)
	}
	if !strings.Contains(repo.captured.WarehouseNotes, "PHYSICAL_CUSTODY_WON") {
		t.Errorf("expected warehouse notes to document custody resolution, got %s", repo.captured.WarehouseNotes)
	}
}

func TestOfflineReconciliation_IdempotentAlreadyCompleted(t *testing.T) {
	ctx := context.Background()
	repo := &testRepo{
		found: true,
		order: Order{
			OrderID:    "ord-done-1",
			Status:     StatusCompleted,
			TotalMinor: 30000,
			Currency:   "UZS",
		},
	}
	svc := NewService(ServiceConfig{
		Repo:       repo,
		SupplierID: "sup-1",
		Currency:   "UZS",
		Log:        slog.Default(),
	})

	res, err := svc.ReconcileOfflineDelivery(ctx, OfflineDeliverySyncRequest{
		OrderID:     "ord-done-1",
		DriverID:    "drv-1",
		DeliveredAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}

	if res.Resolution != ResolutionIdempotentNoop {
		t.Errorf("expected IDEMPOTENT_NOOP, got %s", res.Resolution)
	}
	if res.FinalStatus != StatusCompleted {
		t.Errorf("expected COMPLETED, got %s", res.FinalStatus)
	}
}
