package whatif_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/whatif"
)

type stubAuditRecorder struct {
	mu     sync.Mutex
	events []whatif.AuditRecordInput
}

func (s *stubAuditRecorder) RecordEvent(_ context.Context, ev whatif.AuditRecordInput) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, ev)
	return nil
}

func (s *stubAuditRecorder) RecordEventInTxn(_ context.Context, _ *spanner.ReadWriteTransaction, ev whatif.AuditRecordInput) error {
	return s.RecordEvent(context.Background(), ev)
}

type stubProposalCreator struct {
	mu      sync.Mutex
	created []whatif.ProposalCreateInput
}

func (s *stubProposalCreator) CreateProposal(_ context.Context, in whatif.ProposalCreateInput) (whatif.CreatedProposal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.created = append(s.created, in)
	return whatif.CreatedProposal{
		TenantID:             in.TenantID,
		ProposalID:           "prop_wi_" + in.ProposalType,
		AgentID:              in.AgentID,
		ProposalType:         in.ProposalType,
		TargetObjectType:     in.TargetObjectType,
		TargetObjectID:       in.TargetObjectID,
		Title:                in.Title,
		Summary:              in.Summary,
		Status:               "PENDING",
		ApprovalRequestID:    "apr_wi_1",
		EstimatedImpactMinor: in.EstimatedImpactMinor,
		ConfidenceBps:        in.ConfidenceBps,
		Currency:             "UZS",
		CreatedBy:            in.CreatedBy,
	}, nil
}

func TestWhatIf_MultiStagePipeline_Idempotency_And_CommitProposals(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fixedNow := time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)

	repo := whatif.NewMemoryRepository()
	auditRec := &stubAuditRecorder{}
	propCreator := &stubProposalCreator{}
	svc := whatif.NewService(repo, whatif.ServiceConfig{
		Audit:     auditRec,
		Proposals: propCreator,
		Now:       func() time.Time { return fixedNow },
	})

	const tid = "sup-whatif-unit-1"
	repo.SeedDataset(tid, whatif.SimulationDataset{
		TenantID: tid,
		Scenario: &whatif.ScenarioRecord{
			ScenarioID:           "scen-1",
			Status:               "DRAFT",
			HorizonDays:          30,
			FactoryDowntimeHours: 0,
		},
		ScenarioPatches: []whatif.ScenarioPatchRecord{
			{
				ScenarioID: "scen-1",
				ObjectType: "Product",
				ObjectID:   "SKU-MILK",
				Operation:  "UPSERT",
				PatchJSON:  `{"unit_price_minor":120000,"unit_cost_minor":80000}`,
			},
		},
		Forecasts: []whatif.ForecastRecord{
			{SKU: "SKU-MILK", ProductID: "SKU-MILK", LocationID: "wh-1", IsWinner: true, P50Milli: 1_000_000}, // 1000 units
		},
		Inventory: []whatif.InventoryRecord{
			{WarehouseID: "wh-1", ProductID: "SKU-MILK", QuantityOnHand: 950, QuantityReserved: 50, ReorderThreshold: 100},
		},
		Products: []whatif.ProductPriceRecord{
			{ProductID: "SKU-MILK", Name: "Milk 1L", PriceMinor: 100_000, Currency: "UZS"},
		},
		CostLayers: []whatif.CostLayerRecord{
			{ProductID: "SKU-MILK", UnitCostMinor: 70_000},
		},
		Orders: []whatif.OrderDispatchRecord{
			{OrderID: "ord-1", WarehouseID: "wh-1", RouteID: "rt-1", Status: "DELIVERED", TotalMinor: 12_000_000, DeliveryFeeMinor: 1_000_000},
		},
		Routes: []whatif.RouteRecord{
			{RouteID: "rt-1", ReplanCount: 0},
		},
		CreditProfiles: []whatif.CreditProfileRecord{
			{RetailerID: "ret-1", CreditLimitMinor: 10_000_000, CurrentBalanceMinor: 8_000_000, ReservedMinor: 500_000, RiskScore: 10, DelinquencyCount: 1, Status: "ACTIVE"},
		},
		ArInvoices: []whatif.ArInvoiceRecord{
			{InvoiceID: "inv-1", RetailerID: "ret-1", BalanceMinor: 8_500_000, Status: "OPEN"},
		},
	})

	// Validation: empty name or out-of-range bps rejected.
	if _, _, err := svc.Simulate(ctx, whatif.SimulateInput{TenantID: tid, Name: ""}); !errors.Is(err, whatif.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for empty name, got %v", err)
	}
	if _, _, err := svc.Simulate(ctx, whatif.SimulateInput{TenantID: tid, Name: "bad", DemandDeltaBps: -20000}); !errors.Is(err, whatif.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for out-of-range demand_delta_bps, got %v", err)
	}

	simIn := whatif.SimulateInput{
		TenantID:             tid,
		Name:                 "Ramadan +20% Shock",
		ScenarioID:           "scen-1",
		HorizonDays:          30,
		DemandDeltaBps:       2000, // +20%
		FactoryDowntimeHours: 24,   // 24h downtime
		FxRateShockBps:       500,  // +5% FX
		CreditLimitDeltaBps:  1000, // +10% credit limit
		UnitPriceDeltaBps:    200,  // +2% price
		IdempotencyKey:       "idem-sim-1",
		ActorID:              "owner-1",
		ActorType:            auth.ActorTypeHuman,
		ActorRole:            auth.SupplierRoleOwner,
	}

	sim, replayed, err := svc.Simulate(ctx, simIn)
	if err != nil {
		t.Fatalf("Simulate: %v", err)
	}
	if replayed {
		t.Fatal("expected replayed=false on first simulation")
	}
	if sim.Status != whatif.StatusSimulated || sim.ForecastDeltaMinor <= 0 || sim.ReplenishmentCostMinor <= 0 || sim.DispatchCostDeltaMinor <= 0 || sim.CreditExposureDeltaMinor <= 0 {
		t.Fatalf("expected positive stage deltas across pipeline, got %+v", sim)
	}
	if sim.Breakdown == nil || !sim.Breakdown.ScenarioApplied || sim.Breakdown.PatchCount != 1 || len(sim.Breakdown.RecommendedActions) < 3 {
		t.Fatalf("expected scenario overlay applied and >=3 recommended actions, got %+v", sim.Breakdown)
	}

	// Idempotent replay with same IdempotencyKey.
	simReplay, replayed2, err := svc.Simulate(ctx, simIn)
	if err != nil || !replayed2 || simReplay.SimulationID != sim.SimulationID {
		t.Fatalf("expected idempotent replay, got err=%v replayed=%v simID=%s", err, replayed2, simReplay.SimulationID)
	}

	// AUDITOR and AGENT cannot commit proposals.
	if _, err := svc.CommitProposals(ctx, whatif.CommitProposalsInput{
		TenantID:     tid,
		SimulationID: sim.SimulationID,
		ActorID:      "aud-1",
		ActorRole:    auth.SupplierRoleAuditor,
	}); !errors.Is(err, whatif.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for AUDITOR commit, got %v", err)
	}

	// Commit proposals -> creates L1 PENDING Proposals via ProposalCreator, NEVER self-approves!
	commitRes, err := svc.CommitProposals(ctx, whatif.CommitProposalsInput{
		TenantID:       tid,
		SimulationID:   sim.SimulationID,
		ActorID:        "fin-mgr-1",
		ActorType:      auth.ActorTypeHuman,
		ActorRole:      auth.SupplierRoleFinanceManager,
		IdempotencyKey: "commit-key-1",
	})
	if err != nil {
		t.Fatalf("CommitProposals: %v", err)
	}
	if commitRes.Simulation.Status != whatif.StatusCommitted || len(commitRes.CommittedProposalIDs) < 3 || len(commitRes.Proposals) != len(commitRes.CommittedProposalIDs) {
		t.Fatalf("unexpected CommitProposals result: %+v", commitRes)
	}
	for _, p := range commitRes.Proposals {
		if p.Status != "PENDING" {
			t.Fatalf("committed What-If proposal must be PENDING for human approval, got %s", p.Status)
		}
	}

	// Idempotent commit replay succeeds; different key returns ErrConflict.
	if _, err := svc.CommitProposals(ctx, whatif.CommitProposalsInput{
		TenantID:       tid,
		SimulationID:   sim.SimulationID,
		ActorID:        "fin-mgr-1",
		IdempotencyKey: "commit-key-1",
	}); err != nil {
		t.Fatalf("expected idempotent commit replay, got %v", err)
	}
	if _, err := svc.CommitProposals(ctx, whatif.CommitProposalsInput{
		TenantID:       tid,
		SimulationID:   sim.SimulationID,
		ActorID:        "fin-mgr-1",
		IdempotencyKey: "commit-key-other",
	}); !errors.Is(err, whatif.ErrConflict) {
		t.Fatalf("expected ErrConflict on duplicate commit, got %v", err)
	}

	if len(auditRec.events) < 2 {
		t.Fatalf("expected >=2 audit events (simulate + commit), got %d", len(auditRec.events))
	}
}
