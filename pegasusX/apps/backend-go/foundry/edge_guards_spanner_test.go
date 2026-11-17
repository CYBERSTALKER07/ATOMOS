package foundry_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/foundry"
)

type fakeEdgeGuardRepo struct {
	mu          sync.Mutex
	exposure    map[string]int64
	leases      map[string]*foundry.OfflineQuotaLease
	injectedErr error
}

func newFakeEdgeGuardRepo() *fakeEdgeGuardRepo {
	return &fakeEdgeGuardRepo{
		exposure: make(map[string]int64),
		leases:   make(map[string]*foundry.OfflineQuotaLease),
	}
}

func (f *fakeEdgeGuardRepo) amlKey(tin, sup, date string) string {
	return fmt.Sprintf("%s::%s::%s", tin, sup, date)
}

func (f *fakeEdgeGuardRepo) CheckAndIncrementAMLExposure(
	_ context.Context,
	legalEntityTIN string,
	supplierID string,
	calendarDate string,
	deltaMinor int64,
	ceilingMinor int64,
	enforced bool,
) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.injectedErr != nil {
		return 0, f.injectedErr
	}
	k := f.amlKey(legalEntityTIN, supplierID, calendarDate)
	cur := f.exposure[k]
	if enforced && ceilingMinor > 0 && cur+deltaMinor > ceilingMinor {
		return cur, &foundry.AMLDailyCashLimitExceededError{
			LegalEntityTIN:       legalEntityTIN,
			CurrentExposureMinor: cur,
			AttemptedCashMinor:   deltaMinor,
			CeilingMinor:         ceilingMinor,
			CalendarDate:         calendarDate,
		}
	}
	f.exposure[k] = cur + deltaMinor
	return f.exposure[k], nil
}

func (f *fakeEdgeGuardRepo) GetAMLExposureMinor(
	_ context.Context,
	legalEntityTIN string,
	supplierID string,
	calendarDate string,
) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.injectedErr != nil {
		return 0, f.injectedErr
	}
	return f.exposure[f.amlKey(legalEntityTIN, supplierID, calendarDate)], nil
}

func (f *fakeEdgeGuardRepo) IssueOfflineQuotaLease(
	_ context.Context,
	calendarDate string,
	lease *foundry.OfflineQuotaLease,
	ceilingMinor int64,
	enforced bool,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.injectedErr != nil {
		return f.injectedErr
	}
	k := f.amlKey(lease.LegalEntityTIN, lease.SupplierID, calendarDate)
	cur := f.exposure[k]
	if enforced && ceilingMinor > 0 && cur+lease.MaxCashAllowedMinor > ceilingMinor {
		return &foundry.AMLDailyCashLimitExceededError{
			LegalEntityTIN:       lease.LegalEntityTIN,
			CurrentExposureMinor: cur,
			AttemptedCashMinor:   lease.MaxCashAllowedMinor,
			CeilingMinor:         ceilingMinor,
			CalendarDate:         calendarDate,
		}
	}
	f.exposure[k] = cur + lease.MaxCashAllowedMinor
	cp := *lease
	f.leases[lease.LeaseID] = &cp
	return nil
}

func (f *fakeEdgeGuardRepo) GetOfflineQuotaLease(_ context.Context, leaseID string) (*foundry.OfflineQuotaLease, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.injectedErr != nil {
		return nil, f.injectedErr
	}
	lease, ok := f.leases[leaseID]
	if !ok {
		return nil, foundry.ErrObjectNotFound
	}
	cp := *lease
	return &cp, nil
}

func TestEdgeGuards_SpannerBackedSharedAcrossPods(t *testing.T) {
	ctx := context.Background()
	sharedRepo := newFakeEdgeGuardRepo()

	// Two independent Engine instances simulate two separate API pods.
	pod1 := foundry.NewEngine().WithEdgeGuardRepository(sharedRepo)
	pod2 := foundry.NewEngine().WithEdgeGuardRepository(sharedRepo)

	// Pod 1 records 1.5B Tiyin on TIN-309988776.
	total, err := pod1.CheckAndRecordAMLCashExposure(ctx, "TIN-309988776", "sup_1", "2026-11-13", 1_500_000_000)
	if err != nil {
		t.Fatalf("pod1 AML exposure failed: %v", err)
	}
	if total != 1_500_000_000 {
		t.Fatalf("expected 1_500_000_000, got %d", total)
	}

	// Pod 2 immediately sees Pod 1's Spanner exposure and blocks smurfing above 2.5B Tiyin.
	if _, err := pod2.CheckAndRecordAMLCashExposure(ctx, "TIN-309988776", "sup_1", "2026-11-13", 1_200_000_000); !errors.Is(err, foundry.ErrAMLCeilingExceeded) {
		t.Fatalf("expected pod2 to enforce Spanner AML ceiling, got err=%v", err)
	}

	// Pod 2 Doorstep ePOD precondition guard must also check Spanner AML exposure.
	rec, err := pod2.ExecuteKineticAction(ctx, foundry.ActionExecutionRequest{
		CellID:         "cell-uz",
		ActionType:     foundry.ActionCompleteDoorstepEPOD,
		SupplierID:     "sup_1",
		IdempotencyKey: "epod-cross-pod-check-1",
		Parameters: map[string]any{
			"legal_entity_tin":     "TIN-309988776",
			"calendar_date":        "2026-11-13",
			"cash_collected_minor": int64(1_200_000_000),
		},
	}, nil)
	if !errors.Is(err, foundry.ErrPreconditionFailed) {
		t.Fatalf("expected ErrPreconditionFailed from pre-execution AML guard across pods, got rec=%+v err=%v", rec, err)
	}
	if rec == nil || rec.Status != "REJECTED_PRECONDITION" {
		t.Fatalf("expected REJECTED_PRECONDITION record, got %+v", rec)
	}

	// Pod 2 issues an offline quota lease (500M cash, 200M credit) -> total AML becomes 2.0B.
	lease, err := pod2.IssueOfflineQuotaLease(ctx, "ord_100", "drv_1", "sup_1", "TIN-309988776", "2026-11-13", 500_000_000, 200_000_000, 2*time.Hour)
	if err != nil {
		t.Fatalf("pod2 IssueOfflineQuotaLease failed: %v", err)
	}

	// Pod 1 validates collection against lease stored in shared Spanner repo.
	if err := pod1.ValidateOfflineLeaseCollection(ctx, lease.LeaseID, 400_000_000, 100_000_000); err != nil {
		t.Fatalf("pod1 ValidateOfflineLeaseCollection within cap failed: %v", err)
	}
	if err := pod1.ValidateOfflineLeaseCollection(ctx, lease.LeaseID, 600_000_000, 0); !errors.Is(err, foundry.ErrOfflineLeaseExceeded) {
		t.Fatalf("expected ErrOfflineLeaseExceeded for excess cash on pod1, got %v", err)
	}
	if err := pod1.ValidateOfflineLeaseCollection(ctx, "missing-lease-id", 1, 1); !errors.Is(err, foundry.ErrOfflineLeaseExceeded) {
		t.Fatalf("expected ErrOfflineLeaseExceeded for missing lease, got %v", err)
	}
}

func TestEdgeGuards_SpannerErrorsPropagateNeverDrop(t *testing.T) {
	ctx := context.Background()
	repo := newFakeEdgeGuardRepo()
	repo.injectedErr = errors.New("spanner deadline exceeded")

	eng := foundry.NewEngine().WithEdgeGuardRepository(repo)

	if _, err := eng.CheckAndRecordAMLCashExposure(ctx, "TIN-ERR", "sup_1", "2026-11-13", 1000); !errors.Is(err, repo.injectedErr) {
		t.Fatalf("expected Spanner error to propagate on CheckAndRecordAMLCashExposure, got %v", err)
	}
	if _, err := eng.IssueOfflineQuotaLease(ctx, "ord_err", "drv_err", "sup_1", "TIN-ERR", "2026-11-13", 1000, 500, time.Hour); !errors.Is(err, repo.injectedErr) {
		t.Fatalf("expected Spanner error to propagate on IssueOfflineQuotaLease, got %v", err)
	}
	if err := eng.ValidateOfflineLeaseCollection(ctx, "lease-err", 100, 100); !errors.Is(err, repo.injectedErr) {
		t.Fatalf("expected Spanner error to propagate on ValidateOfflineLeaseCollection, got %v", err)
	}

	rec, err := eng.ExecuteKineticAction(ctx, foundry.ActionExecutionRequest{
		CellID:         "cell-uz",
		ActionType:     foundry.ActionCompleteDoorstepEPOD,
		SupplierID:     "sup_1",
		IdempotencyKey: "epod-spanner-err-1",
		Parameters: map[string]any{
			"legal_entity_tin":     "TIN-ERR",
			"calendar_date":        "2026-11-13",
			"cash_collected_minor": int64(1000),
		},
	}, nil)
	if !errors.Is(err, foundry.ErrPreconditionFailed) {
		t.Fatalf("expected pre-execution AML guard to fail closed on Spanner error, got rec=%+v err=%v", rec, err)
	}
	if rec == nil || rec.Status != "REJECTED_PRECONDITION" {
		t.Fatalf("expected REJECTED_PRECONDITION record, got %+v", rec)
	}
}
