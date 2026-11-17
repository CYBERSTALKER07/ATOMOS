package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/approvals"
	"github.com/pegasusx/pegasusx/apps/backend-go/audit"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
	"github.com/pegasusx/pegasusx/apps/backend-go/credit"
	"github.com/pegasusx/pegasusx/apps/backend-go/driver"
	"github.com/pegasusx/pegasusx/apps/backend-go/esf"
	"github.com/pegasusx/pegasusx/apps/backend-go/foundry"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
	"github.com/pegasusx/pegasusx/apps/backend-go/iam"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/payout"
	"github.com/pegasusx/pegasusx/apps/backend-go/periods"
	"github.com/pegasusx/pegasusx/apps/backend-go/recon"
	"github.com/pegasusx/pegasusx/apps/backend-go/seed"
	"github.com/pegasusx/pegasusx/apps/backend-go/stocklots"
	"github.com/pegasusx/pegasusx/apps/backend-go/valuation"
	"github.com/pegasusx/pegasusx/apps/backend-go/warehouse"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

func openRootAuditSpannerClient(t *testing.T) *spanner.Client {
	t.Helper()
	requireSpanner := strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1"
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if requireSpanner {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping live Spanner NewHTTPRouter route inventory test")
	}
	envOr := func(k, def string) string {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
		return def
	}
	dbPath := "projects/" + envOr("SPANNER_PROJECT", "pegasusx-local") +
		"/instances/" + envOr("SPANNER_INSTANCE", "pegasusx-instance") +
		"/databases/" + envOr("SPANNER_DATABASE", "pegasusx-db")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, err := spanner.NewClient(ctx, dbPath,
		option.WithEndpoint(host),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		if requireSpanner {
			t.Fatalf("spanner dial: %v", err)
		}
		t.Skipf("spanner unavailable: %v", err)
	}
	if _, pErr := c.Single().WithTimestampBound(spanner.StrongRead()).ReadRow(ctx, "AuditEvents", spanner.Key{"__probe__", "2026-01-01", "__probe__"}, []string{"EventId"}); pErr != nil && spanner.ErrCode(pErr) != codes.NotFound {
		c.Close()
		if requireSpanner {
			t.Fatalf("AuditEvents table probe: %v", pErr)
		}
		t.Skipf("AuditEvents unavailable: %v", pErr)
	}
	t.Cleanup(c.Close)
	return c
}

func fnContainsAuditMiddleware(mw func(http.Handler) http.Handler, refPtr uintptr) bool {
	if mw == nil {
		return false
	}
	ptr := reflect.ValueOf(mw).Pointer()
	if refPtr != 0 && ptr == refPtr {
		return true
	}
	fn := runtime.FuncForPC(ptr)
	if fn == nil {
		return false
	}
	return strings.Contains(fn.Name(), "AuditMiddleware")
}

func TestNewHTTPRouter_100PercentMutatingRouteAuditInventory(t *testing.T) {
	client := openRootAuditSpannerClient(t)
	ctx := context.Background()
	supID := "sup-rt-inv-" + uuid.NewString()[:8]
	today := time.Now().UTC().Format("2006-01-02")

	t.Cleanup(func() {
		cleanCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = client.Apply(cleanCtx, []*spanner.Mutation{
			spanner.Delete("AuditEvents", spanner.KeyRange{Start: spanner.Key{supID, today}, End: spanner.Key{supID, today}, Kind: spanner.ClosedClosed}),
			spanner.Delete("RolePermissions", spanner.KeyRange{Start: spanner.Key{supID}, End: spanner.Key{supID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("SupplierUserRoles", spanner.KeyRange{Start: spanner.Key{supID}, End: spanner.Key{supID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("SodPolicies", spanner.KeyRange{Start: spanner.Key{supID}, End: spanner.Key{supID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("GlAccounts", spanner.KeyRange{Start: spanner.Key{supID}, End: spanner.Key{supID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("SupplierPayoutPolicies", spanner.Key{supID}),
			spanner.Delete("CustomRoles", spanner.KeyRange{Start: spanner.Key{supID}, End: spanner.Key{supID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("ScimIdentities", spanner.KeyRange{Start: spanner.Key{supID}, End: spanner.Key{supID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("AnomalyCases", spanner.KeyRange{Start: spanner.Key{supID}, End: spanner.Key{supID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("WhatIfSimulations", spanner.KeyRange{Start: spanner.Key{supID}, End: spanner.Key{supID}, Kind: spanner.ClosedClosed}),
		})
	})

	auditSvc := audit.NewService(client)
	iamSvc := iam.NewService(client)
	iamSvc.SetAuditRecorder(bootstrap.NewIAMAuditRecorder(auditSvc))

	glSvc := gl.NewService(client)
	_ = glSvc.SeedDefaults(ctx, supID, "UZ", "UZS")
	reconSvc := recon.NewService(client)
	creditSvc := credit.NewService(credit.NewSpannerRepository(client))
	creditPolicySvc := credit.NewPolicyService(credit.NewSpannerPolicyRepository(client), creditSvc)
	payoutSvc := payout.NewService(payout.NewRepository(client))
	orderSvc := order.NewService(order.ServiceConfig{
		Repo:           order.NewSpannerRepository(client),
		SeedSupplierID: supID,
		SpannerClient:  client,
	})
	orderSvc.SetDraftSoDCheck(bootstrap.CheckOrderDraftSoD(iamSvc))
	warehouseSvc := warehouse.NewService(warehouse.ServiceConfig{
		Repo: warehouse.NewSpannerRepository(client),
	})
	driverSvc := driver.NewService(driver.ServiceConfig{
		Repo: driver.NewSpannerRepository(client),
	})
	foundryEng := foundry.NewEngine().WithSpannerClient(client).WithAuditHook(bootstrap.BuildFoundryAuditHook(auditSvc, nil))

	cfg := &bootstrap.Config{
		JWTSecret:             "root-router-audit-inventory-secret-32bytes!",
		JWTIssuer:             "pegasusx-test",
		HTTPPort:              "8080",
		TenantContextEnforced: true,
	}
	app := &bootstrap.App{
		Config:              cfg,
		Spanner:             client,
		Supplier:            seed.Supplier{SupplierID: supID},
		Audit:               auditSvc,
		IAM:                 iamSvc,
		GL:                  glSvc,
		Periods:             periods.NewService(client),
		Approvals:           approvals.NewService(client),
		Valuation:           valuation.NewService(client),
		ESF:                 esf.NewService(esf.ServiceConfig{Repo: esf.NewSpannerRepository(client)}),
		Recon:               reconSvc,
		CreditService:       creditSvc,
		CreditPolicyService: creditPolicySvc,
		PayoutService:       payoutSvc,
		OrderService:        orderSvc,
		WarehouseService:    warehouseSvc,
		DriverService:       driverSvc,
		FoundryEngine:       foundryEng,
	}
	if err := bootstrap.SetupWave3A(ctx, cfg, client, nil, app, nil); err != nil {
		t.Fatalf("SetupWave3A: %v", err)
	}
	if err := bootstrap.SetupWave3B(ctx, cfg, client, app, nil); err != nil {
		t.Fatalf("SetupWave3B: %v", err)
	}
	if err := bootstrap.SetupWave3C(ctx, cfg, client, app, nil); err != nil {
		t.Fatalf("SetupWave3C: %v", err)
	}
	if err := bootstrap.SetupWave4A(ctx, cfg, client, app, nil); err != nil {
		t.Fatalf("SetupWave4A: %v", err)
	}
	if err := bootstrap.SetupWave4B(ctx, cfg, client, app, nil); err != nil {
		t.Fatalf("SetupWave4B: %v", err)
	}
	t.Cleanup(func() {
		gl.SetPeriodLockChecker(nil)
		glSvc.SetLegalEntityOwnershipChecker(nil)
		stocklots.SetPickValuationHook(nil)
		stocklots.SetAdjustmentValuationHook(nil)
	})

	router := NewHTTPRouter(HTTPRouterParams{
		Config: cfg,
		App:    app,
	})

	// 1. Verify root NewHTTPRouter middleware chain contains audit.AuditMiddleware.
	refAuditMWPtr := reflect.ValueOf(audit.AuditMiddleware(auditSvc)).Pointer()
	if cr, ok := router.(*chi.Mux); !ok {
		t.Fatalf("expected *chi.Mux from NewHTTPRouter")
	} else {
		rootHasAudit := false
		var seenMW []string
		for _, mw := range cr.Middlewares() {
			if fn := runtime.FuncForPC(reflect.ValueOf(mw).Pointer()); fn != nil {
				seenMW = append(seenMW, fn.Name())
			}
			if fnContainsAuditMiddleware(mw, refAuditMWPtr) {
				rootHasAudit = true
			}
		}
		if !rootHasAudit {
			t.Fatalf("expected audit.AuditMiddleware in NewHTTPRouter root middleware chain, saw %v", seenMW)
		}
	}

	// 2. Walk 100% of registered routes via chi.Walk and verify every POST/PUT/PATCH/DELETE route.
	mutatingRoutes := make(map[string]struct{})
	err := chi.Walk(router, func(method, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		switch method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			hasAudit := false
			for _, mw := range middlewares {
				if fnContainsAuditMiddleware(mw, refAuditMWPtr) {
					hasAudit = true
					break
				}
			}
			// chi.Walk passes route/group middlewares; root r.Use(...) wraps all routes on r.
			if !hasAudit {
				if cr, ok := router.(*chi.Mux); ok {
					for _, mw := range cr.Middlewares() {
						if fnContainsAuditMiddleware(mw, refAuditMWPtr) {
							hasAudit = true
							break
						}
					}
				}
			}
			if !hasAudit {
				t.Errorf("mutating route %s %s does not have AuditMiddleware in chain", method, route)
			}
			mutatingRoutes[method+" "+route] = struct{}{}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("chi.Walk failed: %v", err)
	}
	if len(mutatingRoutes) < 40 {
		t.Fatalf("expected comprehensive mutating route inventory (>=40), got %d", len(mutatingRoutes))
	}

	expectedMounted := []string{
		"POST /v1/finance/drafts/review",
		"POST /v1/finance/journal/manual",
		"POST /v1/finance/recon/run",
		"POST /v1/warehouse/ops/inventory-adjustments/{adjustmentID}/approve",
		"PATCH /v1/supplier/retailer-credit-profile",
		"PATCH /v1/supplier/payout-policy",
		"POST /v1/iam/users/{userId}/roles",
		"PUT /v1/iam/sod-policies/{id}",
		"POST /v1/audit/anchors",
		"POST /v1/audit/legal-hold",
		"POST /v1/finance/periods",
		"POST /v1/finance/periods/{id}/soft-close",
		"POST /v1/finance/periods/{id}/close",
		"POST /v1/finance/periods/{id}/reopen",
		"POST /v1/approvals/policies",
		"PUT /v1/approvals/policies/{id}",
		"POST /v1/approvals/{id}/approve",
		"POST /v1/approvals/{id}/reject",
		"POST /v1/approvals/delegations",
		"POST /v1/webhooks/telegram/approvals",
		"POST /v1/inventory/cost-layers/receipt",
		"POST /v1/fiscal/esf",
		"POST /v1/fiscal/esf/inbound",
		"POST /v1/fiscal/esf/{id}/status",
		"POST /v1/planning/forecast/run",
		"POST /v1/planning/forecast/explanations/{id}/approve",
		"POST /v1/replenishment/suggestions/{id}/approve",
		"POST /v1/credit/limits/{id}",
		"POST /v1/proposals",
		"POST /v1/proposals/agents/{agentId}/run",
		"POST /v1/proposals/{id}/approve",
		"POST /v1/proposals/{id}/reject",
		"POST /v1/proposals/{id}/execute",
		"POST /v1/roi/baseline",
		"POST /v1/roi/events",
		"PUT /v1/connectors/credentials",
		"POST /v1/connectors/{provider}/health",
		"POST /v1/connectors/{provider}/pull",
		"POST /v1/connectors/{provider}/push",
		"POST /v1/connectors/{provider}/shadow-compare",
		"POST /v1/finance/packs",
		"POST /v1/org/entities",
		"PATCH /v1/org/entities/{entityID}",
		"POST /v1/finance/intercompany-transfers",
		"POST /v1/finance/fx-revaluation",
		"POST /v1/processes/templates",
		"POST /v1/processes/instances",
		"POST /v1/p2p/purchase-orders",
		"POST /v1/p2p/purchase-orders/{poID}/approve",
		"POST /v1/p2p/receipts",
		"POST /v1/p2p/vendor-invoices",
		"POST /v1/p2p/three-way-match",
		"POST /v1/proposals/l2-evaluate",
		"POST /v1/iam/custom-roles",
		"DELETE /v1/iam/custom-roles/{roleName}",
		"POST /scim/v2/Users",
		"PUT /scim/v2/Users/{id}",
		"PATCH /scim/v2/Users/{id}",
		"DELETE /scim/v2/Users/{id}",
		"POST /scim/v2/Groups",
		"PUT /scim/v2/Groups/{id}",
		"PATCH /scim/v2/Groups/{id}",
		"DELETE /scim/v2/Groups/{id}",
		"POST /v1/anomalies/scan",
		"POST /v1/anomalies/cases/{caseID}/resolve",
		"POST /v1/whatif/simulate",
		"POST /v1/whatif/simulations/{simID}/commit-proposals",
	}
	for _, want := range expectedMounted {
		if _, ok := mutatingRoutes[want]; !ok {
			t.Fatalf("expected route %q in NewHTTPRouter mutating route inventory", want)
		}
	}

	// 3. Fire representative HTTP mutations through NewHTTPRouter and verify unbroken hash chain.
	ownerTok, err := auth.Issue(auth.Claims{
		Subject:      "owner-root-1",
		Role:         auth.RoleAdmin,
		SupplierID:   supID,
		SupplierRole: auth.SupplierRoleOwner,
	}, auth.IssueOptions{Secret: cfg.JWTSecret, TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}

	doReq := func(method, path string, body any) *httptest.ResponseRecorder {
		var buf bytes.Buffer
		if body != nil {
			_ = json.NewEncoder(&buf).Encode(body)
		}
		req := httptest.NewRequest(method, path, &buf)
		req.Header.Set("Authorization", "Bearer "+ownerTok)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", uuid.NewString())
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		return rr
	}

	if rr := doReq(http.MethodPost, "/v1/iam/users/u-root-1/roles", map[string]any{
		"role_name": auth.SupplierRoleAccountant,
		"action":    iam.RoleActionAssign,
	}); rr.Code != http.StatusOK {
		t.Fatalf("POST /v1/iam/users/u-root-1/roles = %d body=%s", rr.Code, rr.Body.String())
	}
	if rr := doReq(http.MethodPatch, "/v1/supplier/payout-policy", map[string]any{
		"payout_mode": payout.PayoutModeHQSupplier,
		"reason":      "root router audit verification",
	}); rr.Code != http.StatusOK {
		t.Fatalf("PATCH /v1/supplier/payout-policy = %d body=%s", rr.Code, rr.Body.String())
	}
	if rr := doReq(http.MethodPost, "/v1/iam/custom-roles", map[string]any{
		"role_name":   "AUDIT_ROOT_CLERK",
		"permissions": []string{auth.PermStockCount, auth.PermAuditRead},
	}); rr.Code != http.StatusCreated {
		t.Fatalf("POST /v1/iam/custom-roles = %d body=%s", rr.Code, rr.Body.String())
	}
	if rr := doReq(http.MethodPost, "/v1/whatif/simulate", map[string]any{
		"name":             "Root Route Audit Sim",
		"demand_delta_bps": 1000,
		"horizon_days":     14,
	}); rr.Code != http.StatusCreated {
		t.Fatalf("POST /v1/whatif/simulate = %d body=%s", rr.Code, rr.Body.String())
	}
	if rr := doReq(http.MethodGet, "/v1/compliance/haccp", nil); rr.Code != http.StatusOK {
		t.Fatalf("GET /v1/compliance/haccp = %d body=%s", rr.Code, rr.Body.String())
	}

	rrVerify := doReq(http.MethodGet, "/v1/audit/verify?date="+today, nil)
	if rrVerify.Code != http.StatusOK {
		t.Fatalf("GET /v1/audit/verify = %d body=%s", rrVerify.Code, rrVerify.Body.String())
	}
	var vr audit.ChainVerifyResult
	if err := json.Unmarshal(rrVerify.Body.Bytes(), &vr); err != nil {
		t.Fatal(err)
	}
	if !vr.Valid || vr.EventCount < 5 {
		t.Fatalf("expected valid hash chain with >=5 events via NewHTTPRouter, got %+v", vr)
	}
}
