package anomalyroutes_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/anomalies"
	"github.com/pegasusx/pegasusx/apps/backend-go/anomalyroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

type routeProposalStub struct{}

func (r *routeProposalStub) CreateProposal(_ context.Context, in anomalies.ProposalCreateInput) (anomalies.CreatedProposal, error) {
	return anomalies.CreatedProposal{
		TenantID:         in.TenantID,
		ProposalID:       "prop_rt_" + in.TargetObjectID,
		AgentID:          in.AgentID,
		ProposalType:     in.ProposalType,
		TargetObjectType: in.TargetObjectType,
		TargetObjectID:   in.TargetObjectID,
		Title:            in.Title,
		Status:           "PENDING",
		Currency:         "UZS",
		CreatedBy:        in.CreatedBy,
	}, nil
}

type routeAuditStub struct {
	count int
}

func (a *routeAuditStub) RecordEvent(_ context.Context, _ anomalies.AuditRecordInput) error {
	a.count++
	return nil
}

func (a *routeAuditStub) RecordEventInTxn(_ context.Context, _ *spanner.ReadWriteTransaction, _ anomalies.AuditRecordInput) error {
	a.count++
	return nil
}

func issueTestJWT(t *testing.T, secret, supplierID, sub string, role, supRole auth.Role, actorType string, audExp *time.Time, perms ...string) string {
	t.Helper()
	tok, err := auth.Issue(auth.Claims{
		Subject:          sub,
		Role:             role,
		SupplierID:       supplierID,
		SupplierRole:     supRole,
		ActorType:        actorType,
		SupplierPerms:    perms,
		AuditorExpiresAt: audExp,
	}, auth.IssueOptions{
		Secret: secret,
		Issuer: "anomalyroutes-test",
		TTL:    time.Hour,
	})
	if err != nil {
		t.Fatalf("auth.Issue: %v", err)
	}
	return tok
}

func doRawHTTP(t *testing.T, h http.Handler, method, path, tok string, rawBody []byte) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewReader(rawBody))
	if rawBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestAnomalyRoutes_RBAC_AuditorExpiry401_AuditorAgent403_NoFloats_And_Lifecycle(t *testing.T) {
	t.Setenv("ENFORCE_TENANT_CONTEXT", "1")
	t.Setenv("CSRF_ENFORCE", "0")

	const secret = "anomalyroutes-test-secret-32byte!!"
	const supA = "sup-anom-rt-a"
	const supB = "sup-anom-rt-b"

	repo := anomalies.NewMemoryRepository()
	repo.SeedDataset(supA, anomalies.DetectorDataset{
		TenantID: supA,
		ReconBreaks: []anomalies.ReconBreakRecord{
			{
				BreakID:       "brk-rt-1",
				RunID:         "run-rt-1",
				Dimension:     "STOCK_GL",
				ObjectType:    "Warehouse",
				ObjectID:      "wh-rt-1",
				ExpectedMinor: 2_000_000,
				ActualMinor:   1_000_000,
				DeltaMinor:    -1_000_000,
				Severity:      "HIGH",
				Status:        "OPEN",
			},
		},
	})

	auditStub := &routeAuditStub{}
	svc := anomalies.NewService(repo, anomalies.ServiceConfig{
		Audit:     auditStub,
		Proposals: &routeProposalStub{},
	})

	r := chi.NewRouter()
	anomalyroutes.RegisterRoutes(r, anomalyroutes.Deps{
		Service:   svc,
		JWTSecret: secret,
	})

	ownerTok := issueTestJWT(t, secret, supA, "owner-1", auth.RoleAdmin, auth.SupplierRoleOwner, auth.ActorTypeHuman, nil, auth.PermReconManage, auth.PermAuditRead)
	activeExp := time.Now().UTC().Add(2 * time.Hour)
	auditorTok := issueTestJWT(t, secret, supA, "aud-1", auth.RoleAdmin, auth.SupplierRoleAuditor, auth.ActorTypeHuman, &activeExp, auth.PermAuditRead)
	expiredAt := time.Now().UTC().Add(-10 * time.Minute)
	expiredAuditorTok := issueTestJWT(t, secret, supA, "aud-exp", auth.RoleAdmin, auth.SupplierRoleAuditor, auth.ActorTypeHuman, &expiredAt, auth.PermAuditRead)
	agentTok := issueTestJWT(t, secret, supA, "agent-1", auth.RoleAdmin, auth.SupplierRoleAgent, auth.ActorTypeAgent, nil, auth.PermReconManage)
	otherTenantTok := issueTestJWT(t, secret, supB, "owner-b", auth.RoleAdmin, auth.SupplierRoleOwner, auth.ActorTypeHuman, nil, auth.PermReconManage, auth.PermAuditRead)

	// 1. Expired AUDITOR -> 401 auditor_access_expired on GET & POST.
	if rec := doRawHTTP(t, r, http.MethodGet, "/v1/anomalies/catalogue", expiredAuditorTok, nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("expired auditor GET catalogue = %d body=%s", rec.Code, rec.Body.String())
	}
	if rec := doRawHTTP(t, r, http.MethodPost, "/v1/anomalies/scan", expiredAuditorTok, []byte(`{}`)); rec.Code != http.StatusUnauthorized {
		t.Fatalf("expired auditor POST scan = %d body=%s", rec.Code, rec.Body.String())
	}

	// 2. Active AUDITOR allowed on GETs, blocked 403 on POSTs.
	if rec := doRawHTTP(t, r, http.MethodGet, "/v1/anomalies/catalogue", auditorTok, nil); rec.Code != http.StatusOK {
		t.Fatalf("auditor GET catalogue = %d body=%s", rec.Code, rec.Body.String())
	}
	if rec := doRawHTTP(t, r, http.MethodGet, "/v1/anomalies/cases", auditorTok, nil); rec.Code != http.StatusOK {
		t.Fatalf("auditor GET cases = %d body=%s", rec.Code, rec.Body.String())
	}
	if rec := doRawHTTP(t, r, http.MethodPost, "/v1/anomalies/scan", auditorTok, []byte(`{}`)); rec.Code != http.StatusForbidden {
		t.Fatalf("auditor POST scan = %d, want 403", rec.Code)
	}

	// 3. AGENT blocked 403 on mutations.
	if rec := doRawHTTP(t, r, http.MethodPost, "/v1/anomalies/scan", agentTok, []byte(`{}`)); rec.Code != http.StatusForbidden {
		t.Fatalf("agent POST scan = %d, want 403", rec.Code)
	}

	// 4. Float rejection (400 float_forbidden).
	if rec := doRawHTTP(t, r, http.MethodPost, "/v1/anomalies/scan", ownerTok, []byte(`{"threshold_bps":12.5}`)); rec.Code != http.StatusBadRequest {
		t.Fatalf("float JSON scan = %d, want 400 body=%s", rec.Code, rec.Body.String())
	}

	// 5. Cross-tenant mismatch -> 403.
	if rec := doRawHTTP(t, r, http.MethodGet, "/v1/anomalies/cases?tenant_id="+supA, otherTenantTok, nil); rec.Code != http.StatusForbidden {
		t.Fatalf("cross-tenant GET cases = %d, want 403", rec.Code)
	}

	// 6. Happy path: scan -> list -> escalate with create_proposal=true.
	scanRec := doRawHTTP(t, r, http.MethodPost, "/v1/anomalies/scan", ownerTok, []byte(`{}`))
	if scanRec.Code != http.StatusOK {
		t.Fatalf("owner POST scan = %d body=%s", scanRec.Code, scanRec.Body.String())
	}
	var scanOut struct {
		Cases        []anomalies.AnomalyCase `json:"cases"`
		CreatedCount int                     `json:"created_count"`
	}
	if err := json.Unmarshal(scanRec.Body.Bytes(), &scanOut); err != nil || scanOut.CreatedCount != 1 {
		t.Fatalf("unexpected scan response: err=%v body=%s", err, scanRec.Body.String())
	}
	caseID := scanOut.Cases[0].CaseID

	escRec := doRawHTTP(t, r, http.MethodPost, "/v1/anomalies/cases/"+caseID+"/resolve", ownerTok, []byte(`{"status":"ESCALATED","resolution_note":"Cycle count variance confirmed","create_proposal":true}`))
	if escRec.Code != http.StatusOK {
		t.Fatalf("owner POST resolve = %d body=%s", escRec.Code, escRec.Body.String())
	}
	var escOut anomalies.ResolveResult
	if err := json.Unmarshal(escRec.Body.Bytes(), &escOut); err != nil || escOut.ProposalID == "" {
		t.Fatalf("expected proposal_id on resolve, got err=%v body=%s", err, escRec.Body.String())
	}

	if rec := doRawHTTP(t, r, http.MethodPost, "/v1/anomalies/cases/"+caseID+"/resolve", auditorTok, []byte(`{"status":"RESOLVED","resolution_note":"no"}`)); rec.Code != http.StatusForbidden {
		t.Fatalf("auditor POST resolve = %d, want 403", rec.Code)
	}
	if rec := doRawHTTP(t, r, http.MethodPost, "/v1/anomalies/cases/"+caseID+"/resolve", agentTok, []byte(`{"status":"RESOLVED","resolution_note":"no"}`)); rec.Code != http.StatusForbidden {
		t.Fatalf("agent POST resolve = %d, want 403", rec.Code)
	}
}
