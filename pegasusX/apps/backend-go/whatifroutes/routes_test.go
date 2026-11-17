package whatifroutes_test

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
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/whatif"
	"github.com/pegasusx/pegasusx/apps/backend-go/whatifroutes"
)

type routeProposalStub struct{}

func (r *routeProposalStub) CreateProposal(_ context.Context, in whatif.ProposalCreateInput) (whatif.CreatedProposal, error) {
	return whatif.CreatedProposal{
		TenantID:         in.TenantID,
		ProposalID:       "prop_wirt_" + in.ProposalType,
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

func (a *routeAuditStub) RecordEvent(_ context.Context, _ whatif.AuditRecordInput) error {
	a.count++
	return nil
}

func (a *routeAuditStub) RecordEventInTxn(_ context.Context, _ *spanner.ReadWriteTransaction, _ whatif.AuditRecordInput) error {
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
		Issuer: "whatifroutes-test",
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

func TestWhatIfRoutes_RBAC_AuditorExpiry401_AuditorAgent403_NoFloats_And_Commit(t *testing.T) {
	t.Setenv("ENFORCE_TENANT_CONTEXT", "1")
	t.Setenv("CSRF_ENFORCE", "0")

	const secret = "whatifroutes-test-secret-32byte!!"
	const supA = "sup-wi-rt-a"
	const supB = "sup-wi-rt-b"

	repo := whatif.NewMemoryRepository()
	repo.SeedDataset(supA, whatif.SimulationDataset{
		TenantID: supA,
		Forecasts: []whatif.ForecastRecord{
			{SKU: "SKU-1", ProductID: "SKU-1", LocationID: "wh-1", IsWinner: true, P50Milli: 500_000},
		},
		Inventory: []whatif.InventoryRecord{
			{WarehouseID: "wh-1", ProductID: "SKU-1", QuantityOnHand: 450, ReorderThreshold: 50},
		},
		Products: []whatif.ProductPriceRecord{
			{ProductID: "SKU-1", Name: "Tea", PriceMinor: 50_000, Currency: "UZS"},
		},
	})

	svc := whatif.NewService(repo, whatif.ServiceConfig{
		Audit:     &routeAuditStub{},
		Proposals: &routeProposalStub{},
	})

	r := chi.NewRouter()
	whatifroutes.RegisterRoutes(r, whatifroutes.Deps{
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
	if rec := doRawHTTP(t, r, http.MethodGet, "/v1/whatif/simulations", expiredAuditorTok, nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("expired auditor GET simulations = %d body=%s", rec.Code, rec.Body.String())
	}
	if rec := doRawHTTP(t, r, http.MethodPost, "/v1/whatif/simulate", expiredAuditorTok, []byte(`{"name":"exp"}`)); rec.Code != http.StatusUnauthorized {
		t.Fatalf("expired auditor POST simulate = %d body=%s", rec.Code, rec.Body.String())
	}

	// 2. Active AUDITOR allowed on GET, blocked 403 on POST.
	if rec := doRawHTTP(t, r, http.MethodGet, "/v1/whatif/simulations", auditorTok, nil); rec.Code != http.StatusOK {
		t.Fatalf("auditor GET simulations = %d body=%s", rec.Code, rec.Body.String())
	}
	if rec := doRawHTTP(t, r, http.MethodPost, "/v1/whatif/simulate", auditorTok, []byte(`{"name":"aud"}`)); rec.Code != http.StatusForbidden {
		t.Fatalf("auditor POST simulate = %d, want 403", rec.Code)
	}

	// 3. AGENT blocked 403 on simulate.
	if rec := doRawHTTP(t, r, http.MethodPost, "/v1/whatif/simulate", agentTok, []byte(`{"name":"agt"}`)); rec.Code != http.StatusForbidden {
		t.Fatalf("agent POST simulate = %d, want 403", rec.Code)
	}

	// 4. Float rejection -> 400 float_forbidden.
	if rec := doRawHTTP(t, r, http.MethodPost, "/v1/whatif/simulate", ownerTok, []byte(`{"name":"flt","demand_delta_bps":12.5}`)); rec.Code != http.StatusBadRequest {
		t.Fatalf("float JSON simulate = %d, want 400 body=%s", rec.Code, rec.Body.String())
	}

	// 5. Cross-tenant mismatch -> 403.
	if rec := doRawHTTP(t, r, http.MethodGet, "/v1/whatif/simulations?tenant_id="+supA, otherTenantTok, nil); rec.Code != http.StatusForbidden {
		t.Fatalf("cross-tenant GET simulations = %d, want 403", rec.Code)
	}

	// 6. Happy path: POST simulate -> GET simulations -> GET sim/{id} -> POST commit-proposals -> duplicate commit 409.
	simRec := doRawHTTP(t, r, http.MethodPost, "/v1/whatif/simulate", ownerTok, []byte(`{
		"name":"Peak Promo",
		"horizon_days":30,
		"demand_delta_bps":1500,
		"factory_downtime_hours":12,
		"fx_rate_shock_bps":300,
		"credit_limit_delta_bps":500,
		"unit_price_delta_bps":200
	}`))
	if simRec.Code != http.StatusCreated {
		t.Fatalf("owner POST simulate = %d body=%s", simRec.Code, simRec.Body.String())
	}
	var simOut struct {
		Simulation whatif.WhatIfSimulation `json:"simulation"`
	}
	if err := json.Unmarshal(simRec.Body.Bytes(), &simOut); err != nil || simOut.Simulation.SimulationID == "" {
		t.Fatalf("decode simulate response: err=%v body=%s", err, simRec.Body.String())
	}
	simID := simOut.Simulation.SimulationID

	if rec := doRawHTTP(t, r, http.MethodGet, "/v1/whatif/simulations/"+simID, auditorTok, nil); rec.Code != http.StatusOK {
		t.Fatalf("auditor GET sim/{id} = %d body=%s", rec.Code, rec.Body.String())
	}
	if rec := doRawHTTP(t, r, http.MethodPost, "/v1/whatif/simulations/"+simID+"/commit-proposals", auditorTok, []byte(`{}`)); rec.Code != http.StatusForbidden {
		t.Fatalf("auditor POST commit-proposals = %d, want 403", rec.Code)
	}
	if rec := doRawHTTP(t, r, http.MethodPost, "/v1/whatif/simulations/"+simID+"/commit-proposals", agentTok, []byte(`{}`)); rec.Code != http.StatusForbidden {
		t.Fatalf("agent POST commit-proposals = %d, want 403", rec.Code)
	}

	commitRec := doRawHTTP(t, r, http.MethodPost, "/v1/whatif/simulations/"+simID+"/commit-proposals", ownerTok, []byte(`{}`))
	if commitRec.Code != http.StatusOK {
		t.Fatalf("owner POST commit-proposals = %d body=%s", commitRec.Code, commitRec.Body.String())
	}
	var commitOut whatif.CommitProposalsResult
	if err := json.Unmarshal(commitRec.Body.Bytes(), &commitOut); err != nil || len(commitOut.CommittedProposalIDs) == 0 || commitOut.Simulation.Status != whatif.StatusCommitted {
		t.Fatalf("unexpected commit-proposals output: err=%v body=%s", err, commitRec.Body.String())
	}

	// Duplicate commit without matching idempotency key returns 409 Conflict.
	dupRec := doRawHTTP(t, r, http.MethodPost, "/v1/whatif/simulations/"+simID+"/commit-proposals", ownerTok, []byte(`{}`))
	if dupRec.Code != http.StatusConflict {
		t.Fatalf("duplicate commit-proposals = %d, want 409 body=%s", dupRec.Code, dupRec.Body.String())
	}
}
