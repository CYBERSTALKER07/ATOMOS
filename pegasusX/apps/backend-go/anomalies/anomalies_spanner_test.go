package anomalies_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/anomalies"
	"github.com/pegasusx/pegasusx/apps/backend-go/approvals"
	"github.com/pegasusx/pegasusx/apps/backend-go/audit"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
	"github.com/pegasusx/pegasusx/apps/backend-go/proposals"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

func openAnomaliesSpannerClient(t *testing.T) *spanner.Client {
	t.Helper()
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	requireSpanner := strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1"
	if host == "" {
		if requireSpanner {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping live Spanner anomalies test")
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
	_, pErr := ro.ReadRow(ctx, "AnomalyCases", spanner.Key{"__probe__", "__probe__"}, []string{"CaseId"})
	ro.Close()
	if pErr != nil && spanner.ErrCode(pErr) != codes.NotFound {
		client.Close()
		t.Fatalf("AnomalyCases table unavailable: %v", pErr)
	}
	t.Cleanup(client.Close)
	return client
}

type spannerAuditBridge struct {
	svc *audit.Service
}

func (a *spannerAuditBridge) RecordEvent(ctx context.Context, ev anomalies.AuditRecordInput) error {
	_, err := a.svc.Record(ctx, toAuditInput(ev))
	return err
}

func (a *spannerAuditBridge) RecordEventInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, ev anomalies.AuditRecordInput) error {
	_, err := a.svc.RecordInTxn(ctx, txn, toAuditInput(ev))
	return err
}

func toAuditInput(ev anomalies.AuditRecordInput) audit.EventInput {
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

func (p *spannerProposalBridge) CreateProposal(ctx context.Context, in anomalies.ProposalCreateInput) (anomalies.CreatedProposal, error) {
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
		return anomalies.CreatedProposal{}, err
	}
	return anomalies.CreatedProposal{
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

func TestAnomalies_LiveSpanner_All5Detectors_Dedupe_Outbox_Audit_And_Escalation(t *testing.T) {
	client := openAnomaliesSpannerClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	suffix := uuid.NewString()[:8]
	tenantID := "sup-anom-live-" + suffix
	whID := "wh-anom-" + suffix
	catID := "cat-anom-" + suffix
	skuID := "sku-anom-" + suffix
	ret1 := "ret-anom-1-" + suffix
	ret2 := "ret-anom-2-" + suffix
	ordSpike := "ord-anom-spike-" + suffix
	ordBase := "ord-anom-base-" + suffix
	ordGeo := "ord-anom-geo-" + suffix
	cbID := "cb-anom-" + suffix
	dvID := "dv-anom-" + suffix
	brkID := "brk-anom-" + suffix
	ccID := "cc-anom-" + suffix
	sharedTIN := "30888" + suffix[:4]
	uzt := time.FixedZone("UZT", 5*3600)
	now := time.Now().UTC().Truncate(time.Microsecond)

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
			spanner.Delete("OrderShopClosedLog", spanner.KeyRange{Start: spanner.Key{ordGeo}, End: spanner.Key{ordGeo}, Kind: spanner.ClosedClosed}),
			spanner.Delete("Orders", spanner.Key{ordGeo}),
			spanner.Delete("Orders", spanner.Key{ordBase}),
			spanner.Delete("Orders", spanner.Key{ordSpike}),
			spanner.Delete("Retailers", spanner.Key{ret1}),
			spanner.Delete("Retailers", spanner.Key{ret2}),
			spanner.Delete("CashierBlindCountSessions", spanner.Key{cbID}),
			spanner.Delete("DepotVaultLedgerEntries", spanner.Key{dvID}),
			spanner.Delete("ReconciliationBreaks", spanner.Key{tenantID, brkID}),
			spanner.Delete("CycleCounts", spanner.Key{ccID}),
			spanner.Delete("Warehouses", spanner.Key{whID}),
			spanner.Delete("InventoryCostLayers", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("Products", spanner.Key{skuID}),
			spanner.Delete("ProductCategories", spanner.Key{catID}),
			spanner.Delete("JournalPostings", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("JournalEntries", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("GlBalances", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("GlAccounts", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("AccountingPeriods", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("AnomalyCases", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("Proposals", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("ApprovalDecisions", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("ApprovalRequests", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("AuditEvents", spanner.KeyRange{Start: spanner.Key{tenantID}, End: spanner.Key{tenantID}, Kind: spanner.ClosedClosed}),
		})
	})

	// Seed GL + after-hours manual journal entry via live gl.Service.
	periodID := time.Now().UTC().Format("2006-01")
	glSvc := gl.NewService(client)
	if err := glSvc.SeedDefaults(ctx, tenantID, "UZ", "UZS"); err != nil {
		t.Fatalf("gl SeedDefaults: %v", err)
	}
	yestUZT := now.In(uzt).AddDate(0, 0, -1)
	afterHoursUZT := time.Date(yestUZT.Year(), yestUZT.Month(), yestUZT.Day(), 23, 15, 0, 0, uzt).UTC()
	if _, err := glSvc.PostEntry(ctx, gl.JournalEntryInput{
		EntityID:       tenantID,
		PeriodID:       periodID,
		SourceType:     "MANUAL",
		SourceID:       "man-night-" + suffix,
		Memo:           "After hours manual adjustment",
		Currency:       "UZS",
		IdempotencyKey: "anom-man-" + suffix,
		PostedBy:       "acct-night-" + suffix,
		OccurredAt:     afterHoursUZT,
		Postings: []gl.PostingInput{
			{AccountCode: gl.AccountCashOffice, Direction: gl.DirectionDebit, AmountMinor: 1_500_000, Currency: "UZS"},
			{AccountCode: gl.AccountSalesRevenue, Direction: gl.DirectionCredit, AmountMinor: 1_500_000, Currency: "UZS"},
		},
	}); err != nil {
		t.Fatalf("gl PostEntry after hours: %v", err)
	}

	// Seed live domain tables for the other 4 rules.
	lineJSON := fmt.Sprintf(`[{"sku":%q,"name":"Milk 1L","quantity":10,"unit_price_minor":50000}]`, skuID)
	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("ProductCategories", map[string]any{
			"CategoryId": catID, "SupplierId": tenantID, "Name": "Dairy", "SortOrder": int64(1),
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("Products", map[string]any{
			"ProductId": skuID, "SupplierId": tenantID, "CategoryId": catID, "Name": "Milk 1L",
			"PriceMinor": int64(120_000), "Currency": "UZS", "StockQuantity": int64(100),
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("InventoryCostLayers", map[string]any{
			"EntityId": tenantID, "WarehouseId": whID, "ProductId": skuID, "LayerId": "lay-" + suffix,
			"QtyRemaining": int64(100), "OriginalQty": int64(100), "UnitCostMinor": int64(85_000),
			"Currency": "UZS", "ReceivedAt": spanner.CommitTimestamp, "CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("Warehouses", map[string]any{
			"WarehouseId": whID, "SupplierId": tenantID, "Name": "WH Main", "CoverageRadiusKm": 25.0,
			"IsActive": true, "IsOnShift": true, "CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("CycleCounts", map[string]any{
			"CountId": ccID, "WarehouseId": whID, "LocationId": "loc-1", "ProductId": skuID,
			"ExpectedQty": int64(100), "CountedQty": int64(70), "VarianceQty": int64(-30),
			"Status": "COUNTED", "CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("ReconciliationBreaks", map[string]any{
			"EntityId": tenantID, "BreakId": brkID, "RunId": "run-" + suffix, "Dimension": "STOCK_GL",
			"ObjectType": "Warehouse", "ObjectId": whID, "ExpectedMinor": int64(5_000_000),
			"ActualMinor": int64(3_500_000), "DeltaMinor": int64(-1_500_000), "Currency": "UZS",
			"Severity": "HIGH", "Status": "OPEN", "CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("Retailers", map[string]any{
			"RetailerId": ret1, "Phone": "+9989011" + suffix[:6], "Name": "Shop 1", "CountryCode": "UZ",
			"HomeCell": "cell-1-" + suffix, "NormalizedTaxId": sharedTIN, "H3Cell": "882c05111ffffff",
			"CreatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("Retailers", map[string]any{
			"RetailerId": ret2, "Phone": "+9989022" + suffix[:6], "Name": "Shop 2", "CountryCode": "UZ",
			"HomeCell": "cell-2-" + suffix, "NormalizedTaxId": sharedTIN, "H3Cell": "882c05222ffffff",
			"CreatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("Orders", map[string]any{
			"OrderId": ordBase, "SupplierId": tenantID, "RetailerId": ret1, "Status": "CONFIRMED",
			"OrderSource": "DIRECT", "ConfirmationStatus": "CONFIRMED", "LineItemsJson": []byte("[]"),
			"TotalMinor": int64(500_000), "Currency": "UZS", "Version": int64(1),
			"CreatedAt": now.Add(-48 * time.Hour), "UpdatedAt": now.Add(-48 * time.Hour),
		}),
		spanner.InsertMap("Orders", map[string]any{
			"OrderId": ordSpike, "SupplierId": tenantID, "RetailerId": ret2, "Status": "CONFIRMED",
			"OrderSource": "DIRECT", "ConfirmationStatus": "CONFIRMED", "LineItemsJson": []byte(lineJSON),
			"TotalMinor": int64(8_000_000), "OriginalTotalMinor": int64(14_000_000), "Currency": "UZS",
			"Version": int64(1), "CreatedAt": now.Add(-20 * time.Minute), "UpdatedAt": now.Add(-20 * time.Minute),
		}),
		spanner.InsertMap("Orders", map[string]any{
			"OrderId": ordGeo, "SupplierId": tenantID, "RetailerId": ret1, "Status": "SHOP_CLOSED_PENDING",
			"OrderSource": "DIRECT", "ConfirmationStatus": "CONFIRMED", "LineItemsJson": []byte("[]"),
			"TotalMinor": int64(600_000), "Currency": "UZS", "H3Cell": "882c05999ffffff", "Version": int64(1),
			"CreatedAt": now.Add(-30 * time.Minute), "UpdatedAt": now.Add(-30 * time.Minute),
		}),
		spanner.InsertMap("OrderShopClosedLog", map[string]any{
			"OrderId": ordGeo, "EventId": "ev-geo-" + suffix, "Actor": "drv-1", "Action": "MARKED_CLOSED",
			"Payload":   spanner.NullJSON{Value: map[string]any{"geofence_mismatch": true, "distance_m": 600, "shortfall_minor": 400000}, Valid: true},
			"CreatedAt": now.Add(-25 * time.Minute),
		}),
		spanner.InsertMap("CashierBlindCountSessions", map[string]any{
			"SessionId": cbID, "SupplierId": tenantID, "DepotId": "dep-1", "DriverId": "drv-1", "RouteId": "rt-1",
			"BagSealBarcode": "BAG-" + suffix, "DenominationBreakdownJson": "{}", "CountedCashMinor": int64(1_000_000),
			"ExpectedCashMinor": int64(2_200_000), "VarianceMinor": int64(-1_200_000), "Status": "DISCREPANT_QUARANTINE",
			"CreatedAt": now.Add(-15 * time.Minute),
		}),
		spanner.InsertMap("DepotVaultLedgerEntries", map[string]any{
			"VaultEntryId": dvID, "DepotId": "dep-1", "SupplierId": tenantID, "EntryType": "CASH_SHORTFALL",
			"AmountMinor": int64(-350_000), "RunningBalanceMinor": int64(-100_000), "CreatedAt": now.Add(-10 * time.Minute),
		}),
	}); err != nil {
		t.Fatalf("seed Spanner anomaly tables: %v", err)
	}

	auditSvc := audit.NewService(client)
	appSvc := approvals.NewService(client)
	propSvc := proposals.NewService(client)
	propSvc.SetApprovals(appSvc)

	svc := anomalies.NewSpannerService(client, anomalies.ServiceConfig{
		Audit:     &spannerAuditBridge{svc: auditSvc},
		Proposals: &spannerProposalBridge{svc: propSvc},
	})

	scanRes, err := svc.Scan(ctx, anomalies.ScanInput{
		TenantID:  tenantID,
		ActorID:   "owner-" + suffix,
		ActorType: auth.ActorTypeHuman,
		ActorRole: auth.SupplierRoleOwner,
	})
	if err != nil {
		t.Fatalf("live Spanner Scan: %v", err)
	}
	if scanRes.CreatedCount < 5 {
		t.Fatalf("expected >=5 created anomaly cases across all 5 rules, got %d (%+v)", scanRes.CreatedCount, scanRes.Cases)
	}

	seenRules := map[string]bool{}
	var escCaseID string
	for _, c := range scanRes.Cases {
		seenRules[c.RuleCode] = true
		if c.RuleCode == anomalies.RulePricingDiscountAbuse {
			escCaseID = c.CaseID
		}
	}
	for _, want := range []string{
		anomalies.RuleCashGeofenceMismatch,
		anomalies.RuleBasketSpikeSharedINN,
		anomalies.RuleStockCountVariance,
		anomalies.RulePricingDiscountAbuse,
		anomalies.RuleManualJournalAfterHours,
	} {
		if !seenRules[want] {
			t.Fatalf("live Spanner Scan missing rule %s (saw %v)", want, seenRules)
		}
	}

	// Second scan must deduplicate 100% of findings.
	scanRes2, err := svc.Scan(ctx, anomalies.ScanInput{TenantID: tenantID, ActorID: "owner-" + suffix})
	if err != nil {
		t.Fatalf("live Spanner second Scan: %v", err)
	}
	if scanRes2.CreatedCount != 0 || scanRes2.DeduplicatedCount != scanRes.CreatedCount {
		t.Fatalf("expected 0 created and %d deduplicated, got created=%d dedup=%d",
			scanRes.CreatedCount, scanRes2.CreatedCount, scanRes2.DeduplicatedCount)
	}

	// Verify outbox rows on TopicExceptions for anomaly.case.detected.
	ro := client.ReadOnlyTransaction()
	outIter := ro.Query(ctx, spanner.Statement{
		SQL:    `SELECT AggregateType, TopicName, Payload FROM OutboxEvents WHERE SupplierId = @tid`,
		Params: map[string]any{"tid": tenantID},
	})
	var detectedEvents int
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
		if sErr := row.Columns(&agg, &topic, &payload); sErr != nil {
			t.Fatal(sErr)
		}
		if agg == events.AggregateAnomalyCase && topic == events.TopicExceptions && strings.Contains(string(payload), events.EventAnomalyDetected) {
			detectedEvents++
		}
	}
	outIter.Stop()
	ro.Close()
	if detectedEvents != scanRes.CreatedCount {
		t.Fatalf("expected %d anomaly.case.detected outbox events on %s, got %d", scanRes.CreatedCount, events.TopicExceptions, detectedEvents)
	}

	// Escalate pricing case with create_proposal=true -> verify real PENDING Proposal persisted in Spanner.
	escRes, err := svc.ResolveCase(ctx, anomalies.ResolveInput{
		TenantID:       tenantID,
		CaseID:         escCaseID,
		Status:         anomalies.StatusEscalated,
		ResolutionNote: "Escalate below-cost discount anomaly for governed L1 review",
		CreateProposal: true,
		ActorID:        "fin-mgr-" + suffix,
		ActorType:      auth.ActorTypeHuman,
		ActorRole:      auth.SupplierRoleFinanceManager,
	})
	if err != nil {
		t.Fatalf("live ResolveCase(ESCALATED): %v", err)
	}
	if escRes.ProposalID == "" || escRes.Proposal == nil || escRes.Proposal.Status != proposals.StatusPending {
		t.Fatalf("expected PENDING L1 proposal created, got %+v", escRes)
	}
	storedProp, err := propSvc.Get(ctx, tenantID, escRes.ProposalID)
	if err != nil || storedProp.Status != proposals.StatusPending {
		t.Fatalf("stored Proposal lookup failed: err=%v prop=%+v", err, storedProp)
	}

	// Verify live ListCases filter + AuditEvents persisted in Spanner.
	escList, err := svc.ListCases(ctx, anomalies.ListFilter{TenantID: tenantID, Status: anomalies.StatusEscalated})
	if err != nil || len(escList) != 1 || escList[0].ProposalID != escRes.ProposalID {
		t.Fatalf("ListCases(ESCALATED) mismatch: err=%v list=%+v", err, escList)
	}
	audRO := client.ReadOnlyTransaction()
	audIter := audRO.Query(ctx, spanner.Statement{
		SQL:    `SELECT COUNT(*) FROM AuditEvents WHERE TenantId = @tid`,
		Params: map[string]any{"tid": tenantID},
	})
	row, aErr := audIter.Next()
	audIter.Stop()
	audRO.Close()
	if aErr != nil {
		t.Fatalf("query AuditEvents count: %v", aErr)
	}
	var audCount int64
	_ = row.Column(0, &audCount)
	if audCount < 2 {
		t.Fatalf("expected >=2 AuditEvents rows in Spanner for scan+resolve, got %d", audCount)
	}
}
