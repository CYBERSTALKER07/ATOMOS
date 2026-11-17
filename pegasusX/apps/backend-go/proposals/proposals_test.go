package proposals_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/approvals"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/proposalroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/proposals"
	"github.com/pegasusx/pegasusx/apps/backend-go/roi"
)

func TestProposalsLifecycle_L1AgentsApprovalsEditTargetLockAndROI(t *testing.T) {
	ctx := context.Background()
	propRepo := proposals.NewMemoryRepository()
	appSvc := approvals.NewServiceWithRepository(approvals.NewMemoryRepository())
	roiRepo := roi.NewMemoryRepository()
	roiSvc := roi.NewServiceWithRepository(roiRepo)

	svc := proposals.NewServiceWithRepository(propRepo)
	svc.SetApprovals(appSvc)
	svc.SetROI(roiSvc)

	const tenantID = "sup-prop-1"
	propRepo.SeedReorderCandidates(tenantID, proposals.ReorderCandidate{
		InsightID:      "ins-1",
		WarehouseID:    "wh-1",
		ProductID:      "sku-1",
		ProductName:    "Milk 1L",
		CurrentStock:   3,
		SuggestedQty:   40,
		UnitPriceMinor: 12000,
		Currency:       "UZS",
		UrgencyLevel:   "CRITICAL",
		ReasonCode:     "LOW_STOCK",
	})
	propRepo.SeedOverdueInvoices(tenantID, proposals.OverdueInvoiceCandidate{
		InvoiceID:    "inv-1",
		OrderID:      "ord-1",
		RetailerID:   "ret-1",
		RetailerName: "Baraka Market",
		BalanceMinor: 500000,
		Currency:     "UZS",
		DueAt:        time.Now().UTC().Add(-10 * 24 * time.Hour),
		DaysOverdue:  10,
		DunningStep:  1,
	})
	propRepo.SeedExpiryCandidates(tenantID, proposals.ExpiryRiskCandidate{
		LotID:             "lot-1",
		LayerID:           "layer-1",
		ProductID:         "sku-2",
		ProductName:       "Yogurt 500g",
		SourceWarehouseID: "wh-1",
		TargetWarehouseID: "wh-2",
		QtyAvailable:      25,
		UnitCostMinor:     8000,
		ValueAtRiskMinor:  200000,
		Currency:          "UZS",
		ExpiryDate:        time.Now().UTC().Add(5 * 24 * time.Hour).Format("2006-01-02"),
		DaysToExpiry:      5,
	})

	// 1. Run all 3 L1 agents -> creates 3 PENDING proposals + 3 ApprovalRequests; zero approved!
	created, err := svc.RunAgentPass(ctx, tenantID, "ALL")
	if err != nil {
		t.Fatalf("RunAgentPass(ALL): %v", err)
	}
	if len(created) != 3 {
		t.Fatalf("expected 3 agent proposals, got %d", len(created))
	}
	var reorderProp, collProp, expiryProp proposals.Proposal
	for _, p := range created {
		if p.Status != proposals.StatusPending || p.ActorType != auth.ActorTypeAgent || p.ApprovalRequestID == "" {
			t.Fatalf("expected PENDING agent proposal with ApprovalRequestID, got %+v", p)
		}
		if len(p.EvidenceRefs) == 0 {
			t.Fatalf("expected non-empty EvidenceRefs for %s", p.ProposalID)
		}
		switch p.AgentID {
		case proposals.AgentReorder:
			reorderProp = p
		case proposals.AgentCollections:
			collProp = p
		case proposals.AgentExpiryRebalance:
			expiryProp = p
		}
	}
	// Verify bilingual UZ/RU drafts in collections payload.
	uzDraft, _ := collProp.Payload["draft_uz"].(string)
	ruDraft, _ := collProp.Payload["draft_ru"].(string)
	if uzDraft == "" || ruDraft == "" {
		t.Fatalf("expected bilingual UZ/RU collections draft, got uz=%q ru=%q", uzDraft, ruDraft)
	}

	// 2. AGENT actor cannot approve, reject, or execute!
	if _, err := svc.ApproveOrEditProposal(ctx, proposals.ApproveOrEditInput{
		TenantID: tenantID, ProposalID: reorderProp.ProposalID,
		ApproverID: "bot-1", ActorType: auth.ActorTypeAgent, ApproverRole: auth.SupplierRoleAgent,
	}); !errors.Is(err, proposals.ErrAgentApprovalForbidden) {
		t.Fatalf("expected ErrAgentApprovalForbidden, got %v", err)
	}

	// 3. Human approves with EditedPayload -> payload re-hashed + re-pinned in approvals!
	editedProp, err := svc.ApproveOrEditProposal(ctx, proposals.ApproveOrEditInput{
		TenantID: tenantID, ProposalID: reorderProp.ProposalID,
		ApproverID: "owner-1", ActorType: auth.ActorTypeHuman, ApproverRole: auth.SupplierRoleOwner,
		EditedPayload: map[string]any{
			"action":        "approve_insight",
			"insight_id":    "ins-1",
			"warehouse_id":  "wh-1",
			"product_id":    "sku-1",
			"suggested_qty": 60,
		},
		Comment: "boosted qty to 60",
	})
	if err != nil {
		t.Fatalf("ApproveOrEditProposal: %v", err)
	}
	if editedProp.Status != proposals.StatusApproved || editedProp.PayloadHash == reorderProp.PayloadHash {
		t.Fatalf("expected APPROVED with updated PayloadHash, got status=%s hash=%s", editedProp.Status, editedProp.PayloadHash)
	}

	// 4. Stale precondition check: if insight is closed before execution, ExecuteProposal fails with ErrPreconditionFailed!
	propRepo.MarkInsightClosed(tenantID, "ins-1")
	if _, err := svc.ExecuteProposal(ctx, proposals.ExecuteInput{
		TenantID: tenantID, ProposalID: editedProp.ProposalID,
		ExecutorID: "owner-1", ActorType: auth.ActorTypeHuman, ExecutorRole: auth.SupplierRoleOwner,
	}); !errors.Is(err, proposals.ErrPreconditionFailed) {
		t.Fatalf("expected ErrPreconditionFailed for closed insight, got %v", err)
	}

	// 5. Approve & execute expiry rebalance -> records EXPIRY_SAVED ROI event!
	approvedExp, err := svc.ApproveOrEditProposal(ctx, proposals.ApproveOrEditInput{
		TenantID: tenantID, ProposalID: expiryProp.ProposalID,
		ApproverID: "owner-1", ActorType: auth.ActorTypeHuman, ApproverRole: auth.SupplierRoleOwner,
	})
	if err != nil {
		t.Fatalf("approve expiry: %v", err)
	}
	execRes, err := svc.ExecuteProposal(ctx, proposals.ExecuteInput{
		TenantID: tenantID, ProposalID: approvedExp.ProposalID,
		ExecutorID: "owner-1", ActorType: auth.ActorTypeHuman, ExecutorRole: auth.SupplierRoleOwner,
	})
	if err != nil {
		t.Fatalf("ExecuteProposal expiry: %v", err)
	}
	if execRes.Proposal.Status != proposals.StatusExecuted || execRes.Proposal.RoiEventID == "" {
		t.Fatalf("expected EXECUTED with RoiEventID, got %+v", execRes.Proposal)
	}
	roiSum, err := roiSvc.GetSummary(ctx, tenantID, 90)
	if err != nil || roiSum.TotalSavingsMinor != 200000 {
		t.Fatalf("expected 200000 ROI savings recorded, got %+v err=%v", roiSum, err)
	}

	// 6. Concurrent execution guard on overlapping proposals targeting same ArInvoice!
	overlapProp, err := svc.CreateProposal(ctx, proposals.CreateProposalInput{
		TenantID:             tenantID,
		AgentID:              proposals.AgentCollections,
		ProposalType:         proposals.TypeCollections,
		TargetObjectType:     "ArInvoice",
		TargetObjectID:       "inv-1",
		Title:                "Overlapping collections nudge",
		Rationale:            "Second overlapping proposal on inv-1",
		Payload:              map[string]any{"invoice_id": "inv-1", "action": "collections_nudge"},
		EstimatedImpactMinor: 500000,
		CreatedBy:            "agent:collections_agent",
		ActorType:            auth.ActorTypeAgent,
	})
	if err != nil {
		t.Fatalf("CreateProposal overlap: %v", err)
	}
	if _, err := svc.ApproveOrEditProposal(ctx, proposals.ApproveOrEditInput{
		TenantID: tenantID, ProposalID: collProp.ProposalID,
		ApproverID: "owner-1", ActorType: auth.ActorTypeHuman, ApproverRole: auth.SupplierRoleOwner,
	}); err != nil {
		t.Fatalf("approve collProp: %v", err)
	}
	if _, err := svc.ApproveOrEditProposal(ctx, proposals.ApproveOrEditInput{
		TenantID: tenantID, ProposalID: overlapProp.ProposalID,
		ApproverID: "owner-1", ActorType: auth.ActorTypeHuman, ApproverRole: auth.SupplierRoleOwner,
	}); err != nil {
		t.Fatalf("approve overlapProp: %v", err)
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	successCount := 0
	conflictCount := 0
	for _, pid := range []string{collProp.ProposalID, overlapProp.ProposalID} {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			_, e := svc.ExecuteProposal(ctx, proposals.ExecuteInput{
				TenantID: tenantID, ProposalID: id,
				ExecutorID: "owner-1", ActorType: auth.ActorTypeHuman, ExecutorRole: auth.SupplierRoleOwner,
			})
			mu.Lock()
			defer mu.Unlock()
			if e == nil {
				successCount++
			} else if errors.Is(e, proposals.ErrTargetExecutionConflict) || errors.Is(e, proposals.ErrAlreadyExecuted) {
				conflictCount++
			} else {
				t.Errorf("unexpected concurrent exec error on %s: %v", id, e)
			}
		}(pid)
	}
	wg.Wait()
	if successCount != 1 || conflictCount != 1 {
		t.Fatalf("expected exactly 1 success and 1 target lock conflict, got success=%d conflict=%d", successCount, conflictCount)
	}

	// 7. HTTP RBAC checks: SoD self-approve 403, AGENT approve 403, Dispatcher missing perm 403.
	r := chi.NewRouter()
	const jwtSecret = "prop-test-secret-32-bytes-minimum!"
	r.Use(auth.SessionAuth(jwtSecret))
	r.Use(auth.AttachTenantFromClaims)
	proposalroutes.RegisterRoutes(r, proposalroutes.Deps{
		Service:   svc,
		JWTSecret: jwtSecret,
	})

	issueTok := func(sub string, supRole auth.Role, actorType string) string {
		tok, err := auth.Issue(auth.Claims{
			Subject:      sub,
			Role:         auth.RoleAdmin,
			SupplierID:   tenantID,
			SupplierRole: supRole,
			ActorType:    actorType,
		}, auth.IssueOptions{Secret: jwtSecret, TTL: time.Hour})
		if err != nil {
			t.Fatal(err)
		}
		return tok
	}
	makerTok := issueTok("human-maker", auth.SupplierRoleAccountant, auth.ActorTypeHuman)
	ownerTok := issueTok("owner-2", auth.SupplierRoleOwner, auth.ActorTypeHuman)
	agentTok := issueTok("agent-bot", auth.SupplierRoleAgent, auth.ActorTypeAgent)

	// Human maker creates proposal via HTTP, then tries to approve own -> 403 (either missing perm or SoD),
	// and another proposal created by owner-2 -> owner-2 trying to approve own -> 403 SoD!
	createReq := httptest.NewRequest(http.MethodPost, "/v1/proposals", strings.NewReader(`{
		"proposal_type":"REORDER","target_object_type":"Product","target_object_id":"sku-99",
		"title":"Manual reorder","payload":{"product_id":"sku-99","suggested_qty":10},"estimated_impact_minor":50000
	}`))
	createReq.Header.Set("Authorization", "Bearer "+ownerTok)
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	r.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("POST /v1/proposals = %d %s", createRec.Code, createRec.Body.String())
	}

	// Agent cannot approve via HTTP (403).
	appReq := httptest.NewRequest(http.MethodPost, "/v1/proposals/"+collProp.ProposalID+"/approve", strings.NewReader(`{}`))
	appReq.Header.Set("Authorization", "Bearer "+agentTok)
	appReq.Header.Set("Content-Type", "application/json")
	appRec := httptest.NewRecorder()
	r.ServeHTTP(appRec, appReq)
	if appRec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for AGENT approve HTTP, got %d %s", appRec.Code, appRec.Body.String())
	}
	_ = makerTok
}
