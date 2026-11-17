package anomalies_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/anomalies"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

type stubAuditRecorder struct {
	mu     sync.Mutex
	events []anomalies.AuditRecordInput
}

func (s *stubAuditRecorder) RecordEvent(_ context.Context, ev anomalies.AuditRecordInput) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, ev)
	return nil
}

func (s *stubAuditRecorder) RecordEventInTxn(_ context.Context, _ *spanner.ReadWriteTransaction, ev anomalies.AuditRecordInput) error {
	return s.RecordEvent(context.Background(), ev)
}

type stubProposalCreator struct {
	mu      sync.Mutex
	created []anomalies.ProposalCreateInput
}

func (s *stubProposalCreator) CreateProposal(_ context.Context, in anomalies.ProposalCreateInput) (anomalies.CreatedProposal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.created = append(s.created, in)
	return anomalies.CreatedProposal{
		TenantID:             in.TenantID,
		ProposalID:           "prop_anom_" + in.TargetObjectID,
		AgentID:              in.AgentID,
		ProposalType:         in.ProposalType,
		TargetObjectType:     in.TargetObjectType,
		TargetObjectID:       in.TargetObjectID,
		Title:                in.Title,
		Summary:              in.Summary,
		Status:               "PENDING",
		ApprovalRequestID:    "apr_anom_1",
		EstimatedImpactMinor: in.EstimatedImpactMinor,
		ConfidenceBps:        in.ConfidenceBps,
		Currency:             "UZS",
		CreatedBy:            in.CreatedBy,
	}, nil
}

func TestAnomalies_AllFiveRules_Dedupe_Filter_And_EscalateToProposal(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	uzt := time.FixedZone("UZT", 5*3600)
	baseNow := time.Date(2026, 11, 17, 10, 0, 0, 0, time.UTC)

	repo := anomalies.NewMemoryRepository()
	auditRec := &stubAuditRecorder{}
	propCreator := &stubProposalCreator{}
	svc := anomalies.NewService(repo, anomalies.ServiceConfig{
		Audit:     auditRec,
		Proposals: propCreator,
		Now:       func() time.Time { return baseNow },
	})

	const tid = "sup-anom-unit-1"
	cat := svc.Catalogue()
	if len(cat) != 5 {
		t.Fatalf("expected 5 §7.7 detector rules in catalogue, got %d", len(cat))
	}

	// Empty tenant scan produces zero fake/synthetic cases.
	emptyRes, err := svc.Scan(ctx, anomalies.ScanInput{TenantID: "sup-empty-none", ActorID: "owner-1"})
	if err != nil {
		t.Fatalf("empty Scan: %v", err)
	}
	if emptyRes.CreatedCount != 0 || len(emptyRes.Cases) != 0 {
		t.Fatalf("expected 0 cases on unseeded tenant, got %+v", emptyRes)
	}

	// Seed deterministic dataset triggering all 5 rules + negative controls.
	repo.SeedDataset(tid, anomalies.DetectorDataset{
		TenantID: tid,
		CashierSessions: []anomalies.CashierBlindCountRecord{
			{
				SessionID:         "cb-short-1",
				SupplierID:        tid,
				DepotID:           "dep-1",
				DriverID:          "drv-1",
				RouteID:           "rt-1",
				CountedCashMinor:  1_200_000,
				ExpectedCashMinor: 2_500_000,
				VarianceMinor:     -1_300_000,
				Status:            "DISCREPANT_QUARANTINE",
				CreatedAt:         baseNow.Add(-2 * time.Hour),
			},
			{
				// Clean session within tolerance (must NOT trigger).
				SessionID:         "cb-clean-1",
				SupplierID:        tid,
				DepotID:           "dep-1",
				DriverID:          "drv-2",
				RouteID:           "rt-2",
				CountedCashMinor:  1_000_000,
				ExpectedCashMinor: 1_000_000,
				VarianceMinor:     0,
				Status:            "BALANCED",
				CreatedAt:         baseNow.Add(-1 * time.Hour),
			},
		},
		DepotVaultEntries: []anomalies.DepotVaultLedgerRecord{
			{
				VaultEntryID:        "dv-1",
				DepotID:             "dep-1",
				SupplierID:          tid,
				EntryType:           "DRIVER_DROP_INFLOW",
				AmountMinor:         1_000_000,
				RunningBalanceMinor: 1_000_000,
				CreatedAt:           baseNow.Add(-3 * time.Hour),
			},
			{
				VaultEntryID:        "dv-2",
				DepotID:             "dep-1",
				SupplierID:          tid,
				EntryType:           "CIT_ARMORED_OUTFLOW",
				AmountMinor:         500_000,
				RunningBalanceMinor: 500_000, // clean: 1_000_000 - 500_000 == 500_000
				CreatedAt:           baseNow.Add(-2 * time.Hour),
			},
		},
		ShopClosedLogs: []anomalies.ShopClosedLogRecord{
			{
				OrderID:         "ord-geo-1",
				EventID:         "ev-geo-1",
				Actor:           "drv-1",
				Action:          "MARKED_CLOSED",
				PayloadJSON:     `{"geofence_mismatch":true,"distance_m":850,"shortfall_minor":750000}`,
				OrderTotalMinor: 950_000,
				OrderH3Cell:     "882c05123ffffff",
				RetailerID:      "ret-1",
				RetailerH3Cell:  "882c05999ffffff",
				CreatedAt:       baseNow.Add(-90 * time.Minute),
			},
		},
		Orders: []anomalies.OrderRecord{
			{
				OrderID:         "ord-norm-1",
				SupplierID:      tid,
				RetailerID:      "ret-a",
				NormalizedTaxID: "305001122",
				Status:          "CONFIRMED",
				TotalMinor:      1_000_000,
				Currency:        "UZS",
				LineItemsJSON:   `[{"sku":"SKU-1","quantity":10,"unit_price_minor":100000}]`,
				CreatedAt:       baseNow.Add(-48 * time.Hour),
			},
			{
				OrderID:         "ord-norm-2",
				SupplierID:      tid,
				RetailerID:      "ret-a",
				NormalizedTaxID: "305001122",
				Status:          "CONFIRMED",
				TotalMinor:      1_000_000,
				Currency:        "UZS",
				LineItemsJSON:   `[{"sku":"SKU-1","quantity":10,"unit_price_minor":100000}]`,
				CreatedAt:       baseNow.Add(-24 * time.Hour),
			},
			{
				// Triggers BASKET_SPIKE_SHARED_INN (shared TIN w/ ret-b + 8x basket spike)
				// AND PRICING_DISCOUNT_ABUSE (below cost 60_000 < 90_000 unit cost).
				OrderID:            "ord-spike-belowcost",
				SupplierID:         tid,
				RetailerID:         "ret-b",
				NormalizedTaxID:    "305001122",
				Status:             "CONFIRMED",
				TotalMinor:         8_000_000,
				OriginalTotalMinor: 16_000_000,
				Currency:           "UZS",
				LineItemsJSON:      `[{"sku":"SKU-1","quantity":100,"unit_price_minor":60000,"list_price_minor":120000}]`,
				CreatedAt:          baseNow.Add(-30 * time.Minute),
			},
		},
		ReconBreaks: []anomalies.ReconBreakRecord{
			{
				BreakID:       "brk-stock-1",
				RunID:         "run-1",
				Dimension:     "STOCK_GL",
				ObjectType:    "StockLot",
				ObjectID:      "lot-99",
				ExpectedMinor: 5_000_000,
				ActualMinor:   3_500_000,
				DeltaMinor:    -1_500_000,
				Currency:      "UZS",
				Severity:      "HIGH",
				Status:        "OPEN",
				CreatedAt:     baseNow.Add(-45 * time.Minute),
			},
		},
		CycleCounts: []anomalies.CycleCountRecord{
			{
				CountID:     "cc-var-1",
				WarehouseID: "wh-1",
				LocationID:  "loc-1",
				ProductID:   "SKU-1",
				ExpectedQty: 100,
				CountedQty:  75,
				VarianceQty: -25,
				HasCounted:  true,
				Status:      "COUNTED",
				CreatedAt:   baseNow.Add(-40 * time.Minute),
			},
		},
		CostLayers: []anomalies.CostLayerSummary{
			{ProductID: "SKU-1", UnitCostMinor: 90_000},
		},
		ProductPrices: []anomalies.ProductPriceRecord{
			{ProductID: "SKU-1", SKU: "SKU-1", UnitPriceMinor: 120_000},
		},
		ManualJournals: []anomalies.ManualJournalRecord{
			{
				// After-hours 02:30 UZT
				EntityID:        tid,
				EntryID:         "je-afterhours",
				PeriodID:        "2026-11",
				SourceType:      "MANUAL_JOURNAL",
				PostedBy:        "acct-1",
				DebitTotalMinor: 4_000_000,
				OccurredAt:      time.Date(2026, 11, 17, 2, 30, 0, 0, uzt),
			},
			{
				// Daytime 11:00 UZT + 15:00 UZT same-day threshold splitting (6M + 6M >= 10M)
				EntityID:        tid,
				EntryID:         "je-split-1",
				PeriodID:        "2026-11",
				SourceType:      "MANUAL",
				PostedBy:        "acct-split",
				DebitTotalMinor: 6_000_000,
				OccurredAt:      time.Date(2026, 11, 17, 11, 0, 0, 0, uzt),
			},
			{
				EntityID:        tid,
				EntryID:         "je-split-2",
				PeriodID:        "2026-11",
				SourceType:      "MANUAL",
				PostedBy:        "acct-split",
				DebitTotalMinor: 6_000_000,
				OccurredAt:      time.Date(2026, 11, 17, 15, 0, 0, 0, uzt),
			},
			{
				// Clean daytime non-split entry (must NOT trigger)
				EntityID:        tid,
				EntryID:         "je-clean-day",
				PeriodID:        "2026-11",
				SourceType:      "MANUAL",
				PostedBy:        "acct-clean",
				DebitTotalMinor: 3_000_000,
				OccurredAt:      time.Date(2026, 11, 17, 14, 0, 0, 0, uzt),
			},
		},
	})

	scanRes, err := svc.Scan(ctx, anomalies.ScanInput{
		TenantID:  tid,
		ActorID:   "owner-1",
		ActorType: auth.ActorTypeHuman,
		ActorRole: auth.SupplierRoleOwner,
	})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if scanRes.CreatedCount < 5 || len(scanRes.ScannedRules) != 5 {
		t.Fatalf("expected all 5 rules and >=5 created cases, got %+v", scanRes)
	}

	rulesHit := map[string]bool{}
	objectHit := map[string]bool{}
	var targetCaseID string
	for _, c := range scanRes.Cases {
		rulesHit[c.RuleCode] = true
		objectHit[c.ObjectID] = true
		if c.RuleCode == anomalies.RulePricingDiscountAbuse {
			targetCaseID = c.CaseID
		}
		if c.Status != anomalies.StatusOpen {
			t.Fatalf("expected OPEN status on new case, got %s", c.Status)
		}
	}
	for _, wantRule := range []string{
		anomalies.RuleCashGeofenceMismatch,
		anomalies.RuleBasketSpikeSharedINN,
		anomalies.RuleStockCountVariance,
		anomalies.RulePricingDiscountAbuse,
		anomalies.RuleManualJournalAfterHours,
	} {
		if !rulesHit[wantRule] {
			t.Fatalf("missing finding for rule %s; got rules=%v", wantRule, rulesHit)
		}
	}
	if objectHit["cb-clean-1"] || objectHit["dv-2"] || objectHit["je-clean-day"] {
		t.Fatalf("clean control records should not be flagged: %v", objectHit)
	}
	if !objectHit["je-afterhours"] || !objectHit["je-split-1"] || !objectHit["je-split-2"] {
		t.Fatalf("expected after-hours and split manual journals flagged, got %v", objectHit)
	}

	// Idempotent second scan deduplicates all previously created findings.
	scanRes2, err := svc.Scan(ctx, anomalies.ScanInput{TenantID: tid, ActorID: "owner-1"})
	if err != nil {
		t.Fatalf("second Scan: %v", err)
	}
	if scanRes2.CreatedCount != 0 || scanRes2.DeduplicatedCount != scanRes.CreatedCount {
		t.Fatalf("expected full deduplication on second scan, got created=%d deduped=%d (first=%d)",
			scanRes2.CreatedCount, scanRes2.DeduplicatedCount, scanRes.CreatedCount)
	}

	// Filter ListCases by category & rule code.
	pricingCases, err := svc.ListCases(ctx, anomalies.ListFilter{
		TenantID: tid,
		Category: anomalies.CategoryPricingFraud,
		Status:   anomalies.StatusOpen,
	})
	if err != nil || len(pricingCases) != 1 || pricingCases[0].CaseID != targetCaseID {
		t.Fatalf("ListCases filter mismatch: err=%v got=%+v", err, pricingCases)
	}

	// Escalate with create_proposal=true creates L1 Proposal via ProposalCreator.
	escRes, err := svc.ResolveCase(ctx, anomalies.ResolveInput{
		TenantID:       tid,
		CaseID:         targetCaseID,
		Status:         anomalies.StatusEscalated,
		ResolutionNote: "Verified below-cost sale in order lines; escalate for credit/price audit",
		CreateProposal: true,
		ActorID:        "fin-mgr-1",
		ActorType:      auth.ActorTypeHuman,
		ActorRole:      auth.SupplierRoleFinanceManager,
	})
	if err != nil {
		t.Fatalf("ResolveCase(ESCALATED): %v", err)
	}
	if escRes.Case.Status != anomalies.StatusEscalated || escRes.Proposal == nil || escRes.ProposalID == "" {
		t.Fatalf("expected escalated case with created proposal, got %+v", escRes)
	}
	if escRes.Proposal.Status != "PENDING" {
		t.Fatalf("escalation proposal must remain PENDING for human approval, got %s", escRes.Proposal.Status)
	}

	// ESCALATED -> RESOLVED succeeds; subsequent RESOLVED -> DISMISSED fails with ErrConflict.
	resFinal, err := svc.ResolveCase(ctx, anomalies.ResolveInput{
		TenantID:       tid,
		CaseID:         targetCaseID,
		Status:         anomalies.StatusResolved,
		ResolutionNote: "Margin adjustment posted",
		ActorID:        "owner-1",
	})
	if err != nil || resFinal.Case.Status != anomalies.StatusResolved {
		t.Fatalf("ResolveCase(RESOLVED): err=%v res=%+v", err, resFinal)
	}
	if _, err := svc.ResolveCase(ctx, anomalies.ResolveInput{
		TenantID:       tid,
		CaseID:         targetCaseID,
		Status:         anomalies.StatusDismissed,
		ResolutionNote: "Should conflict",
		ActorID:        "owner-1",
	}); !errors.Is(err, anomalies.ErrConflict) {
		t.Fatalf("expected ErrConflict on terminal case re-resolve, got %v", err)
	}

	// AUDITOR and AGENT must be rejected on ResolveCase.
	if _, err := svc.ResolveCase(ctx, anomalies.ResolveInput{
		TenantID:       tid,
		CaseID:         scanRes.Cases[0].CaseID,
		Status:         anomalies.StatusDismissed,
		ResolutionNote: "auditor attempt",
		ActorID:        "aud-1",
		ActorRole:      auth.SupplierRoleAuditor,
	}); !errors.Is(err, anomalies.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for AUDITOR, got %v", err)
	}

	if len(auditRec.events) < 3 {
		t.Fatalf("expected AuditEvents recorded across scan + resolve, got %d", len(auditRec.events))
	}
	if !strings.Contains(strings.ToUpper(escRes.Proposal.Title), "PRICING_DISCOUNT_ABUSE") {
		t.Fatalf("unexpected proposal title: %s", escRes.Proposal.Title)
	}
}
