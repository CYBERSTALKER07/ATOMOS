package proposalroutes_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/approvals"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/proposalroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/proposals"
	"github.com/pegasusx/pegasusx/apps/backend-go/roi"
)

func TestProposalRoutes_EndToEndMountAndGovernance(t *testing.T) {
	t.Parallel()

	const jwtSecret = "proposal-routes-test-secret-32bytes!"
	propRepo := proposals.NewMemoryRepository()
	propRepo.SeedExpiryCandidates("sup-pr-rt", proposals.ExpiryRiskCandidate{
		LotID: "lot-rt-1", ProductID: "sku-rt-1", ProductName: "Cheese",
		SourceWarehouseID: "wh-1", TargetWarehouseID: "wh-2",
		QtyAvailable: 12, UnitCostMinor: 10000, ValueAtRiskMinor: 120000, Currency: "UZS",
		ExpiryDate: time.Now().UTC().Add(4 * 24 * time.Hour).Format("2006-01-02"), DaysToExpiry: 4,
	})

	roiSvc := roi.NewServiceWithRepository(roi.NewMemoryRepository())
	svc := proposals.NewServiceWithRepository(propRepo)
	svc.SetApprovals(approvals.NewServiceWithRepository(approvals.NewMemoryRepository()))
	svc.SetROI(roiSvc)

	r := chi.NewRouter()
	r.Use(auth.SessionAuth(jwtSecret))
	r.Use(auth.AttachTenantFromClaims)
	proposalroutes.RegisterRoutes(r, proposalroutes.Deps{
		Service:   svc,
		JWTSecret: jwtSecret,
	})

	issue := func(sub, supRole, actorType string) string {
		tok, err := auth.Issue(auth.Claims{
			Subject:      sub,
			Role:         auth.RoleAdmin,
			SupplierID:   "sup-pr-rt",
			SupplierRole: auth.Role(supRole),
			ActorType:    actorType,
		}, auth.IssueOptions{Secret: jwtSecret, TTL: time.Hour})
		if err != nil {
			t.Fatal(err)
		}
		return tok
	}
	ownerTok := issue("owner-1", auth.SupplierRoleOwner, auth.ActorTypeHuman)
	agentTok := issue("agent-1", auth.SupplierRoleAgent, auth.ActorTypeAgent)
	auditorTok := issue("aud-1", auth.SupplierRoleAuditor, auth.ActorTypeHuman)

	call := func(method, path string, body any, tok string) *httptest.ResponseRecorder {
		var buf bytes.Buffer
		if body != nil {
			_ = json.NewEncoder(&buf).Encode(body)
		}
		req := httptest.NewRequest(method, path, &buf)
		if tok != "" {
			req.Header.Set("Authorization", "Bearer "+tok)
		}
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec
	}

	if rec := call(http.MethodGet, "/v1/proposals", nil, ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated GET /v1/proposals = %d, want 401", rec.Code)
	}
	if rec := call(http.MethodPost, "/v1/proposals/agents/EXPIRY_REBALANCE_AGENT/run", nil, auditorTok); rec.Code != http.StatusForbidden {
		t.Fatalf("auditor POST agent run = %d, want 403", rec.Code)
	}

	runRec := call(http.MethodPost, "/v1/proposals/agents/EXPIRY_REBALANCE_AGENT/run", nil, ownerTok)
	if runRec.Code != http.StatusOK {
		t.Fatalf("POST agent run = %d body=%s", runRec.Code, runRec.Body.String())
	}
	var runResp struct {
		CreatedCount int                  `json:"created_count"`
		Proposals    []proposals.Proposal `json:"proposals"`
	}
	_ = json.Unmarshal(runRec.Body.Bytes(), &runResp)
	if runResp.CreatedCount != 1 || len(runResp.Proposals) != 1 {
		t.Fatalf("expected 1 proposal, got %+v", runResp)
	}
	pid := runResp.Proposals[0].ProposalID

	// Agent cannot approve (403)
	if rec := call(http.MethodPost, "/v1/proposals/"+pid+"/approve", map[string]any{}, agentTok); rec.Code != http.StatusForbidden {
		t.Fatalf("agent approve = %d, want 403", rec.Code)
	}

	// Owner approves & executes
	if rec := call(http.MethodPost, "/v1/proposals/"+pid+"/approve", map[string]any{"comment": "ok"}, ownerTok); rec.Code != http.StatusOK {
		t.Fatalf("owner approve = %d body=%s", rec.Code, rec.Body.String())
	}
	if rec := call(http.MethodPost, "/v1/proposals/"+pid+"/execute", map[string]any{}, ownerTok); rec.Code != http.StatusOK {
		t.Fatalf("owner execute = %d body=%s", rec.Code, rec.Body.String())
	}
	// Replay execute -> 409
	if rec := call(http.MethodPost, "/v1/proposals/"+pid+"/execute", map[string]any{}, ownerTok); rec.Code != http.StatusConflict {
		t.Fatalf("replay execute = %d, want 409", rec.Code)
	}

	// Create + reject flow
	cRec := call(http.MethodPost, "/v1/proposals", map[string]any{
		"proposal_type": "REORDER", "target_object_type": "Product", "target_object_id": "sku-2",
		"title": "Manual reorder", "payload": map[string]any{"sku": "sku-2", "qty": 5}, "estimated_impact_minor": 10000,
	}, agentTok)
	if cRec.Code != http.StatusCreated {
		t.Fatalf("agent create proposal = %d body=%s", cRec.Code, cRec.Body.String())
	}
	var cResp struct {
		Proposal proposals.Proposal `json:"proposal"`
	}
	_ = json.Unmarshal(cRec.Body.Bytes(), &cResp)
	if rec := call(http.MethodGet, "/v1/proposals/"+cResp.Proposal.ProposalID, nil, auditorTok); rec.Code != http.StatusOK {
		t.Fatalf("auditor GET /v1/proposals/{id} = %d", rec.Code)
	}
	if rec := call(http.MethodPost, "/v1/proposals/"+cResp.Proposal.ProposalID+"/reject", map[string]any{"comment": "not needed"}, ownerTok); rec.Code != http.StatusOK {
		t.Fatalf("owner reject = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestProposalRoutes_L2Evaluate_RBAC_Expiry_And_SoakGate(t *testing.T) {
	t.Parallel()

	const jwtSecret = "proposal-routes-l2-test-secret-32b!"
	const tid = "sup-pr-l2"
	now := time.Now().UTC()
	earliest := now.Add(-15 * 24 * time.Hour)

	propRepo := proposals.NewMemoryRepository()
	for i := 0; i < 10; i++ {
		ts := earliest.Add(time.Duration(i) * time.Hour)
		propRepo.SeedHistoricalProposal(proposals.Proposal{
			TenantID: tid, ProposalID: fmt.Sprintf("h-%d", i), AgentID: proposals.AgentReorder,
			ProposalType: proposals.TypeReorder, TargetObjectType: "Product", TargetObjectID: fmt.Sprintf("sku-h-%d", i),
			Title: "hist", Rationale: "hist", PayloadJSON: "{}", PayloadHash: "h",
			Status: proposals.StatusExecuted, AutonomyLevel: proposals.AutonomyLevelL1,
			EstimatedImpactMinor: 50_000, CreatedBy: "agent:reorder_agent", ActorType: auth.ActorTypeAgent,
			ExecutedAt: &ts, CreatedAt: ts, UpdatedAt: ts,
		})
	}

	roiSvc := roi.NewServiceWithRepository(roi.NewMemoryRepository())
	svc := proposals.NewServiceWithRepository(propRepo)
	svc.SetApprovals(approvals.NewServiceWithRepository(approvals.NewMemoryRepository()))
	svc.SetROI(roiSvc)

	r := chi.NewRouter()
	r.Use(auth.SessionAuth(jwtSecret))
	r.Use(auth.AttachTenantFromClaims)
	proposalroutes.RegisterRoutes(r, proposalroutes.Deps{
		Service:   svc,
		JWTSecret: jwtSecret,
	})

	issue := func(sub, supRole, actorType string, audExp *time.Time) string {
		tok, err := auth.Issue(auth.Claims{
			Subject:          sub,
			Role:             auth.RoleAdmin,
			SupplierID:       tid,
			SupplierRole:     auth.Role(supRole),
			ActorType:        actorType,
			AuditorExpiresAt: audExp,
		}, auth.IssueOptions{Secret: jwtSecret, TTL: time.Hour})
		if err != nil {
			t.Fatal(err)
		}
		return tok
	}
	ownerTok := issue("owner-1", auth.SupplierRoleOwner, auth.ActorTypeHuman, nil)
	agentTok := issue("agent-1", auth.SupplierRoleAgent, auth.ActorTypeAgent, nil)
	auditorTok := issue("aud-1", auth.SupplierRoleAuditor, auth.ActorTypeHuman, nil)
	past := now.Add(-time.Minute)
	expiredAuditorTok := issue("aud-exp", auth.SupplierRoleAuditor, auth.ActorTypeHuman, &past)

	call := func(method, path string, body any, tok string) *httptest.ResponseRecorder {
		var buf bytes.Buffer
		if body != nil {
			_ = json.NewEncoder(&buf).Encode(body)
		}
		req := httptest.NewRequest(method, path, &buf)
		if tok != "" {
			req.Header.Set("Authorization", "Bearer "+tok)
		}
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec
	}

	// Create 1 low-risk REORDER proposal + 1 high-risk PAYOUT proposal
	if rec := call(http.MethodPost, "/v1/proposals", map[string]any{
		"agent_id": "REORDER_AGENT", "proposal_type": "REORDER", "target_object_type": "Product", "target_object_id": "sku-l2-rt",
		"title": "Low reorder", "payload": map[string]any{"action": "open_supply_request", "product_id": "sku-l2-rt", "suggested_qty": 4, "unit_price_minor": 10000},
		"estimated_impact_minor": 40_000,
	}, agentTok); rec.Code != http.StatusCreated {
		t.Fatalf("create reorder = %d body=%s", rec.Code, rec.Body.String())
	}
	if rec := call(http.MethodPost, "/v1/proposals", map[string]any{
		"agent_id": "REORDER_AGENT", "proposal_type": "PAYOUT", "target_object_type": "PayoutBatch", "target_object_id": "pay-1",
		"title": "High risk payout", "payload": map[string]any{"action": "PAYOUT", "amount_minor": 5000},
		"estimated_impact_minor": 5_000,
	}, agentTok); rec.Code != http.StatusCreated {
		t.Fatalf("create payout = %d body=%s", rec.Code, rec.Body.String())
	}

	// Expired AUDITOR -> 401 auditor_access_expired
	if rec := call(http.MethodGet, "/v1/proposals/l2-evaluate", nil, expiredAuditorTok); rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), "auditor_access_expired") {
		t.Fatalf("expired auditor GET l2-evaluate = %d body=%s", rec.Code, rec.Body.String())
	}
	// Active AUDITOR can GET, blocked 403 on POST
	if rec := call(http.MethodGet, "/v1/proposals/l2-evaluate", nil, auditorTok); rec.Code != http.StatusOK {
		t.Fatalf("auditor GET l2-evaluate = %d body=%s", rec.Code, rec.Body.String())
	}
	if rec := call(http.MethodPost, "/v1/proposals/l2-evaluate", map[string]any{"auto_execute": true}, auditorTok); rec.Code != http.StatusForbidden {
		t.Fatalf("auditor POST l2-evaluate = %d, want 403", rec.Code)
	}
	// AGENT blocked 403 on POST
	if rec := call(http.MethodPost, "/v1/proposals/l2-evaluate", map[string]any{"auto_execute": true}, agentTok); rec.Code != http.StatusForbidden {
		t.Fatalf("agent POST l2-evaluate = %d, want 403", rec.Code)
	}

	// OWNER executes L2 -> promotes REORDER to L2, blocks PAYOUT!
	execRec := call(http.MethodPost, "/v1/proposals/l2-evaluate", map[string]any{"auto_execute": true, "max_l2_amount_minor": 100000}, ownerTok)
	if execRec.Code != http.StatusOK {
		t.Fatalf("owner POST l2-evaluate = %d body=%s", execRec.Code, execRec.Body.String())
	}
	var out proposals.L2EvaluationResult
	if err := json.Unmarshal(execRec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode L2 result: %v", err)
	}
	if !out.Eligible || out.AutoExecutedCount != 1 || len(out.BlockedProposals) != 1 {
		t.Fatalf("unexpected L2 HTTP outcome: %+v", out)
	}
	if out.ExecutedProposals[0].AutonomyLevel != proposals.AutonomyLevelL2 {
		t.Fatalf("expected L2 autonomy level, got %+v", out.ExecutedProposals[0])
	}
}
