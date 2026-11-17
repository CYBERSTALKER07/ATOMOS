package proposals_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/approvals"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/proposals"
	"github.com/pegasusx/pegasusx/apps/backend-go/roi"
	"google.golang.org/api/iterator"
)

type spyPropAudit struct {
	events []proposals.AuditRecordInput
}

func (s *spyPropAudit) RecordEvent(_ context.Context, ev proposals.AuditRecordInput) error {
	s.events = append(s.events, ev)
	return nil
}

func seedSoakHistory(repo *proposals.MemoryRepository, tid string, now time.Time, executedCount, rollbackCount int, soakDays int64) {
	earliest := now.Add(-time.Duration(soakDays) * 24 * time.Hour)
	for i := 0; i < executedCount; i++ {
		ts := earliest.Add(time.Duration(i) * time.Hour)
		repo.SeedHistoricalProposal(proposals.Proposal{
			TenantID:             tid,
			ProposalID:           fmt.Sprintf("hist-exec-%d", i),
			AgentID:              proposals.AgentReorder,
			ProposalType:         proposals.TypeReorder,
			TargetObjectType:     "Product",
			TargetObjectID:       fmt.Sprintf("sku-hist-%d", i),
			Title:                "Historical reorder",
			Rationale:            "History",
			PayloadJSON:          "{}",
			PayloadHash:          "abc",
			Status:               proposals.StatusExecuted,
			AutonomyLevel:        proposals.AutonomyLevelL1,
			EstimatedImpactMinor: 50_000,
			CreatedBy:            "agent:reorder_agent",
			ActorType:            auth.ActorTypeAgent,
			ExecutedAt:           &ts,
			CreatedAt:            ts,
			UpdatedAt:            ts,
		})
	}
	for i := 0; i < rollbackCount; i++ {
		ts := earliest.Add(time.Duration(i+1) * time.Hour)
		repo.SeedHistoricalProposal(proposals.Proposal{
			TenantID:         tid,
			ProposalID:       fmt.Sprintf("hist-rb-%d", i),
			AgentID:          proposals.AgentReorder,
			ProposalType:     proposals.TypeReorder,
			TargetObjectType: "Product",
			TargetObjectID:   fmt.Sprintf("sku-rb-%d", i),
			Title:            "Rolled back reorder",
			Rationale:        "History",
			PayloadJSON:      "{}",
			PayloadHash:      "abc",
			Status:           "ROLLED_BACK",
			CreatedBy:        "agent:reorder_agent",
			ActorType:        auth.ActorTypeAgent,
			ExecutedAt:       &ts,
			CreatedAt:        ts,
			UpdatedAt:        ts,
		})
	}
}

func TestL2Policy_SoakGating_HighRiskHardBlock_And_AutoExecute(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	const tid = "sup-l2-1"

	propRepo := proposals.NewMemoryRepository()
	appSvc := approvals.NewServiceWithRepository(approvals.NewMemoryRepository())
	roiSvc := roi.NewServiceWithRepository(roi.NewMemoryRepository())
	auditSpy := &spyPropAudit{}

	svc := proposals.NewServiceWithRepository(propRepo)
	svc.SetApprovals(appSvc)
	svc.SetROI(roiSvc)
	svc.SetAuditRecorder(auditSpy)

	// 1. Seed insufficient soak (only 5 executed over 3 days) + 1 low-risk REORDER proposal.
	seedSoakHistory(propRepo, tid, now, 5, 0, 3)

	lowReorder, err := svc.CreateProposal(ctx, proposals.CreateProposalInput{
		TenantID:             tid,
		AgentID:              proposals.AgentReorder,
		ProposalType:         proposals.TypeReorder,
		TargetObjectType:     "Product",
		TargetObjectID:       "sku-low-1",
		Title:                "Low-value reorder",
		Rationale:            "Low stock",
		Payload:              map[string]any{"action": "approve_insight", "product_id": "sku-low-1", "suggested_qty": 10, "unit_price_minor": 10000},
		ConfidenceBps:        8500,
		EstimatedImpactMinor: 100_000,
		Currency:             "UZS",
		CreatedBy:            "agent:reorder_agent",
		ActorType:            auth.ActorTypeAgent,
	})
	if err != nil {
		t.Fatalf("CreateProposal lowReorder: %v", err)
	}

	// Evaluate before soak passes -> eligible=false, auto_executed_count=0 even with AutoExecute=true!
	resFail, err := svc.EvaluateL2(ctx, proposals.L2EvaluateInput{
		TenantID:    tid,
		AutoExecute: true,
		CallerID:    "owner-1",
		ActorType:   auth.ActorTypeHuman,
		CallerRole:  auth.SupplierRoleOwner,
	})
	if err != nil {
		t.Fatalf("EvaluateL2 (soak fail): %v", err)
	}
	if resFail.Eligible || resFail.AutoExecutedCount != 0 || len(resFail.Blockers) < 2 {
		t.Fatalf("expected ineligible soak with >=2 blockers, got %+v", resFail)
	}

	// 2. Seed rollback rate > 500 bps -> still blocked!
	seedSoakHistory(propRepo, tid, now, 10, 2, 16) // 2 / 12 = 1666 bps > 500 bps
	resRB, err := svc.EvaluateL2(ctx, proposals.L2EvaluateInput{
		TenantID:    tid,
		AutoExecute: true,
		CallerID:    "owner-1",
		ActorType:   auth.ActorTypeHuman,
		CallerRole:  auth.SupplierRoleOwner,
	})
	if err != nil {
		t.Fatalf("EvaluateL2 (rollback fail): %v", err)
	}
	if resRB.Eligible || resRB.RollbackRateBps <= 500 || resRB.AutoExecutedCount != 0 {
		t.Fatalf("expected high rollback rate to block L2, got %+v", resRB)
	}

	// 3. Clean tenant with passing soak (15 days, 10 executed, 0 rollbacks).
	const tidPass = "sup-l2-pass"
	propRepo2 := proposals.NewMemoryRepository()
	svc2 := proposals.NewServiceWithRepository(propRepo2)
	svc2.SetApprovals(approvals.NewServiceWithRepository(approvals.NewMemoryRepository()))
	svc2.SetROI(roi.NewServiceWithRepository(roi.NewMemoryRepository()))
	auditSpy2 := &spyPropAudit{}
	svc2.SetAuditRecorder(auditSpy2)
	seedSoakHistory(propRepo2, tidPass, now, 10, 0, 15)

	// Add 1 eligible low-risk REORDER, 1 over-ceiling REORDER, and 4 hard-blocked high-risk actions.
	okProp, err := svc2.CreateProposal(ctx, proposals.CreateProposalInput{
		TenantID:             tidPass,
		AgentID:              proposals.AgentReorder,
		ProposalType:         proposals.TypeReorder,
		TargetObjectType:     "Product",
		TargetObjectID:       "sku-l2-ok",
		Title:                "L2 eligible reorder",
		Rationale:            "Safety stock",
		Payload:              map[string]any{"action": "open_supply_request", "product_id": "sku-l2-ok", "suggested_qty": 5, "unit_price_minor": 20000},
		ConfidenceBps:        9000,
		EstimatedImpactMinor: 100_000,
		CreatedBy:            "agent:reorder_agent",
		ActorType:            auth.ActorTypeAgent,
	})
	if err != nil {
		t.Fatalf("CreateProposal okProp: %v", err)
	}
	overProp, err := svc2.CreateProposal(ctx, proposals.CreateProposalInput{
		TenantID:             tidPass,
		AgentID:              proposals.AgentReorder,
		ProposalType:         proposals.TypeReorder,
		TargetObjectType:     "Product",
		TargetObjectID:       "sku-l2-over",
		Title:                "Over-ceiling reorder",
		Rationale:            "Large order",
		Payload:              map[string]any{"action": "open_supply_request", "product_id": "sku-l2-over", "suggested_qty": 500, "unit_price_minor": 100_000},
		ConfidenceBps:        9000,
		EstimatedImpactMinor: 5_000_000,
		CreatedBy:            "agent:reorder_agent",
		ActorType:            auth.ActorTypeAgent,
	})
	if err != nil {
		t.Fatalf("CreateProposal overProp: %v", err)
	}
	for _, hr := range []string{"CREDIT_LIMIT", "WRITE_OFF", "PAYOUT", "PERIOD_CLOSE"} {
		if _, err := svc2.CreateProposal(ctx, proposals.CreateProposalInput{
			TenantID:             tidPass,
			AgentID:              proposals.AgentReorder,
			ProposalType:         hr,
			TargetObjectType:     hr,
			TargetObjectID:       "obj-" + hr,
			Title:                "Blocked high risk " + hr,
			Rationale:            "Must never auto-exec",
			Payload:              map[string]any{"action": hr, "amount_minor": 1000},
			EstimatedImpactMinor: 1000,
			CreatedBy:            "agent:reorder_agent",
			ActorType:            auth.ActorTypeAgent,
		}); err != nil {
			t.Fatalf("CreateProposal %s: %v", hr, err)
		}
	}

	// AUDITOR and AGENT callers blocked with 403 sentinels.
	if _, err := svc2.EvaluateL2(ctx, proposals.L2EvaluateInput{
		TenantID: tidPass, AutoExecute: true, CallerID: "aud-1", ActorType: auth.ActorTypeHuman, CallerRole: auth.SupplierRoleAuditor,
	}); !errors.Is(err, proposals.ErrAuditorReadOnly) {
		t.Fatalf("expected ErrAuditorReadOnly, got %v", err)
	}
	if _, err := svc2.EvaluateL2(ctx, proposals.L2EvaluateInput{
		TenantID: tidPass, AutoExecute: true, CallerID: "agt-1", ActorType: auth.ActorTypeAgent, CallerRole: auth.SupplierRoleAgent,
	}); !errors.Is(err, proposals.ErrAgentApprovalForbidden) {
		t.Fatalf("expected ErrAgentApprovalForbidden, got %v", err)
	}

	resPass, err := svc2.EvaluateL2(ctx, proposals.L2EvaluateInput{
		TenantID:         tidPass,
		AutoExecute:      true,
		MaxL2AmountMinor: 500_000,
		CallerID:         "owner-1",
		ActorType:        auth.ActorTypeHuman,
		CallerRole:       auth.SupplierRoleOwner,
	})
	if err != nil {
		t.Fatalf("EvaluateL2 pass: %v", err)
	}
	if !resPass.Eligible || resPass.AutoExecutedCount != 1 || len(resPass.AutoExecutedProposalIDs) != 1 || resPass.AutoExecutedProposalIDs[0] != okProp.ProposalID {
		t.Fatalf("expected exactly 1 L2 auto-executed proposal %s, got %+v", okProp.ProposalID, resPass)
	}
	execP := resPass.ExecutedProposals[0]
	if execP.AutonomyLevel != proposals.AutonomyLevelL2 || execP.Status != proposals.StatusExecuted || execP.RoiEventID == "" {
		t.Fatalf("expected AutonomyLevel=L2, Status=EXECUTED, RoiEventID set; got %+v", execP)
	}
	if len(resPass.BlockedProposals) != 5 {
		t.Fatalf("expected 5 blocked proposals (1 over-ceiling + 4 high-risk), got %d: %+v", len(resPass.BlockedProposals), resPass.BlockedProposals)
	}
	// Over-ceiling proposal stays PENDING.
	if stillOver, err := svc2.Get(ctx, tidPass, overProp.ProposalID); err != nil || stillOver.Status != proposals.StatusPending {
		t.Fatalf("expected overProp still PENDING, got %+v err=%v", stillOver, err)
	}
	// Audit event recorded.
	if len(auditSpy2.events) != 1 || auditSpy2.events[0].Action != "proposal.l2.auto_executed" {
		t.Fatalf("expected proposal.l2.auto_executed audit event, got %+v", auditSpy2.events)
	}
	_ = lowReorder
}

func TestL2Policy_LiveSpanner_SoakAutoExecAndOutbox(t *testing.T) {
	client := openProposalsSpannerClient(t)
	ctx := context.Background()
	tid := "sup-l2-sp-" + uuid.NewString()[:8]
	now := time.Now().UTC()
	earliest := now.Add(-15 * 24 * time.Hour)

	appSvc := approvals.NewService(client)
	roiSvc := roi.NewService(client)
	svc := proposals.NewService(client)
	svc.SetApprovals(appSvc)
	svc.SetROI(roiSvc)
	svc.SetDomainExecutor(&proposals.SpannerDomainExecutor{Client: client})

	wh1 := "wh-l2-" + uuid.NewString()[:8]
	fac1 := "fac-l2-" + uuid.NewString()[:8]
	prod1 := "sku-l2-" + uuid.NewString()[:8]
	ins1 := "ins-l2-" + uuid.NewString()[:8]

	t.Cleanup(func() {
		cleanCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		muts := []*spanner.Mutation{
			spanner.Delete("ReplenishmentInsights", spanner.Key{ins1}),
			spanner.Delete("Products", spanner.Key{prod1}),
			spanner.Delete("Warehouses", spanner.Key{wh1}),
			spanner.Delete("Proposals", spanner.KeyRange{Start: spanner.Key{tid}, End: spanner.Key{tid}, Kind: spanner.ClosedClosed}),
			spanner.Delete("ApprovalRequests", spanner.KeyRange{Start: spanner.Key{tid}, End: spanner.Key{tid}, Kind: spanner.ClosedClosed}),
			spanner.Delete("ApprovalDecisions", spanner.KeyRange{Start: spanner.Key{tid}, End: spanner.Key{tid}, Kind: spanner.ClosedClosed}),
			spanner.Delete("RoiEvents", spanner.KeyRange{Start: spanner.Key{tid}, End: spanner.Key{tid}, Kind: spanner.ClosedClosed}),
		}
		ro := client.ReadOnlyTransaction()
		wsIter := ro.Query(cleanCtx, spanner.Statement{
			SQL:    `SELECT RequestId FROM WarehouseSupplyRequests WHERE SupplierId = @tid`,
			Params: map[string]any{"tid": tid},
		})
		for {
			row, err := wsIter.Next()
			if err != nil {
				break
			}
			var rid string
			if row.Columns(&rid) == nil {
				muts = append(muts, spanner.Delete("WarehouseSupplyRequests", spanner.Key{rid}))
			}
		}
		wsIter.Stop()
		oIter := ro.Query(cleanCtx, spanner.Statement{
			SQL:    `SELECT EventId FROM OutboxEvents WHERE SupplierId = @tid`,
			Params: map[string]any{"tid": tid},
		})
		for {
			row, err := oIter.Next()
			if err != nil {
				break
			}
			var eid string
			if row.Columns(&eid) == nil {
				muts = append(muts, spanner.Delete("OutboxEvents", spanner.Key{eid}))
			}
		}
		oIter.Stop()
		ro.Close()
		_, _ = client.Apply(cleanCtx, muts)
	})

	// Seed Warehouse, Product, ReplenishmentInsight + 10 historical EXECUTED L1 proposals starting 15 days ago in live Spanner.
	seedMuts := []*spanner.Mutation{
		spanner.InsertMap("Warehouses", map[string]any{
			"WarehouseId": wh1, "SupplierId": tid, "Name": "L2 Main WH", "CoverageRadiusKm": float64(50),
			"PrimaryFactoryId": fac1, "IsActive": true, "IsOnShift": true,
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("Products", map[string]any{
			"ProductId": prod1, "SupplierId": tid, "CategoryId": "cat-1",
			"Name": "Cheese 250g", "PriceMinor": int64(15000), "Currency": "UZS",
			"StockQuantity": int64(4), "IsActive": true, "Version": int64(1),
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("ReplenishmentInsights", map[string]any{
			"InsightId": ins1, "WarehouseId": wh1, "ProductId": prod1, "SupplierId": tid,
			"CurrentStock": int64(4), "SuggestedQuantity": int64(8), "UrgencyLevel": "HIGH",
			"ReasonCode": "LOW_STOCK", "Status": "PENDING", "TargetFactoryId": fac1,
			"CreatedAt": spanner.CommitTimestamp,
		}),
	}
	for i := 0; i < 10; i++ {
		ts := earliest.Add(time.Duration(i) * time.Hour)
		seedMuts = append(seedMuts, spanner.InsertMap("Proposals", map[string]any{
			"TenantId":             tid,
			"ProposalId":           fmt.Sprintf("hist-sp-%d-%s", i, uuid.NewString()[:6]),
			"AgentId":              proposals.AgentReorder,
			"ProposalType":         proposals.TypeReorder,
			"TargetObjectType":     "Product",
			"TargetObjectId":       fmt.Sprintf("sku-sp-h-%d", i),
			"Title":                "Historical L1 reorder",
			"Rationale":            "Soak seed",
			"PayloadJson":          "{}",
			"PayloadHash":          "abc",
			"EvidenceRefsJson":     "[]",
			"ConfidenceBps":        int64(8500),
			"EstimatedImpactMinor": int64(50000),
			"Currency":             "UZS",
			"AutonomyLevel":        proposals.AutonomyLevelL1,
			"Status":               proposals.StatusExecuted,
			"CreatedBy":            "agent:reorder_agent",
			"ActorType":            auth.ActorTypeAgent,
			"ExecutedBy":           "owner-sp",
			"ExecutedAt":           ts,
			"CreatedAt":            ts,
			"UpdatedAt":            ts,
		}))
	}
	if _, err := client.Apply(ctx, seedMuts); err != nil {
		t.Fatalf("seed historical Spanner proposals: %v", err)
	}

	reorders, err := svc.RunReorderAgent(ctx, tid)
	if err != nil || len(reorders) != 1 {
		t.Fatalf("RunReorderAgent: len=%d err=%v", len(reorders), err)
	}
	okProp := reorders[0]
	creditProp, err := svc.CreateProposal(ctx, proposals.CreateProposalInput{
		TenantID:             tid,
		AgentID:              proposals.AgentReorder,
		ProposalType:         "CREDIT_LIMIT",
		TargetObjectType:     "CreditLimit",
		TargetObjectID:       "ret-sp-1",
		Title:                "High risk credit limit increase",
		Rationale:            "Must block at L2",
		Payload:              map[string]any{"action": "CREDIT_LIMIT", "amount_minor": 5000},
		EstimatedImpactMinor: 5000,
		CreatedBy:            "agent:reorder_agent",
		ActorType:            auth.ActorTypeAgent,
	})
	if err != nil {
		t.Fatalf("CreateProposal creditProp: %v", err)
	}

	res, err := svc.EvaluateL2(ctx, proposals.L2EvaluateInput{
		TenantID:         tid,
		AutoExecute:      true,
		MaxL2AmountMinor: 500_000,
		CallerID:         "owner-sp-1",
		ActorType:        auth.ActorTypeHuman,
		CallerRole:       auth.SupplierRoleOwner,
	})
	if err != nil {
		t.Fatalf("EvaluateL2 Spanner: %v", err)
	}
	if !res.Eligible || res.SoakDays < 14 || res.ExecutedCount < 10 || res.AutoExecutedCount != 1 {
		t.Fatalf("unexpected Spanner L2 evaluation: %+v", res)
	}

	got, err := svc.Get(ctx, tid, okProp.ProposalID)
	if err != nil {
		t.Fatalf("Get executed L2 proposal: %v", err)
	}
	if got.AutonomyLevel != proposals.AutonomyLevelL2 || got.Status != proposals.StatusExecuted || got.RoiEventID == "" {
		t.Fatalf("expected AutonomyLevel=L2, Status=EXECUTED, RoiEventID set; got %+v", got)
	}
	gotCredit, err := svc.Get(ctx, tid, creditProp.ProposalID)
	if err != nil || gotCredit.Status != proposals.StatusPending {
		t.Fatalf("expected high-risk creditProp to stay PENDING, got %+v err=%v", gotCredit, err)
	}

	// Verify proposal.l2.auto_executed outbox event in live Spanner!
	ro := client.ReadOnlyTransaction()
	defer ro.Close()
	oIter := ro.Query(ctx, spanner.Statement{
		SQL:    `SELECT Payload FROM OutboxEvents WHERE SupplierId = @tid`,
		Params: map[string]any{"tid": tid},
	})
	defer oIter.Stop()
	foundL2Event := false
	for {
		row, err := oIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			t.Fatalf("query OutboxEvents: %v", err)
		}
		var raw []byte
		if err := row.Columns(&raw); err != nil {
			t.Fatalf("scan OutboxEvents payload: %v", err)
		}
		if strings.Contains(string(raw), `"proposal.l2.auto_executed"`) && strings.Contains(string(raw), `"autonomy_level":"L2"`) {
			foundL2Event = true
			break
		}
	}
	if !foundL2Event {
		t.Fatalf("expected proposal.l2.auto_executed outbox event in live Spanner")
	}
}
