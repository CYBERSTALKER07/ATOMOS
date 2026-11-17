package whatif_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/approvals"
	"github.com/pegasusx/pegasusx/apps/backend-go/audit"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/proposals"
	"github.com/pegasusx/pegasusx/apps/backend-go/whatif"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

func openWhatIfSpannerClient(t *testing.T) *spanner.Client {
	t.Helper()
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	requireSpanner := strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1"
	if host == "" {
		if requireSpanner {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping live Spanner whatif test")
	}
	envOr := func(k, def string) string {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
		return def
	}
	dsn := fmt.Sprintf("projects/%s/instances/%s/databases/%s",
		envOr("SPANNER_PROJECT", "pegasusx-local"),
		envOr("SPANNER_INSTANCE", "pegasusx-instance"),
		envOr("SPANNER_DATABASE", "pegasusx-db"),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := spanner.NewClient(ctx, dsn,
		option.WithEndpoint(host),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		t.Fatalf("spanner.NewClient: %v", err)
	}
	ro := client.ReadOnlyTransaction()
	_, pErr := ro.ReadRow(ctx, "WhatIfSimulations", spanner.Key{"__probe__", "__probe__"}, []string{"SimulationId"})
	ro.Close()
	if pErr != nil && spanner.ErrCode(pErr) != codes.NotFound {
		client.Close()
		t.Fatalf("WhatIfSimulations table unavailable: %v", pErr)
	}
	t.Cleanup(client.Close)
	return client
}

type spannerAuditBridge struct {
	svc *audit.Service
}

func (a *spannerAuditBridge) RecordEvent(ctx context.Context, ev whatif.AuditRecordInput) error {
	_, err := a.svc.Record(ctx, toAuditInput(ev))
	return err
}

func (a *spannerAuditBridge) RecordEventInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, ev whatif.AuditRecordInput) error {
	_, err := a.svc.RecordInTxn(ctx, txn, toAuditInput(ev))
	return err
}

func toAuditInput(ev whatif.AuditRecordInput) audit.EventInput {
	return audit.EventInput{
		TenantID:   ev.TenantID,
		ActorID:    ev.ActorID,
		ActorType:  audit.NormalizeActorType(ev.ActorType),
		Role:       ev.Role,
		Action:     ev.Action,
		ObjectType: ev.ObjectType,
		ObjectID:   ev.ObjectID,
		BeforeJSON: ev.BeforeJSON,
		AfterJSON:  ev.AfterJSON,
		ReasonCode: ev.ReasonCode,
		ApprovalID: ev.ApprovalID,
		TraceID:    ev.TraceID,
		IP:         ev.IP,
		DeviceID:   ev.DeviceID,
	}
}

type spannerProposalBridge struct {
	svc *proposals.Service
}

func (p *spannerProposalBridge) CreateProposal(ctx context.Context, in whatif.ProposalCreateInput) (whatif.CreatedProposal, error) {
	refs := make([]proposals.EvidenceRef, 0, len(in.EvidenceRefs))
	for _, r := range in.EvidenceRefs {
		refs = append(refs, proposals.EvidenceRef{Type: r.Type, ID: r.ID, Label: r.Label})
	}
	created, err := p.svc.CreateProposal(ctx, proposals.CreateProposalInput{
		TenantID:             in.TenantID,
		AgentID:              in.AgentID,
		ProposalType:         in.ProposalType,
		TargetObjectType:     in.TargetObjectType,
		TargetObjectID:       in.TargetObjectID,
		Title:                in.Title,
		Summary:              in.Summary,
		Rationale:            in.Rationale,
		Payload:              in.Payload,
		EvidenceRefs:         refs,
		ConfidenceBps:        in.ConfidenceBps,
		EstimatedImpactMinor: in.EstimatedImpactMinor,
		Currency:             in.Currency,
		RoiCategory:          in.RoiCategory,
		CreatedBy:            in.CreatedBy,
		ActorType:            in.ActorType,
		IdempotencyKey:       in.IdempotencyKey,
		ExpiresInHours:       in.ExpiresInHours,
	})
	if err != nil {
		return whatif.CreatedProposal{}, err
	}
	return whatif.CreatedProposal{
		TenantID:             created.TenantID,
		ProposalID:           created.ProposalID,
		AgentID:              created.AgentID,
		ProposalType:         created.ProposalType,
		TargetObjectType:     created.TargetObjectType,
		TargetObjectID:       created.TargetObjectID,
		Title:                created.Title,
		Summary:              created.Summary,
		Status:               created.Status,
		ApprovalRequestID:    created.ApprovalRequestID,
		PayloadHash:          created.PayloadHash,
		EstimatedImpactMinor: created.EstimatedImpactMinor,
		ConfidenceBps:        created.ConfidenceBps,
		Currency:             created.Currency,
		CreatedBy:            created.CreatedBy,
	}, nil
}

func TestWhatIf_LiveSpanner_MultiStage_NoDirectDomainMutation_And_CommitProposals(t *testing.T) {
	client := openWhatIfSpannerClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	suffix := uuid.NewString()[:8]
	tenantID := "sup-wi-live-" + suffix
	scenID := "scen-wi-" + suffix
	whID := "wh-wi-" + suffix
	catID := "cat-wi-" + suffix
	skuID := "sku-wi-" + suffix
	retID := "ret-wi-" + suffix
	ordID := "ord-wi-" + suffix
	invID := "inv-wi-" + suffix
	routeID := "rt-wi-" + suffix
	runID := "frun-wi-" + suffix

	t.Cleanup(func() {
		cctx, ccancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer ccancel()
		var outboxMuts []*spanner.Mutation
		ro := client.ReadOnlyTransaction()
		it := ro.Query(cctx, spanner.Statement{
			SQL:    `SELECT EventId FROM OutboxEvents WHERE SupplierId = @tid`,
			Params: map[string]any{"tid": tenantID},
		})
		for {
			r, err := it.Next()
			if err != nil {
				break
			}
			var eid string
			if r.Column(0, &eid) == nil {
				outboxMuts = append(outboxMuts, spanner.Delete("OutboxEvents", spanner.Key{eid}))
			}
		}
		it.Stop()
		ro.Close()
		if len(outboxMuts) > 0 {
			_, _ = client.Apply(cctx, outboxMuts)
		}
		_, _ = client.Apply(cctx, []*spanner.Mutation{
			spanner.Delete("PlanningScenarios", spanner.Key{tenantID, scenID}),
			spanner.Delete("ScenarioObjectPatches", spanner.KeyRange{Start: spanner.Key{scenID}, End: spanner.Key{scenID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("ForecastValues", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("SupplierInventoryV2", spanner.Key{tenantID, whID, skuID}),
			spanner.Delete("InventoryCostLayers", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("Products", spanner.Key{skuID}),
			spanner.Delete("ProductCategories", spanner.Key{catID}),
			spanner.Delete("Orders", spanner.Key{ordID}),
			spanner.Delete("RoutePerformanceAnalytics", spanner.Key{routeID}),
			spanner.Delete("RetailerCreditProfiles", spanner.Key{retID, tenantID}),
			spanner.Delete("ArInvoices", spanner.Key{invID}),
			spanner.Delete("WhatIfSimulations", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("Proposals", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("ApprovalDecisions", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("ApprovalRequests", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("AuditEvents", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}),
		})
	})

	now := time.Now().UTC().Truncate(time.Microsecond)
	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("PlanningScenarios", map[string]any{
			"SupplierId": tenantID, "ScenarioId": scenID, "Version": int64(1), "Status": "DRAFT",
			"Label": "Q4 Shock", "HorizonDays": int64(30), "FactoryDowntimeHours": int64(0),
			"DemandDeltaPct": 0.0, "ResultJSON": "{}", "CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("ScenarioObjectPatches", map[string]any{
			"ScenarioId": scenID, "ObjectType": "Product", "ObjectId": skuID, "SupplierId": tenantID,
			"Operation": "UPSERT", "BaseVersion": int64(1), "PatchJson": `{"unit_price_minor":120000}`,
			"UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("ProductCategories", map[string]any{
			"CategoryId": catID, "SupplierId": tenantID, "Name": "Beverages", "SortOrder": int64(1),
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("Products", map[string]any{
			"ProductId": skuID, "SupplierId": tenantID, "CategoryId": catID, "Name": "Juice 1L",
			"PriceMinor": int64(100_000), "Currency": "UZS", "StockQuantity": int64(900),
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("InventoryCostLayers", map[string]any{
			"EntityId": tenantID, "WarehouseId": whID, "ProductId": skuID, "LayerId": "lay-" + suffix,
			"QtyRemaining": int64(900), "OriginalQty": int64(900), "UnitCostMinor": int64(75_000),
			"Currency": "UZS", "ReceivedAt": spanner.CommitTimestamp, "CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("SupplierInventoryV2", map[string]any{
			"SupplierId": tenantID, "WarehouseId": whID, "ProductId": skuID,
			"QuantityOnHand": int64(900), "QuantityReserved": int64(50), "ReorderThreshold": int64(150),
			"UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("ForecastValues", map[string]any{
			"EntityId": tenantID, "RunId": runID, "ForecastDate": "2026-11-20",
			"LocationId": whID, "Sku": skuID, "ProductId": skuID, "Model": "XGB",
			"IsWinner": true, "P50Milli": int64(1_000_000), "CreatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("Orders", map[string]any{
			"OrderId": ordID, "SupplierId": tenantID, "RetailerId": retID, "WarehouseId": whID,
			"RouteId": routeID, "Status": "DELIVERED", "OrderSource": "DIRECT", "ConfirmationStatus": "CONFIRMED",
			"LineItemsJson": []byte("[]"), "TotalMinor": int64(10_000_000), "DeliveryFeeMinor": int64(1_000_000),
			"Currency": "UZS", "Version": int64(1), "CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("RoutePerformanceAnalytics", map[string]any{
			"RouteId": routeID, "SupplierId": tenantID, "DriverId": "drv-1",
			"PlannedStops": int64(10), "ActualStops": int64(10), "ComputedAt": now,
		}),
		spanner.InsertMap("RetailerCreditProfiles", map[string]any{
			"RetailerId": retID, "SupplierId": tenantID, "CreditLimitMinor": int64(12_000_000),
			"CurrentBalanceMinor": int64(9_000_000), "ReservedMinor": int64(500_000), "RiskScore": int64(12),
			"DelinquencyCount": int64(1), "Status": "ACTIVE",
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("ArInvoices", map[string]any{
			"InvoiceId": invID, "SupplierId": tenantID, "RetailerId": retID, "OrderId": ordID,
			"Status": "OPEN", "PrincipalMinor": int64(9_000_000), "BalanceMinor": int64(9_000_000),
			"Currency": "UZS", "CreditLeaveAt": now, "DueAt": now.Add(30 * 24 * time.Hour),
			"TermsDays": int64(30), "CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
	}); err != nil {
		t.Fatalf("seed live Spanner What-If tables: %v", err)
	}

	auditSvc := audit.NewService(client)
	appSvc := approvals.NewService(client)
	propSvc := proposals.NewService(client)
	propSvc.SetApprovals(appSvc)

	svc := whatif.NewSpannerService(client, whatif.ServiceConfig{
		Audit:     &spannerAuditBridge{svc: auditSvc},
		Proposals: &spannerProposalBridge{svc: propSvc},
	})

	sim, replayed, err := svc.Simulate(ctx, whatif.SimulateInput{
		TenantID:             tenantID,
		Name:                 "Live E2E Winter Shock",
		ScenarioID:           scenID,
		HorizonDays:          30,
		DemandDeltaBps:       2500,
		FactoryDowntimeHours: 36,
		FxRateShockBps:       600,
		CreditLimitDeltaBps:  1000,
		UnitPriceDeltaBps:    500,
		IdempotencyKey:       "wi-live-" + suffix,
		ActorID:              "owner-" + suffix,
		ActorType:            auth.ActorTypeHuman,
		ActorRole:            auth.SupplierRoleOwner,
	})
	if err != nil {
		t.Fatalf("live Simulate: %v", err)
	}
	if replayed || sim.ForecastDeltaMinor <= 0 || sim.ReplenishmentCostMinor <= 0 || sim.DispatchCostDeltaMinor <= 0 || sim.CreditExposureDeltaMinor <= 0 {
		t.Fatalf("unexpected live simulation result: replayed=%v sim=%+v", replayed, sim)
	}

	// Verify live domain tables were NEVER mutated by simulation (§13).
	ro := client.ReadOnlyTransaction()
	invRow, err := ro.ReadRow(ctx, "SupplierInventoryV2", spanner.Key{tenantID, whID, skuID}, []string{"QuantityOnHand", "QuantityReserved"})
	if err != nil {
		ro.Close()
		t.Fatalf("read SupplierInventoryV2: %v", err)
	}
	var qoh, qres int64
	_ = invRow.Columns(&qoh, &qres)
	if qoh != 900 || qres != 50 {
		ro.Close()
		t.Fatalf("What-If mutated SupplierInventoryV2 directly! got qoh=%d qres=%d", qoh, qres)
	}
	cpRow, err := ro.ReadRow(ctx, "RetailerCreditProfiles", spanner.Key{retID, tenantID}, []string{"CreditLimitMinor"})
	if err != nil {
		ro.Close()
		t.Fatalf("read RetailerCreditProfiles: %v", err)
	}
	var clim int64
	_ = cpRow.Columns(&clim)
	if clim != 12_000_000 {
		ro.Close()
		t.Fatalf("What-If mutated RetailerCreditProfiles directly! got limit=%d", clim)
	}

	// Verify whatif.simulation.completed outbox event emitted on TopicIntel.
	outIter := ro.Query(ctx, spanner.Statement{
		SQL:    `SELECT AggregateType, TopicName, Payload FROM OutboxEvents WHERE SupplierId = @tid`,
		Params: map[string]any{"tid": tenantID},
	})
	var simEvents int
	for {
		row, qErr := outIter.Next()
		if qErr == iterator.Done {
			break
		}
		if qErr != nil {
			outIter.Stop()
			ro.Close()
			t.Fatalf("query OutboxEvents: %v", qErr)
		}
		var agg, topic string
		var payload []byte
		_ = row.Columns(&agg, &topic, &payload)
		if agg == events.AggregateWhatIfSimulation && topic == events.TopicIntel && strings.Contains(string(payload), events.EventWhatIfSimulated) {
			simEvents++
		}
	}
	outIter.Stop()
	ro.Close()
	if simEvents != 1 {
		t.Fatalf("expected 1 whatif.simulation.completed event on %s, got %d", events.TopicIntel, simEvents)
	}

	// Commit simulation proposals -> creates real PENDING Proposals in Spanner (never self-approves!).
	commitRes, err := svc.CommitProposals(ctx, whatif.CommitProposalsInput{
		TenantID:       tenantID,
		SimulationID:   sim.SimulationID,
		ActorID:        "fin-mgr-" + suffix,
		ActorType:      auth.ActorTypeHuman,
		ActorRole:      auth.SupplierRoleFinanceManager,
		IdempotencyKey: "commit-live-" + suffix,
	})
	if err != nil {
		t.Fatalf("live CommitProposals: %v", err)
	}
	if commitRes.Simulation.Status != whatif.StatusCommitted || len(commitRes.CommittedProposalIDs) < 3 {
		t.Fatalf("expected COMMITTED simulation with >=3 proposals, got %+v", commitRes)
	}
	for _, pid := range commitRes.CommittedProposalIDs {
		stored, gErr := propSvc.Get(ctx, tenantID, pid)
		if gErr != nil || stored.Status != proposals.StatusPending {
			t.Fatalf("committed proposal %s not PENDING in Spanner: err=%v status=%s", pid, gErr, stored.Status)
		}
	}

	// Verify ListSimulations, GetSimulation, and AuditEvents in Spanner.
	sims, err := svc.ListSimulations(ctx, whatif.ListFilter{TenantID: tenantID, Status: whatif.StatusCommitted})
	if err != nil || len(sims) != 1 || sims[0].SimulationID != sim.SimulationID {
		t.Fatalf("ListSimulations mismatch: err=%v sims=%+v", err, sims)
	}
	gotSim, err := svc.GetSimulation(ctx, tenantID, sim.SimulationID)
	if err != nil || gotSim.Status != whatif.StatusCommitted {
		t.Fatalf("GetSimulation mismatch: err=%v sim=%+v", err, gotSim)
	}
	audRO := client.ReadOnlyTransaction()
	audIter := audRO.Query(ctx, spanner.Statement{
		SQL:    `SELECT COUNT(*) FROM AuditEvents WHERE TenantId = @tid`,
		Params: map[string]any{"tid": tenantID},
	})
	audRow, aErr := audIter.Next()
	audIter.Stop()
	audRO.Close()
	if aErr != nil {
		t.Fatalf("query AuditEvents count: %v", aErr)
	}
	var audCount int64
	_ = audRow.Column(0, &audCount)
	if audCount < 2 {
		t.Fatalf("expected >=2 AuditEvents rows in Spanner for simulate+commit, got %d", audCount)
	}
}
