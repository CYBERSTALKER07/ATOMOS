package e2ep0

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
	"github.com/pegasusx/pegasusx/apps/backend-go/approvalroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/approvals"
	"github.com/pegasusx/pegasusx/apps/backend-go/audit"
	"github.com/pegasusx/pegasusx/apps/backend-go/auditroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
	"github.com/pegasusx/pegasusx/apps/backend-go/credit"
	"github.com/pegasusx/pegasusx/apps/backend-go/creditroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/esf"
	"github.com/pegasusx/pegasusx/apps/backend-go/esfroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
	"github.com/pegasusx/pegasusx/apps/backend-go/glroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/iam"
	"github.com/pegasusx/pegasusx/apps/backend-go/iamroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/orderroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/payout"
	"github.com/pegasusx/pegasusx/apps/backend-go/periodroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/periods"
	"github.com/pegasusx/pegasusx/apps/backend-go/recon"
	"github.com/pegasusx/pegasusx/apps/backend-go/reconroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/stocklots"
	"github.com/pegasusx/pegasusx/apps/backend-go/valuation"
	"github.com/pegasusx/pegasusx/apps/backend-go/valuationroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/warehouse"
	"github.com/pegasusx/pegasusx/apps/backend-go/warehouseroutes"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

func openInventorySpannerClient(t *testing.T) *spanner.Client {
	t.Helper()
	requireSpanner := strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1"
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if requireSpanner {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping live Spanner audit route inventory E2E test")
	}
	dbPath := "projects/pegasusx-local/instances/pegasusx-instance/databases/pegasusx-db"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, err := spanner.NewClient(ctx, dbPath,
		option.WithEndpoint(host),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		if requireSpanner {
			t.Fatalf("open spanner client: %v", err)
		}
		t.Skipf("spanner unavailable: %v", err)
	}
	if _, probeErr := client.Single().WithTimestampBound(spanner.StrongRead()).ReadRow(ctx, "AuditEvents", spanner.Key{"__inv_probe__", "2026-01-01", "__inv_ev__"}, []string{"EventId"}); probeErr != nil && spanner.ErrCode(probeErr) != codes.NotFound {
		client.Close()
		if requireSpanner {
			t.Fatalf("AuditEvents unavailable: %v", probeErr)
		}
		t.Skipf("AuditEvents unavailable: %v", probeErr)
	}
	t.Cleanup(client.Close)
	return client
}

func isAuditMiddlewareHandler(mw func(http.Handler) http.Handler, refPtr uintptr) bool {
	if mw == nil {
		return false
	}
	ptr := reflect.ValueOf(mw).Pointer()
	if refPtr != 0 && ptr == refPtr {
		return true
	}
	fn := runtime.FuncForPC(ptr)
	return fn != nil && strings.Contains(fn.Name(), "AuditMiddleware")
}

func TestE2E_100PercentMutatingRouteAuditInventory_SoDAndHashChain(t *testing.T) {
	client := openInventorySpannerClient(t)
	ctx := context.Background()

	stocklots.SetCycleCountsEnabled(true)
	t.Cleanup(func() { stocklots.SetCycleCountsEnabled(false) })

	suffix := uuid.NewString()[:8]
	supID := "sup-e2e-inv-" + suffix
	whID := "wh-e2e-inv-" + suffix
	retID := "ret-e2e-inv-" + suffix
	ordID := "ord-e2e-inv-" + suffix
	lotID := "lot-e2e-inv-" + suffix
	prodID := "prod-e2e-inv-" + suffix
	adjID := "adj-e2e-inv-" + suffix
	breakID := "brk-e2e-inv-" + suffix
	runID := "run-e2e-inv-" + suffix
	today := time.Now().UTC().Format("2006-01-02")
	jwtSecret := DefaultJWTSecret

	// Services backed by live Spanner emulator.
	auditSvc := audit.NewService(client)
	iamSvc := iam.NewService(client)
	iamSvc.SetAuditRecorder(bootstrap.NewIAMAuditRecorder(auditSvc))

	glSvc := gl.NewService(client)
	if err := glSvc.SeedDefaults(ctx, supID, "UZ", "UZS"); err != nil {
		t.Fatalf("gl SeedDefaults: %v", err)
	}
	reconSvc := recon.NewService(client)
	creditSvc := credit.NewService(credit.NewSpannerRepository(client))
	creditPolicySvc := credit.NewPolicyService(credit.NewSpannerPolicyRepository(client), creditSvc)
	payoutSvc := payout.NewService(payout.NewRepository(client))

	orderRepo := NewMemoryOrderRepo()
	orderRepo.SeedOrder(order.Order{
		OrderID:    ordID,
		SupplierID: supID,
		RetailerID: retID,
		Status:     order.StatusCompleted,
		Currency:   "UZS",
		TotalMinor: 500_000,
	})
	orderSvc := order.NewService(order.ServiceConfig{
		Repo:           orderRepo,
		JWTSecret:      jwtSecret,
		SeedSupplierID: supID,
		SpannerClient:  client,
	})
	orderSvc.SetDraftSoDCheck(bootstrap.CheckOrderDraftSoD(iamSvc))

	whSvc := warehouse.NewService(warehouse.ServiceConfig{
		Repo:      warehouse.NewSpannerRepository(client),
		JWTSecret: jwtSecret,
	})
	wmsHandler := &stocklots.Handler{
		Spanner:    client,
		SupplierID: supID,
		SoDCheck:   bootstrap.CheckStockAdjustmentSoD(iamSvc),
	}

	periodsSvc := periods.NewService(client)
	approvalsSvc := approvals.NewService(client)
	valuationSvc := valuation.NewService(client)
	esfSvc := esf.NewService(esf.ServiceConfig{Repo: esf.NewSpannerRepository(client)})
	_ = bootstrap.SetupWave3A(ctx, &bootstrap.Config{}, client, nil, &bootstrap.App{
		Audit:            auditSvc,
		IAM:              iamSvc,
		GL:               glSvc,
		OrderService:     orderSvc,
		WarehouseService: whSvc,
		Periods:          periodsSvc,
		Approvals:        approvalsSvc,
		Valuation:        valuationSvc,
		ESF:              esfSvc,
	}, nil)
	t.Cleanup(func() {
		gl.SetPeriodLockChecker(nil)
		stocklots.SetPickValuationHook(nil)
		stocklots.SetAdjustmentValuationHook(nil)
	})

	// Build unified router mirroring production NewHTTPRouter middleware + route groups.
	r := chi.NewRouter()
	r.Use(bootstrap.DevCORSMiddleware())
	r.Use(auth.SessionAuth(jwtSecret))
	r.Use(auth.AttachTenantFromClaims)
	r.Use(audit.AuditMiddleware(auditSvc))

	orderroutes.RegisterRoutes(r, orderroutes.Deps{Service: orderSvc})
	glroutes.RegisterRoutes(r, glroutes.Deps{Service: glSvc, JWTSecret: jwtSecret})
	periodroutes.RegisterRoutes(r, periodroutes.Deps{Service: periodsSvc, JWTSecret: jwtSecret})
	approvalroutes.RegisterRoutes(r, approvalroutes.Deps{Service: approvalsSvc, JWTSecret: jwtSecret})
	valuationroutes.RegisterRoutes(r, valuationroutes.Deps{Service: valuationSvc, JWTSecret: jwtSecret})
	esfroutes.RegisterRoutes(r, esfroutes.Deps{Service: esfSvc, JWTSecret: jwtSecret})
	reconroutes.RegisterRoutes(r, reconroutes.Deps{Service: reconSvc, JWTSecret: jwtSecret})
	warehouseroutes.RegisterRoutes(r, warehouseroutes.Deps{Service: whSvc, OrderService: orderSvc, WMSHandler: wmsHandler, JWTSecret: jwtSecret, Spanner: client})
	creditroutes.RegisterRoutes(r, creditroutes.Deps{Service: creditSvc, PolicyService: creditPolicySvc})
	payout.RegisterRoutes(r, &payout.Handlers{Svc: payoutSvc})
	auditroutes.RegisterRoutes(r, auditroutes.Deps{Service: auditSvc, JWTSecret: jwtSecret})
	iamroutes.RegisterRoutes(r, iamroutes.Deps{Service: iamSvc, JWTSecret: jwtSecret})

	// Seed stocklots adjustment + recon break rows in Spanner.
	_, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("SupplierInventoryV2", map[string]any{
			"SupplierId": supID, "WarehouseId": whID, "ProductId": prodID,
			"QuantityOnHand": int64(30), "QuantityReserved": int64(0), "ReorderThreshold": int64(5),
			"UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("StockLots", map[string]any{
			"LotId": lotID, "SupplierId": supID, "WarehouseId": whID, "ProductId": prodID,
			"LocationId": "bin-e2e", "QuantityOnHand": int64(30), "QuantityReserved": int64(0),
			"Status": "AVAILABLE", "ReceivedAt": spanner.CommitTimestamp,
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("InventoryAdjustments", map[string]any{
			"AdjustmentId": adjID, "WarehouseId": whID, "ProductId": prodID, "LotId": lotID,
			"DeltaQty": int64(4), "ReasonCode": "COUNT_DIFF", "Status": "PENDING",
			"ActorId": "maker-wh-" + suffix, "CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("ReconciliationRuns", map[string]any{
			"EntityId": supID, "RunId": runID, "PeriodId": "2026-11",
			"Mode": "MANUAL", "Status": "BREAKS_FOUND",
			"ChecksJson": "[]", "SummaryJson": "{}",
			"BreakCount": int64(1), "ExposureMinor": int64(5000),
			"TriggeredBy": "seed", "StartedAt": spanner.CommitTimestamp, "CompletedAt": time.Now().UTC(),
		}),
		spanner.InsertMap("ReconciliationBreaks", map[string]any{
			"EntityId": supID, "BreakId": breakID, "RunId": runID, "Dimension": "AR",
			"ObjectType": "ArInvoice", "ObjectId": "inv-" + suffix,
			"ExpectedMinor": int64(10000), "ActualMinor": int64(5000), "DeltaMinor": int64(5000),
			"Currency": "UZS", "Severity": "MEDIUM", "Status": "OPEN",
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
	})
	if err != nil {
		t.Fatalf("seed Spanner fixtures: %v", err)
	}

	var draftIDToClean string
	t.Cleanup(func() {
		cleanCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		muts := []*spanner.Mutation{
			spanner.Delete("AuditEvents", spanner.KeyRange{Start: spanner.Key{supID, today}, End: spanner.Key{supID, today}, Kind: spanner.ClosedClosed}),
			spanner.Delete("RolePermissions", spanner.KeyRange{Start: spanner.Key{supID}, End: spanner.Key{supID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("SupplierUserRoles", spanner.KeyRange{Start: spanner.Key{supID}, End: spanner.Key{supID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("SodPolicies", spanner.KeyRange{Start: spanner.Key{supID}, End: spanner.Key{supID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("SodViolations", spanner.KeyRange{Start: spanner.Key{supID}, End: spanner.Key{supID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("InventoryAdjustments", spanner.Key{adjID}),
			spanner.Delete("StockLots", spanner.Key{lotID}),
			spanner.Delete("SupplierInventoryV2", spanner.KeyRange{Start: spanner.Key{supID, whID}, End: spanner.Key{supID, whID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("ReconciliationBreaks", spanner.KeyRange{Start: spanner.Key{supID}, End: spanner.Key{supID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("ReconciliationRuns", spanner.KeyRange{Start: spanner.Key{supID}, End: spanner.Key{supID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("RetailerCreditProfiles", spanner.Key{retID, supID}),
			spanner.Delete("SupplierPayoutPolicies", spanner.Key{supID}),
			spanner.Delete("JournalPostings", spanner.KeyRange{Start: spanner.Key{supID}, End: spanner.Key{supID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("JournalEntries", spanner.KeyRange{Start: spanner.Key{supID}, End: spanner.Key{supID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("JournalIdempotency", spanner.KeyRange{Start: spanner.Key{supID}, End: spanner.Key{supID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("GLAccounts", spanner.KeyRange{Start: spanner.Key{supID}, End: spanner.Key{supID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("GlBalances", spanner.KeyRange{Start: spanner.Key{supID}, End: spanner.Key{supID}, Kind: spanner.ClosedClosed}),
		}
		if draftIDToClean != "" {
			muts = append(muts, spanner.Delete("DraftCreditNotes", spanner.Key{draftIDToClean}))
		}
		_, _ = client.Apply(cleanCtx, muts)
	})

	// 1. Walk 100% of registered POST/PUT/PATCH/DELETE routes via chi.Walk and verify AuditMiddleware coverage.
	refAuditMWPtr := reflect.ValueOf(audit.AuditMiddleware(auditSvc)).Pointer()
	mutatingCount := 0
	if walkErr := chi.Walk(r, func(method, route string, _ http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		switch method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			mutatingCount++
			found := false
			for _, mw := range middlewares {
				if isAuditMiddlewareHandler(mw, refAuditMWPtr) {
					found = true
					break
				}
			}
			if !found {
				for _, mw := range r.Middlewares() {
					if isAuditMiddlewareHandler(mw, refAuditMWPtr) {
						found = true
						break
					}
				}
			}
			if !found {
				t.Errorf("route %s %s missing audit.AuditMiddleware", method, route)
			}
		}
		return nil
	}); walkErr != nil {
		t.Fatalf("chi.Walk error: %v", walkErr)
	}
	if mutatingCount < 40 {
		t.Fatalf("expected >=40 mutating routes walked, got %d", mutatingCount)
	}

	ts := httptest.NewServer(r)
	defer ts.Close()

	issue := func(sub string, role auth.Role, supRole auth.Role, actorType, warehouseScope string) string {
		claims := auth.Claims{
			Subject:      sub,
			Role:         role,
			SupplierID:   supID,
			SupplierRole: supRole,
			ActorType:    actorType,
		}
		if warehouseScope != "" {
			claims.HomeNodeID = warehouseScope
			claims.HomeNodeType = auth.HomeNodeWarehouse
		}
		tok, err := auth.Issue(claims, auth.IssueOptions{Secret: jwtSecret, Issuer: DefaultJWTIssuer, TTL: time.Hour})
		if err != nil {
			t.Fatalf("issue token: %v", err)
		}
		return tok
	}

	acctTok := issue("acct-e2e-"+suffix, auth.RoleAdmin, auth.SupplierRoleAccountant, auth.ActorTypeHuman, "")
	finMgrTok := issue("finmgr-e2e-"+suffix, auth.RoleAdmin, auth.SupplierRoleFinanceManager, auth.ActorTypeHuman, "")
	credTok := issue("cred-e2e-"+suffix, auth.RoleAdmin, auth.SupplierRoleCreditOfficer, auth.ActorTypeHuman, "")
	ownerTok := issue("owner-e2e-"+suffix, auth.RoleAdmin, auth.SupplierRoleOwner, auth.ActorTypeHuman, "")
	makerWhTok := issue("maker-wh-"+suffix, auth.RoleWarehouseAdmin, auth.SupplierRoleWarehouseManager, auth.ActorTypeHuman, whID)
	checkerWhTok := issue("checker-wh-"+suffix, auth.RoleWarehouseAdmin, auth.SupplierRoleWarehouseManager, auth.ActorTypeHuman, whID)
	auditorTok := issue("aud-e2e-"+suffix, auth.RoleAdmin, auth.SupplierRoleAuditor, auth.ActorTypeHuman, whID)
	agentTok := issue("agent-e2e-"+suffix, auth.RoleAdmin, auth.SupplierRoleAgent, auth.ActorTypeAgent, whID)

	call := func(method, path string, body any, tok string) (int, []byte) {
		var buf bytes.Buffer
		if body != nil {
			_ = json.NewEncoder(&buf).Encode(body)
		}
		req, _ := http.NewRequest(method, ts.URL+path, &buf)
		req.Header.Set("Authorization", "Bearer "+tok)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", uuid.NewString())
		resp, err := ts.Client().Do(req)
		if err != nil {
			t.Fatalf("%s %s failed: %v", method, path, err)
		}
		defer resp.Body.Close()
		var out bytes.Buffer
		_, _ = out.ReadFrom(resp.Body)
		return resp.StatusCode, out.Bytes()
	}

	// Domain 1: order (credit note propose -> self-review 403 -> distinct checker 200).
	st, b := call(http.MethodPost, "/v1/finance/credit-note-drafts", map[string]any{
		"orderId":             ordID,
		"proposedAmountMinor": int64(25000),
		"reasonCode":          "DAMAGED",
	}, acctTok)
	if st != http.StatusCreated {
		t.Fatalf("[order] create credit-note draft = %d body=%s", st, b)
	}
	var cnResp order.DraftResponse
	_ = json.Unmarshal(b, &cnResp)
	draftIDToClean = cnResp.DraftID

	st, b = call(http.MethodPost, "/v1/finance/credit-note-drafts/review", map[string]any{
		"draftId": cnResp.DraftID, "action": "APPROVE",
	}, acctTok)
	if st != http.StatusForbidden {
		t.Fatalf("[order] self-review expected 403, got %d body=%s", st, b)
	}
	st, b = call(http.MethodPost, "/v1/finance/credit-note-drafts/review", map[string]any{
		"draftId": cnResp.DraftID, "action": "APPROVE",
	}, finMgrTok)
	if st != http.StatusOK {
		t.Fatalf("[order] checker review expected 200, got %d body=%s", st, b)
	}

	// Domain 2: gl (manual post by ACCOUNTANT -> 201, ACCOUNTANT reverse -> 403, FINANCE_MANAGER reverse -> 201).
	st, b = call(http.MethodPost, "/v1/finance/journal/manual", map[string]any{
		"source_id": "inv-gl-" + suffix, "currency": "UZS", "memo": "route inventory gl",
		"postings": []map[string]any{
			{"account_code": "CASH:OFFICE", "direction": "DEBIT", "amount_minor": 40000},
			{"account_code": "REV:SALES", "direction": "CREDIT", "amount_minor": 40000},
		},
	}, acctTok)
	if st != http.StatusCreated {
		t.Fatalf("[gl] manual post = %d body=%s", st, b)
	}
	var glRes gl.JournalEntryResult
	_ = json.Unmarshal(b, &glRes)
	if stRevAcct, bRevAcct := call(http.MethodPost, "/v1/finance/journal/"+glRes.EntryID+"/reverse", map[string]any{"reason": "acct try"}, acctTok); stRevAcct != http.StatusForbidden {
		t.Fatalf("[gl] accountant reverse expected 403, got %d body=%s", stRevAcct, bRevAcct)
	}
	if stRevFin, bRevFin := call(http.MethodPost, "/v1/finance/journal/"+glRes.EntryID+"/reverse", map[string]any{"reason": "finmgr reverse"}, finMgrTok); stRevFin != http.StatusCreated {
		t.Fatalf("[gl] finance_manager reverse expected 201, got %d body=%s", stRevFin, bRevFin)
	}

	// Domain 3: recon (POST /run -> 200, PATCH /breaks/{id} -> 200).
	if st, b = call(http.MethodPost, "/v1/finance/recon/run", map[string]any{"period_id": "2026-11"}, finMgrTok); st != http.StatusOK {
		t.Fatalf("[recon] run = %d body=%s", st, b)
	}
	if st, b = call(http.MethodPatch, "/v1/finance/recon/breaks/"+breakID, map[string]any{"action": "RESOLVE", "note": "matched"}, finMgrTok); st != http.StatusOK {
		t.Fatalf("[recon] patch break = %d body=%s", st, b)
	}

	// Domain 4: stocklots (self-approve -> 403 sod_violation, distinct checker -> 200).
	// Note: stocklots reads warehouse_id from query since WarehouseOps claims aren't embedded in Bearer JWT for RoleWarehouseAdmin unless query param is set.
	if st, b = call(http.MethodPost, "/v1/warehouse/ops/inventory-adjustments/"+adjID+"/approve?warehouse_id="+whID, map[string]any{}, makerWhTok); st != http.StatusForbidden || !strings.Contains(string(b), "sod_violation") {
		t.Fatalf("[stocklots] maker self-approve expected 403 sod_violation, got %d body=%s", st, b)
	}
	if st, b = call(http.MethodPost, "/v1/warehouse/ops/inventory-adjustments/"+adjID+"/approve?warehouse_id="+whID, map[string]any{}, checkerWhTok); st != http.StatusOK {
		t.Fatalf("[stocklots] checker approve expected 200, got %d body=%s", st, b)
	}

	// Domain 5: credit (PATCH /v1/supplier/retailer-credit-profile -> 200).
	if st, b = call(http.MethodPatch, "/v1/supplier/retailer-credit-profile", map[string]any{
		"retailer_id": retID, "credit_limit_minor": int64(750000), "status": "ACTIVE", "reason": "initial limit",
	}, credTok); st != http.StatusOK {
		t.Fatalf("[credit] patch profile = %d body=%s", st, b)
	}

	// Domain 6: payout (PATCH /v1/supplier/payout-policy -> 200, POST /v1/supplier/payouts/batches -> 409 nothing_payable logged).
	if st, b = call(http.MethodPatch, "/v1/supplier/payout-policy", map[string]any{
		"payout_mode": payout.PayoutModeHQSupplier, "reason": "e2e route verification",
	}, ownerTok); st != http.StatusOK {
		t.Fatalf("[payout] patch policy = %d body=%s", st, b)
	}
	if st, _ = call(http.MethodPost, "/v1/supplier/payouts/batches", map[string]any{
		"period_start": "2026-11-01", "period_end": "2026-11-30",
	}, finMgrTok); st != http.StatusOK && st != http.StatusConflict {
		t.Fatalf("[payout] generate batch unexpected status=%d", st)
	}

	// Domain 7: iam (assign ACCOUNTANT -> 200, conflicting FINANCE_MANAGER -> 403 sod_violation, PUT policy -> 200).
	targetUser := "u-iam-" + suffix
	if st, b = call(http.MethodPost, "/v1/iam/users/"+targetUser+"/roles", map[string]any{
		"role_name": auth.SupplierRoleAccountant, "action": iam.RoleActionAssign,
	}, ownerTok); st != http.StatusOK {
		t.Fatalf("[iam] assign accountant = %d body=%s", st, b)
	}
	if st, b = call(http.MethodPost, "/v1/iam/users/"+targetUser+"/roles", map[string]any{
		"role_name": auth.SupplierRoleFinanceManager, "action": iam.RoleActionAssign,
	}, ownerTok); st != http.StatusForbidden || !strings.Contains(string(b), "sod_violation") {
		t.Fatalf("[iam] conflicting role expected 403 sod_violation, got %d body=%s", st, b)
	}
	if st, b = call(http.MethodPut, "/v1/iam/sod-policies/sod_credit_limit_over_limit", map[string]any{
		"enforce_same_object": true,
	}, ownerTok); st != http.StatusOK {
		t.Fatalf("[iam] update sod policy = %d body=%s", st, b)
	}

	// Negative checks: AGENT approvals -> 403; AUDITOR mutations -> 403 while read -> 200.
	agentBlocked := []struct {
		method, path string
		body         any
	}{
		{http.MethodPost, "/v1/finance/drafts/review", map[string]any{"draftId": cnResp.DraftID, "action": "APPROVE"}},
		{http.MethodPost, "/v1/finance/journal/" + glRes.EntryID + "/reverse", map[string]any{"reason": "agent"}},
		{http.MethodPost, "/v1/warehouse/ops/inventory-adjustments/" + adjID + "/approve?warehouse_id=" + whID, map[string]any{}},
		{http.MethodPost, "/v1/supplier/payouts/batches/b1/mark-paid", map[string]any{}},
	}
	for _, tc := range agentBlocked {
		if st, _ = call(tc.method, tc.path, tc.body, agentTok); st != http.StatusForbidden {
			t.Fatalf("[agent] %s %s expected 403, got %d", tc.method, tc.path, st)
		}
	}

	auditorBlocked := []struct {
		method, path string
		body         any
	}{
		{http.MethodPost, "/v1/finance/journal/manual", map[string]any{"source_id": "x"}},
		{http.MethodPost, "/v1/iam/users/u-aud/roles", map[string]any{"role_name": "ACCOUNTANT"}},
		{http.MethodPatch, "/v1/supplier/retailer-credit-profile", map[string]any{"retailer_id": retID}},
		{http.MethodPost, "/v1/finance/recon/run", map[string]any{"period_id": "2026-11"}},
	}
	for _, tc := range auditorBlocked {
		if st, _ = call(tc.method, tc.path, tc.body, auditorTok); st != http.StatusForbidden {
			t.Fatalf("[auditor] %s %s expected 403, got %d", tc.method, tc.path, st)
		}
	}
	if st, b = call(http.MethodGet, "/v1/audit/events?limit=25", nil, auditorTok); st != http.StatusOK {
		t.Fatalf("[auditor] GET /v1/audit/events expected 200, got %d body=%s", st, b)
	}

	// Confirm SodViolations recorded.
	if st, b = call(http.MethodGet, "/v1/iam/sod-violations?limit=20", nil, ownerTok); st != http.StatusOK {
		t.Fatalf("GET /v1/iam/sod-violations = %d body=%s", st, b)
	}
	var violResp struct {
		Count int `json:"count"`
	}
	_ = json.Unmarshal(b, &violResp)
	if violResp.Count < 2 {
		t.Fatalf("expected >=2 SodViolations recorded (stocklots self-approve + iam static conflict), got %d body=%s", violResp.Count, b)
	}

	// Verify continuous unbroken hash chain across ALL 7 domains!
	st, b = call(http.MethodGet, "/v1/audit/verify?date="+today, nil, ownerTok)
	if st != http.StatusOK {
		t.Fatalf("GET /v1/audit/verify = %d body=%s", st, b)
	}
	var vr audit.ChainVerifyResult
	if err := json.Unmarshal(b, &vr); err != nil {
		t.Fatal(err)
	}
	if !vr.Valid || vr.EventCount < 10 {
		t.Fatalf("expected unbroken hash chain (>=10 events), got %+v", vr)
	}
}
