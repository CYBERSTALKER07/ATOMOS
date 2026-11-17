package e2ep0

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/credit"
	"github.com/pegasusx/pegasusx/apps/backend-go/forecastv2"
	"github.com/pegasusx/pegasusx/apps/backend-go/proposals"
	"github.com/pegasusx/pegasusx/apps/backend-go/replenishment"
	"github.com/pegasusx/pegasusx/apps/backend-go/roi"
)

func TestWave3B_FullStack_Forecast_CreditV2_Proposals_ROI_LiveSpanner(t *testing.T) {
	requireSpanner := strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1"
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		if requireSpanner {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST unset; skipping Wave 3B E2E test")
	}

	client := openInventorySpannerClient(t)
	ctx := context.Background()
	ro := client.ReadOnlyTransaction()
	_, pErr := ro.ReadRow(ctx, "Proposals", spanner.Key{"__w3b_probe__", "__probe__"}, []string{"Status"})
	ro.Close()
	if pErr != nil && spanner.ErrCode(pErr) != codes.NotFound {
		t.Fatalf("Proposals table unavailable: %v", pErr)
	}

	t.Setenv("PEGASUSX_ENV", "local")
	t.Setenv("ENFORCE_TENANT_CONTEXT", "1")
	t.Setenv("CSRF_ENFORCE", "0")
	t.Setenv("CREDIT_ML_SCORER_URL", "")
	t.Setenv("CREDIT_ML_SCORER_REQUIRED", "")
	t.Setenv("GEMINI_EXPLAINER_REQUIRED", "")

	tag := strings.ReplaceAll(uuid.NewString(), "-", "")[:8]
	entityID := "sup-w3b-" + tag
	otherConsentSup := "sup-consent-" + tag
	noConsentSup := "sup-noconsent-" + tag
	retID := "ret-w3b-" + tag
	wh1 := "wh1-w3b-" + tag
	wh2 := "wh2-w3b-" + tag
	facID := "fac-w3b-" + tag
	sku1 := "sku1-w3b-" + tag
	sku2 := "sku2-w3b-" + tag
	insID := "ins-w3b-" + tag
	invOverdueID := "inv-od-" + tag
	invDisputedID := "inv-disp-" + tag
	lotID := "lot-w3b-" + tag
	schedID := "fsched-w3b-" + tag
	bmID := "bm-w3b-" + tag
	today := time.Now().UTC().Format("2006-01-02")
	now := time.Now().UTC()

	h := NewTestHarnessWithOptions(t, HarnessOptions{SpannerClient: client})
	if h.ForecastV2Svc == nil || h.ReplenishmentSuggestions == nil || h.CreditSvc == nil || h.ProposalsSvc == nil || h.ROISvc == nil {
		t.Fatalf("expected all Wave 3B services wired on TestHarness")
	}

	t.Cleanup(func() {
		bg := context.Background()
		_, _ = client.Apply(bg, []*spanner.Mutation{
			spanner.Delete("Proposals", spanner.KeyRange{Start: spanner.Key{entityID}, End: spanner.Key{entityID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("RoiBaselines", spanner.KeyRange{Start: spanner.Key{entityID}, End: spanner.Key{entityID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("RoiEvents", spanner.KeyRange{Start: spanner.Key{entityID}, End: spanner.Key{entityID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("ForecastRuns", spanner.KeyRange{Start: spanner.Key{entityID}, End: spanner.Key{entityID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("ForecastValues", spanner.KeyRange{Start: spanner.Key{entityID}, End: spanner.Key{entityID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("ForecastExplanations", spanner.KeyRange{Start: spanner.Key{entityID}, End: spanner.Key{entityID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("ApprovalDecisions", spanner.KeyRange{Start: spanner.Key{entityID}, End: spanner.Key{entityID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("ApprovalRequests", spanner.KeyRange{Start: spanner.Key{entityID}, End: spanner.Key{entityID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("ApprovalPolicies", spanner.KeyRange{Start: spanner.Key{entityID}, End: spanner.Key{entityID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("CreditScoreHistory", spanner.KeyRange{Start: spanner.Key{retID}, End: spanner.Key{retID}, Kind: spanner.ClosedClosed}),
			spanner.Delete("RetailerCreditScores", spanner.Key{retID}),
			spanner.Delete("RetailerCreditProfiles", spanner.Key{retID, entityID}),
			spanner.Delete("RetailerCreditProfiles", spanner.Key{retID, otherConsentSup}),
			spanner.Delete("RetailerPaymentTerms", spanner.Key{retID, entityID}),
			spanner.Delete("RetailerPaymentTerms", spanner.Key{retID, otherConsentSup}),
			spanner.Delete("RetailerPaymentTerms", spanner.Key{retID, noConsentSup}),
			spanner.Delete("ReorderSuggestions", spanner.Key{retID, sku1}),
			spanner.Delete("ReplenishmentInsights", spanner.Key{insID}),
			spanner.Delete("StockLots", spanner.Key{lotID}),
			spanner.Delete("ArInvoices", spanner.Key{invOverdueID}),
			spanner.Delete("ArInvoices", spanner.Key{invDisputedID}),
			spanner.Delete("BillingFeeSchedules", spanner.Key{schedID}),
			spanner.Delete("BillingMeterEvents", spanner.Key{bmID}),
			spanner.Delete("Products", spanner.Key{sku1}),
			spanner.Delete("Products", spanner.Key{sku2}),
			spanner.Delete("Warehouses", spanner.Key{wh1}),
			spanner.Delete("Warehouses", spanner.Key{wh2}),
			spanner.Delete("Retailers", spanner.Key{retID}),
		})
	})

	// Seed realistic Spanner rows backing M1.5, M1.6, M1.7, and M1.8.
	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdateMap("Warehouses", map[string]any{
			"WarehouseId": wh1, "SupplierId": entityID, "Name": "Tashkent Hub", "CoverageRadiusKm": float64(50),
			"PrimaryFactoryId": facID, "IsActive": true, "IsOnShift": true,
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertOrUpdateMap("Warehouses", map[string]any{
			"WarehouseId": wh2, "SupplierId": entityID, "Name": "Samarkand WH", "CoverageRadiusKm": float64(50),
			"PrimaryFactoryId": facID, "IsActive": true, "IsOnShift": true,
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertOrUpdateMap("Products", map[string]any{
			"ProductId": sku1, "SupplierId": entityID, "CategoryId": "dairy",
			"Name": "Butter 200g", "PriceMinor": int64(25000), "Currency": "UZS",
			"StockQuantity": int64(4), "IsActive": true, "Version": int64(1),
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertOrUpdateMap("Products", map[string]any{
			"ProductId": sku2, "SupplierId": entityID, "CategoryId": "dairy",
			"Name": "Yogurt 500ml", "PriceMinor": int64(12000), "Currency": "UZS",
			"StockQuantity": int64(60), "IsActive": true, "Version": int64(1),
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertOrUpdateMap("Retailers", map[string]any{
			"RetailerId": retID, "Phone": "+998901112233", "Name": "Baraka Market", "CountryCode": "UZ",
			"CreatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertOrUpdateMap("RetailerCreditProfiles", map[string]any{
			"RetailerId": retID, "SupplierId": entityID,
			"CreditLimitMinor": int64(1000000), "CurrentBalanceMinor": int64(200000), "ReservedMinor": int64(0),
			"AvailableCreditMinor": int64(800000), "RiskScore": int64(75), "DelinquencyCount": int64(0),
			"PrepayOnlyLocked": false, "Status": "ACTIVE", "Version": int64(1),
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertOrUpdateMap("RetailerCreditProfiles", map[string]any{
			"RetailerId": retID, "SupplierId": otherConsentSup,
			"CreditLimitMinor": int64(800000), "CurrentBalanceMinor": int64(100000), "ReservedMinor": int64(0),
			"AvailableCreditMinor": int64(700000), "RiskScore": int64(82), "DelinquencyCount": int64(0),
			"PrepayOnlyLocked": false, "Status": "ACTIVE", "Version": int64(1),
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertOrUpdateMap("RetailerPaymentTerms", map[string]any{
			"RetailerId": retID, "SupplierId": entityID,
			"CreditEnabled": true, "TermsDays": int64(30), "GracePeriodDays": int64(3),
			"CreditLimitMinor": int64(1000000), "UseGlobalDefaults": false,
			"ConsentVerified": true, "Version": int64(1), "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertOrUpdateMap("RetailerPaymentTerms", map[string]any{
			"RetailerId": retID, "SupplierId": otherConsentSup,
			"CreditEnabled": true, "TermsDays": int64(30), "GracePeriodDays": int64(0),
			"CreditLimitMinor": int64(800000), "UseGlobalDefaults": false,
			"ConsentVerified": true, "Version": int64(1), "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertOrUpdateMap("RetailerPaymentTerms", map[string]any{
			"RetailerId": retID, "SupplierId": noConsentSup,
			"CreditEnabled": true, "TermsDays": int64(14), "GracePeriodDays": int64(0),
			"CreditLimitMinor": int64(5000000), "UseGlobalDefaults": false,
			"ConsentVerified": false, "Version": int64(1), "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertOrUpdateMap("ReorderSuggestions", map[string]any{
			"RetailerId": retID, "Sku": sku1,
			"SuggestedQty": int64(24), "AdjustedDemand": float64(12.0), "CurrentStock": int64(3),
			"InFlightQty": int64(0), "SafetyStock": float64(8.0),
			"SuggestedByDate": civil.DateOf(now.AddDate(0, 0, 2)), "ComputedAt": now, "Status": "OPEN",
		}),
		spanner.InsertOrUpdateMap("ReplenishmentInsights", map[string]any{
			"InsightId": insID, "WarehouseId": wh1, "ProductId": sku1, "SupplierId": entityID,
			"CurrentStock": int64(2), "SuggestedQuantity": int64(40), "UrgencyLevel": "CRITICAL",
			"ReasonCode": "LOW_STOCK", "Status": "PENDING", "TargetFactoryId": facID,
			"CreatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertOrUpdateMap("ArInvoices", map[string]any{
			"InvoiceId": invOverdueID, "SupplierId": entityID, "RetailerId": retID, "OrderId": "ord-od-" + tag,
			"Status": "OPEN", "PrincipalMinor": int64(600000), "BalanceMinor": int64(600000),
			"Currency": "UZS", "CreditLeaveAt": now.Add(-25 * 24 * time.Hour), "DueAt": now.Add(-10 * 24 * time.Hour),
			"TermsDays": int64(15), "GracePeriodDays": int64(0), "DunningStep": int64(1),
			"Version": int64(1), "CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertOrUpdateMap("ArInvoices", map[string]any{
			"InvoiceId": invDisputedID, "SupplierId": otherConsentSup, "RetailerId": retID, "OrderId": "ord-disp-" + tag,
			"Status": "DISPUTED", "PrincipalMinor": int64(350000), "BalanceMinor": int64(350000),
			"Currency": "UZS", "CreditLeaveAt": now.Add(-40 * 24 * time.Hour), "DueAt": now.Add(-20 * 24 * time.Hour),
			"TermsDays": int64(20), "GracePeriodDays": int64(0), "DunningStep": int64(0),
			"Version": int64(1), "CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertOrUpdateMap("StockLots", map[string]any{
			"LotId": lotID, "SupplierId": entityID, "WarehouseId": wh1, "ProductId": sku2, "LocationId": "bin-b2",
			"ExpiryDate": civil.DateOf(now.Add(5 * 24 * time.Hour)), "QuantityOnHand": int64(25),
			"QuantityReserved": int64(0), "ReceivedAt": spanner.CommitTimestamp, "Status": "AVAILABLE",
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertOrUpdateMap("BillingFeeSchedules", map[string]any{
			"FeeScheduleId": schedID, "SupplierId": entityID, "Tier": "STANDARD",
			"PerOrderMinor": int64(25000), "GmvBps": int64(100), "MonthlySubscriptionMinor": int64(100000),
			"Currency": "UZS", "EffectiveFrom": now.Add(-48 * time.Hour), "CreatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertOrUpdateMap("BillingMeterEvents", map[string]any{
			"EventId": bmID, "SupplierId": entityID, "OrderId": "ord-bm-" + tag,
			"MeterType": "ORDER_COMPLETED", "Amount": int64(2000000), "ProcessedAt": now.Add(-2 * time.Hour),
		}),
	}); err != nil {
		t.Fatalf("seed Wave 3B Spanner fixtures: %v", err)
	}

	issueUser := func(sub string, supRole auth.Role, actorType string, perms ...string) string {
		tok, err := auth.Issue(auth.Claims{
			Subject:       sub,
			Role:          auth.RoleAdmin,
			SupplierID:    entityID,
			SupplierRole:  supRole,
			SupplierPerms: perms,
			ActorType:     actorType,
		}, auth.IssueOptions{Secret: h.JWTSecret, Issuer: h.JWTIssuer, TTL: time.Hour})
		if err != nil {
			t.Fatalf("auth.Issue: %v", err)
		}
		return tok
	}

	ownerTok := issueUser("owner-"+tag, auth.SupplierRoleOwner, auth.ActorTypeHuman)
	finTok := issueUser("fin-"+tag, auth.SupplierRoleFinanceManager, auth.ActorTypeHuman,
		auth.PermConfigManage, auth.PermAgentProposalApprove, auth.PermCreditLimitChange,
		auth.PermDraftApprove, auth.PermAuditRead, auth.PermReconManage)
	agentTok := issueUser("agent-"+tag, auth.SupplierRoleAgent, auth.ActorTypeAgent)
	auditorTok := issueUser("auditor-"+tag, auth.SupplierRoleAuditor, auth.ActorTypeHuman, auth.PermAuditRead)

	doReq := func(tok, method, path string, body any) *httptest.ResponseRecorder {
		var raw []byte
		if body != nil {
			raw, _ = json.Marshal(body)
		}
		req := httptest.NewRequest(method, path, bytes.NewReader(raw))
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		req.Header.Set("Authorization", "Bearer "+tok)
		req.Header.Set("Idempotency-Key", uuid.NewString())
		rec := httptest.NewRecorder()
		h.Router.ServeHTTP(rec, req)
		return rec
	}

	// --- 1. M1.5 Forecast v2 & 1-tap Suggestion Approval ---
	fRunRec := doReq(ownerTok, http.MethodPost, "/v1/planning/forecast/run", map[string]any{
		"horizon_days":       7,
		"promo_discount_bps": 1000,
		"series": []map[string]any{
			{
				"location_id":        wh1,
				"sku":                sku1,
				"product_id":         sku1,
				"daily_units_milli":  []int64{10000, 12000, 11000, 13000, 14000, 12000, 15000, 11000, 13000, 12000, 14000, 16000, 13000, 15000},
				"promo_discount_bps": 1000,
				"active_promo":       true,
			},
		},
	})
	if fRunRec.Code != http.StatusCreated {
		t.Fatalf("POST /v1/planning/forecast/run failed: %d %s", fRunRec.Code, fRunRec.Body.String())
	}
	var fOut forecastv2.RunForecastOutput
	if err := json.Unmarshal(fRunRec.Body.Bytes(), &fOut); err != nil {
		t.Fatalf("unmarshal RunForecastOutput: %v", err)
	}
	if fOut.Run.RunID == "" || fOut.Run.WinnerModel == "" || len(fOut.Forecasts) == 0 || len(fOut.Explanations) == 0 {
		t.Fatalf("unexpected RunForecastOutput: %+v", fOut)
	}
	fGetRec := doReq(auditorTok, http.MethodGet, "/v1/planning/forecast?run_id="+fOut.Run.RunID, nil)
	if fGetRec.Code != http.StatusOK {
		t.Fatalf("GET /v1/planning/forecast failed: %d %s", fGetRec.Code, fGetRec.Body.String())
	}

	expID := fOut.Explanations[0].ExplanationID
	if rec := doReq(agentTok, http.MethodPost, "/v1/planning/forecast/explanations/"+expID+"/approve", map[string]any{}); rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for AGENT explanation approve, got %d %s", rec.Code, rec.Body.String())
	}
	if rec := doReq(auditorTok, http.MethodPost, "/v1/planning/forecast/explanations/"+expID+"/approve", map[string]any{}); rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for AUDITOR explanation approve, got %d %s", rec.Code, rec.Body.String())
	}
	if rec := doReq(ownerTok, http.MethodPost, "/v1/planning/forecast/explanations/"+expID+"/approve", map[string]any{
		"actual_qty_milli": int64(15000),
		"comment":          "approved baseline promo uplift",
	}); rec.Code != http.StatusOK {
		t.Fatalf("expected 200 owner explanation approve, got %d %s", rec.Code, rec.Body.String())
	}

	sugKey := retID + ":" + sku1
	if rec := doReq(agentTok, http.MethodPost, "/v1/replenishment/suggestions/"+sugKey+"/approve", map[string]any{"mode": "approve_only"}); rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for AGENT suggestion approve, got %d %s", rec.Code, rec.Body.String())
	}
	sugApproveRec := doReq(ownerTok, http.MethodPost, "/v1/replenishment/suggestions/"+sugKey+"/approve", map[string]any{"qty": 30, "mode": "approve_only"})
	if sugApproveRec.Code != http.StatusOK {
		t.Fatalf("expected 200 suggestion approve, got %d %s", sugApproveRec.Code, sugApproveRec.Body.String())
	}
	var sugResp replenishment.ApproveSuggestionResponse
	if err := json.Unmarshal(sugApproveRec.Body.Bytes(), &sugResp); err != nil || sugResp.Status != "APPROVED" || sugResp.ApprovedQty != 30 {
		t.Fatalf("unexpected suggestion approve response: %+v err=%v body=%s", sugResp, err, sugApproveRec.Body.String())
	}

	// --- 2. M1.6 Credit v2 Scoring (Consent & Dispute Isolation) + CEL Limit Approval ---
	if rec := doReq(finTok, http.MethodPost, "/v1/approvals/policies", map[string]any{
		"name":           "Credit Limit Change > 500k",
		"object_type":    "CreditLimit",
		"action":         "change",
		"condition_expr": "amount_minor > 500000",
		"steps": []map[string]any{
			{"step_index": 0, "name": "Owner", "allowed_roles": []string{auth.SupplierRoleOwner}, "required_approvers": 1},
		},
	}); rec.Code != http.StatusCreated {
		t.Fatalf("create CreditLimit CEL policy: %d %s", rec.Code, rec.Body.String())
	}

	scoreRec := doReq(ownerTok, http.MethodGet, "/v1/credit/retailers/"+retID+"/score?scope=BOTH&recompute=true", nil)
	if scoreRec.Code != http.StatusOK {
		t.Fatalf("GET /v1/credit/retailers/{id}/score failed: %d %s", scoreRec.Code, scoreRec.Body.String())
	}
	var scoreView credit.RetailerScoreView
	if err := json.Unmarshal(scoreRec.Body.Bytes(), &scoreView); err != nil {
		t.Fatalf("unmarshal RetailerScoreView: %v", err)
	}
	if scoreView.RelationshipScore == nil || scoreView.NetworkScore == nil {
		t.Fatalf("expected both RelationshipScore and NetworkScore, got %+v", scoreView)
	}
	if !scoreView.NetworkScore.ConsentVerified || scoreView.NetworkScore.DisputesIsolatedCount < 1 {
		t.Fatalf("expected NetworkScore consent_verified=true and disputes_isolated_count>=1, got %+v", scoreView.NetworkScore)
	}
	if len(scoreView.RelationshipScore.ReasonCodes) == 0 {
		t.Fatalf("expected non-empty ReasonCodes on RelationshipScore")
	}

	portRec := doReq(auditorTok, http.MethodGet, "/v1/credit/portfolio", nil)
	if portRec.Code != http.StatusOK || !strings.Contains(portRec.Body.String(), retID) {
		t.Fatalf("GET /v1/credit/portfolio failed: %d %s", portRec.Code, portRec.Body.String())
	}

	if rec := doReq(agentTok, http.MethodPost, "/v1/credit/limits/"+retID, map[string]any{
		"new_limit_minor": 2000000, "reason": "agent test",
	}); rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for AGENT credit limit post, got %d %s", rec.Code, rec.Body.String())
	}

	limReq := doReq(finTok, http.MethodPost, "/v1/credit/limits/"+retID, map[string]any{
		"new_limit_minor": 2000000,
		"reason":          "seasonal expansion",
	})
	if limReq.Code != http.StatusAccepted {
		t.Fatalf("expected 202 APPROVAL_REQUIRED for CreditLimit delta>500k, got %d %s", limReq.Code, limReq.Body.String())
	}
	var limPending credit.LimitChangeOutput
	if err := json.Unmarshal(limReq.Body.Bytes(), &limPending); err != nil || !limPending.ApprovalRequired || limPending.ApprovalRequestID == "" {
		t.Fatalf("unexpected limit pending response: %+v body=%s", limPending, limReq.Body.String())
	}

	if rec := doReq(finTok, http.MethodPost, "/v1/approvals/"+limPending.ApprovalRequestID+"/approve", map[string]any{}); rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 SoD self-approve block on credit limit, got %d %s", rec.Code, rec.Body.String())
	}
	if rec := doReq(agentTok, http.MethodPost, "/v1/approvals/"+limPending.ApprovalRequestID+"/approve", map[string]any{}); rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 AGENT approve block on credit limit, got %d %s", rec.Code, rec.Body.String())
	}
	if rec := doReq(ownerTok, http.MethodPost, "/v1/approvals/"+limPending.ApprovalRequestID+"/approve", map[string]any{"comment": "owner approved limit"}); rec.Code != http.StatusOK {
		t.Fatalf("owner approve credit limit request: %d %s", rec.Code, rec.Body.String())
	}

	// Tampered payload with approved ID must fail 409 payload_hash_mismatch!
	if rec := doReq(finTok, http.MethodPost, "/v1/credit/limits/"+retID, map[string]any{
		"new_limit_minor": 2500000,
		"reason":          "seasonal expansion",
		"approval_id":     limPending.ApprovalRequestID,
	}); rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 payload_hash_mismatch on tampered credit limit, got %d %s", rec.Code, rec.Body.String())
	}

	// Exact approved payload applies limit cleanly -> 200 APPLIED!
	limApplyRec := doReq(finTok, http.MethodPost, "/v1/credit/limits/"+retID, map[string]any{
		"new_limit_minor": 2000000,
		"reason":          "seasonal expansion",
		"approval_id":     limPending.ApprovalRequestID,
	})
	if limApplyRec.Code != http.StatusOK {
		t.Fatalf("expected 200 APPLIED on approved credit limit retry, got %d %s", limApplyRec.Code, limApplyRec.Body.String())
	}

	// --- 3. M1.7 3 Governed L1 Agents, Edit+Approve, Target Lock, Stale Precondition, Execution -> ROI ---
	runRec := doReq(ownerTok, http.MethodPost, "/v1/proposals/agents/ALL/run", map[string]any{})
	if runRec.Code != http.StatusOK {
		t.Fatalf("POST /v1/proposals/agents/ALL/run failed: %d %s", runRec.Code, runRec.Body.String())
	}
	var agentRunResp struct {
		CreatedCount int                  `json:"created_count"`
		Proposals    []proposals.Proposal `json:"proposals"`
	}
	if err := json.Unmarshal(runRec.Body.Bytes(), &agentRunResp); err != nil {
		t.Fatalf("unmarshal agent run response: %v", err)
	}
	if agentRunResp.CreatedCount != 3 {
		t.Fatalf("expected 3 created proposals from ALL agents, got %d (%s)", agentRunResp.CreatedCount, runRec.Body.String())
	}

	var reorderProp, collectionsProp, expiryProp proposals.Proposal
	for _, p := range agentRunResp.Proposals {
		if p.Status != proposals.StatusPending || p.ApprovalRequestID == "" || len(p.EvidenceRefs) == 0 {
			t.Fatalf("expected PENDING proposal with approval_request_id & evidence_refs, got %+v", p)
		}
		switch p.ProposalType {
		case proposals.TypeReorder:
			reorderProp = p
		case proposals.TypeCollections:
			collectionsProp = p
		case proposals.TypeExpiryRebalance:
			expiryProp = p
		}
	}
	if reorderProp.ProposalID == "" || collectionsProp.ProposalID == "" || expiryProp.ProposalID == "" {
		t.Fatalf("missing one of 3 expected L1 proposal types: %+v", agentRunResp.Proposals)
	}
	if !strings.Contains(collectionsProp.PayloadJSON, "draft_uz") || !strings.Contains(collectionsProp.PayloadJSON, "draft_ru") {
		t.Fatalf("expected UZ and RU message drafts in collections payload, got %s", collectionsProp.PayloadJSON)
	}

	// Agent & Auditor must be blocked (403) on approve/reject/execute.
	if rec := doReq(agentTok, http.MethodPost, "/v1/proposals/"+collectionsProp.ProposalID+"/approve", map[string]any{}); rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 AGENT approve proposal, got %d %s", rec.Code, rec.Body.String())
	}
	if rec := doReq(auditorTok, http.MethodPost, "/v1/proposals/"+collectionsProp.ProposalID+"/approve", map[string]any{}); rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 AUDITOR approve proposal, got %d %s", rec.Code, rec.Body.String())
	}

	// Human edits & approves Collections proposal -> PayloadHash re-pinned!
	origCollHash := collectionsProp.PayloadHash
	editedPayload := map[string]any{}
	for k, v := range collectionsProp.Payload {
		editedPayload[k] = v
	}
	editedPayload["preferred_lang"] = "ru"
	editedPayload["dunning_step"] = int64(2)
	approveCollRec := doReq(ownerTok, http.MethodPost, "/v1/proposals/"+collectionsProp.ProposalID+"/approve", map[string]any{
		"edited_payload": editedPayload,
		"comment":        "switch preferred outreach to RU",
	})
	if approveCollRec.Code != http.StatusOK {
		t.Fatalf("approve+edit collections proposal failed: %d %s", approveCollRec.Code, approveCollRec.Body.String())
	}
	var approvedCollWrap struct {
		Proposal proposals.Proposal `json:"proposal"`
	}
	_ = json.Unmarshal(approveCollRec.Body.Bytes(), &approvedCollWrap)
	if approvedCollWrap.Proposal.Status != proposals.StatusApproved || approvedCollWrap.Proposal.PayloadHash == origCollHash {
		t.Fatalf("expected APPROVED with updated PayloadHash, got status=%s hash=%s", approvedCollWrap.Proposal.Status, approvedCollWrap.Proposal.PayloadHash)
	}

	// Stale hash execute must fail 409!
	if rec := doReq(ownerTok, http.MethodPost, "/v1/proposals/"+collectionsProp.ProposalID+"/execute", map[string]any{
		"payload_hash": origCollHash,
	}); rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 payload_hash_mismatch with stale hash, got %d %s", rec.Code, rec.Body.String())
	}

	// Valid execute Collections -> 200 EXECUTED + RoiEventID + replay 409!
	execCollRec := doReq(ownerTok, http.MethodPost, "/v1/proposals/"+collectionsProp.ProposalID+"/execute", map[string]any{
		"payload_hash": approvedCollWrap.Proposal.PayloadHash,
	})
	if execCollRec.Code != http.StatusOK {
		t.Fatalf("execute collections proposal failed: %d %s", execCollRec.Code, execCollRec.Body.String())
	}
	if rec := doReq(ownerTok, http.MethodPost, "/v1/proposals/"+collectionsProp.ProposalID+"/execute", map[string]any{}); rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 proposal_already_executed on replay, got %d %s", rec.Code, rec.Body.String())
	}

	// Create an overlapping proposal targeting the same StockLot BEFORE executing expiryProp!
	dupPropRec := doReq(finTok, http.MethodPost, "/v1/proposals", map[string]any{
		"agent_id":               "EXPIRY_REBALANCE_AGENT",
		"proposal_type":          proposals.TypeExpiryRebalance,
		"target_object_type":     expiryProp.TargetObjectType,
		"target_object_id":       expiryProp.TargetObjectID,
		"title":                  "Overlapping expiry move",
		"payload":                expiryProp.Payload,
		"evidence_refs":          expiryProp.EvidenceRefs,
		"confidence_bps":         int64(8000),
		"estimated_impact_minor": int64(50000),
	})
	if dupPropRec.Code != http.StatusCreated {
		t.Fatalf("create overlapping proposal failed: %d %s", dupPropRec.Code, dupPropRec.Body.String())
	}
	var dupWrap struct {
		Proposal proposals.Proposal `json:"proposal"`
	}
	_ = json.Unmarshal(dupPropRec.Body.Bytes(), &dupWrap)

	// Approve & execute Reorder + ExpiryRebalance too.
	for _, prop := range []proposals.Proposal{reorderProp, expiryProp} {
		if rec := doReq(ownerTok, http.MethodPost, "/v1/proposals/"+prop.ProposalID+"/approve", map[string]any{"comment": "ok"}); rec.Code != http.StatusOK {
			t.Fatalf("approve %s failed: %d %s", prop.ProposalType, rec.Code, rec.Body.String())
		}
		if rec := doReq(ownerTok, http.MethodPost, "/v1/proposals/"+prop.ProposalID+"/execute", map[string]any{}); rec.Code != http.StatusOK {
			t.Fatalf("execute %s failed: %d %s", prop.ProposalType, rec.Code, rec.Body.String())
		}
	}

	// Self-approve SoD blocked -> 403!
	if rec := doReq(finTok, http.MethodPost, "/v1/proposals/"+dupWrap.Proposal.ProposalID+"/approve", map[string]any{}); rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 SoD self-approve block, got %d %s", rec.Code, rec.Body.String())
	}
	if rec := doReq(ownerTok, http.MethodPost, "/v1/proposals/"+dupWrap.Proposal.ProposalID+"/approve", map[string]any{}); rec.Code != http.StatusOK {
		t.Fatalf("approve overlapping proposal failed: %d %s", rec.Code, rec.Body.String())
	}
	if rec := doReq(ownerTok, http.MethodPost, "/v1/proposals/"+dupWrap.Proposal.ProposalID+"/execute", map[string]any{}); rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 target_execution_conflict on overlapping target, got %d %s", rec.Code, rec.Body.String())
	}

	// Create and reject a manual proposal cleanly.
	rejCreateRec := doReq(finTok, http.MethodPost, "/v1/proposals", map[string]any{
		"agent_id":               "COLLECTIONS_AGENT",
		"proposal_type":          proposals.TypeCollections,
		"target_object_type":     "ArInvoice",
		"target_object_id":       invOverdueID,
		"title":                  "Manual collections escalation",
		"payload":                collectionsProp.Payload,
		"evidence_refs":          collectionsProp.EvidenceRefs,
		"confidence_bps":         int64(7500),
		"estimated_impact_minor": int64(10000),
	})
	if rejCreateRec.Code != http.StatusCreated {
		t.Fatalf("create rejectable proposal failed: %d %s", rejCreateRec.Code, rejCreateRec.Body.String())
	}
	var rejCreateWrap struct {
		Proposal proposals.Proposal `json:"proposal"`
	}
	_ = json.Unmarshal(rejCreateRec.Body.Bytes(), &rejCreateWrap)
	if rec := doReq(ownerTok, http.MethodPost, "/v1/proposals/"+rejCreateWrap.Proposal.ProposalID+"/reject", map[string]any{"comment": "not needed"}); rec.Code != http.StatusOK {
		t.Fatalf("reject proposal failed: %d %s", rec.Code, rec.Body.String())
	}

	// --- 4. M1.8 ROI v1 Baseline, Treatment vs Holdout, Labour Estimate, QBR JSON & CSV ---
	baseRec := doReq(finTok, http.MethodPost, "/v1/roi/baseline", map[string]any{
		"window_days":              90,
		"baseline_labour_hours":    int64(25),
		"survey_hourly_wage_minor": int64(30000),
		"survey_notes":             "onboarding QBR baseline",
	})
	if baseRec.Code != http.StatusCreated {
		t.Fatalf("POST /v1/roi/baseline failed: %d %s", baseRec.Code, baseRec.Body.String())
	}
	if rec := doReq(auditorTok, http.MethodGet, "/v1/roi/baseline", nil); rec.Code != http.StatusOK {
		t.Fatalf("GET /v1/roi/baseline failed: %d %s", rec.Code, rec.Body.String())
	}

	if rec := doReq(finTok, http.MethodPost, "/v1/roi/events", map[string]any{
		"category":        roi.CategoryStockoutAvoided,
		"savings_minor":   int64(500000),
		"confidence_bps":  int64(9000),
		"holdout_control": false,
		"notes":           "treatment cell",
	}); rec.Code != http.StatusCreated {
		t.Fatalf("POST treatment ROI event failed: %d %s", rec.Code, rec.Body.String())
	}
	if rec := doReq(finTok, http.MethodPost, "/v1/roi/events", map[string]any{
		"category":        roi.CategoryStockoutAvoided,
		"savings_minor":   int64(100000),
		"confidence_bps":  int64(9000),
		"holdout_control": true,
		"notes":           "holdout control cell",
	}); rec.Code != http.StatusCreated {
		t.Fatalf("POST holdout ROI event failed: %d %s", rec.Code, rec.Body.String())
	}
	labourRec := doReq(finTok, http.MethodPost, "/v1/roi/events", map[string]any{
		"category":      roi.CategoryLabourSaved,
		"savings_minor": int64(150000),
		"is_estimate":   false, // must be forced true by service
		"notes":         "planner hours saved",
	})
	if labourRec.Code != http.StatusCreated {
		t.Fatalf("POST LABOUR_SAVED ROI event failed: %d %s", labourRec.Code, labourRec.Body.String())
	}
	var labourWrap struct {
		Event roi.RoiEvent `json:"event"`
	}
	_ = json.Unmarshal(labourRec.Body.Bytes(), &labourWrap)
	if !labourWrap.Event.IsEstimate {
		t.Fatalf("expected LABOUR_SAVED IsEstimate forced true, got %+v", labourWrap.Event)
	}

	sumRec := doReq(auditorTok, http.MethodGet, "/v1/roi/summary?days=90", nil)
	if sumRec.Code != http.StatusOK {
		t.Fatalf("GET /v1/roi/summary failed: %d %s", sumRec.Code, sumRec.Body.String())
	}
	var roiSum roi.Summary
	if err := json.Unmarshal(sumRec.Body.Bytes(), &roiSum); err != nil {
		t.Fatalf("unmarshal roi.Summary: %v", err)
	}
	if roiSum.PlatformFeesMinor <= 0 || roiSum.RoiMultipleBps <= 0 || roiSum.HoldoutStats.HoldoutSavingsMinor != 100000 {
		t.Fatalf("unexpected ROI summary metrics: %+v", roiSum)
	}

	qbrJSONRec := doReq(auditorTok, http.MethodGet, "/v1/roi/report?format=json&days=90", nil)
	if qbrJSONRec.Code != http.StatusOK || !strings.Contains(qbrJSONRec.Body.String(), `"roi_multiple_bps"`) {
		t.Fatalf("GET /v1/roi/report?format=json failed: %d %s", qbrJSONRec.Code, qbrJSONRec.Body.String())
	}
	qbrCSVRec := doReq(auditorTok, http.MethodGet, "/v1/roi/report?format=csv&days=90", nil)
	if qbrCSVRec.Code != http.StatusOK || !strings.Contains(qbrCSVRec.Header().Get("Content-Type"), "text/csv") ||
		!strings.Contains(qbrCSVRec.Body.String(), "LABOUR_SAVED") ||
		!strings.Contains(qbrCSVRec.Body.String(), "roi_multiple_bps") ||
		!strings.Contains(qbrCSVRec.Body.String(), "SUMMARY,") {
		t.Fatalf("GET /v1/roi/report?format=csv failed: %d headers=%v body=%s", qbrCSVRec.Code, qbrCSVRec.Header(), qbrCSVRec.Body.String())
	}

	// Verify live OutboxEvents contains Wave 3B events across M1.5-M1.8!
	seenOutbox := map[string]bool{}
	obIter := client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, spanner.Statement{
		SQL: `SELECT CAST(Payload AS STRING) FROM OutboxEvents
		      WHERE SupplierId = @eid OR AggregateId IN (@eid, @rid, @runid) OR CreatedAt >= @since`,
		Params: map[string]any{
			"eid":   entityID,
			"rid":   retID,
			"runid": fOut.Run.RunID,
			"since": now.Add(-2 * time.Minute),
		},
	})
	for {
		row, err := obIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			obIter.Stop()
			t.Fatalf("query OutboxEvents: %v", err)
		}
		var payloadStr string
		if row.Columns(&payloadStr) == nil {
			if !strings.Contains(payloadStr, entityID) && !strings.Contains(payloadStr, retID) && !strings.Contains(payloadStr, fOut.Run.RunID) {
				continue
			}
			var env struct {
				Type string `json:"type"`
			}
			if json.Unmarshal([]byte(payloadStr), &env) == nil && env.Type != "" {
				seenOutbox[env.Type] = true
			}
		}
	}
	obIter.Stop()
	for _, wantEv := range []string{
		"proposal.created",
		"proposal.decided",
		"proposal.executed",
		"roi.event.recorded",
		"credit.score.computed",
		"credit.limit.changed",
		"forecast.run.completed",
	} {
		if !seenOutbox[wantEv] {
			t.Fatalf("expected outbox event %q for tenant %s, got %+v", wantEv, entityID, seenOutbox)
		}
	}

	// Verify unbroken AuditEvents SHA-256 hash chain across all Wave 3B mutations!
	audChain, err := h.AuditSvc.VerifyDayChain(ctx, entityID, today)
	if err != nil || !audChain.Valid || audChain.EventCount < 10 {
		t.Fatalf("AuditEvents hash chain invalid: %+v err=%v", audChain, err)
	}
}
