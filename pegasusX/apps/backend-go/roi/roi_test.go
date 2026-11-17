package roi_test

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/roi"
	"github.com/pegasusx/pegasusx/apps/backend-go/roiroutes"
)

func TestROILifecycle_BaselineEventsSummaryHoldoutAndQBR(t *testing.T) {
	ctx := context.Background()
	repo := roi.NewMemoryRepository()
	repo.SetHistoricalMetrics("sup-roi-1", roi.HistoricalMetrics{
		ArMinor:      5000000,
		BadDebtMinor: 350000,
		ExpiredMinor: 120000,
		Currency:     "UZS",
	})
	repo.SetPlatformFeesMinor("sup-roi-1", 200000)

	svc := roi.NewServiceWithRepository(repo)

	base, err := svc.CaptureBaseline(ctx, roi.CaptureBaselineInput{
		TenantID:               "sup-roi-1",
		WindowDays:             90,
		BaselineStockoutMinor:  400000,
		BaselineLogisticsMinor: 250000,
		BaselineLabourHours:    40,
		SurveyHourlyWageMinor:  25000,
		SurveyNotes:            "Onboarding Q3 baseline survey",
		Survey:                 map[string]any{"manual_orders_per_week": 150},
		CapturedBy:             "owner-1",
	})
	if err != nil {
		t.Fatalf("CaptureBaseline: %v", err)
	}
	if base.BaselineArMinor != 5000000 || base.BaselineBadDebtMinor != 350000 || base.BaselineExpiredMinor != 120000 {
		t.Fatalf("unexpected historical baseline values: %+v", base)
	}

	// Reject float in Survey
	if _, err := svc.CaptureBaseline(ctx, roi.CaptureBaselineInput{
		TenantID: "sup-roi-1",
		Survey:   map[string]any{"avg_rate": 12.5},
	}); !errors.Is(err, roi.ErrFloatForbidden) {
		t.Fatalf("expected ErrFloatForbidden, got %v", err)
	}

	// Record all 6 categories, including holdout control and labour estimate flag enforcement.
	_, err = svc.RecordEvent(ctx, roi.RecordEventInput{
		TenantID: "sup-roi-1", Category: roi.CategoryBadDebtAvoided, SavingsMinor: 300000,
		ProposalID: "prop-1", EvidenceRefType: "ArInvoice", EvidenceRefID: "inv-1", IdempotencyKey: "k1",
	})
	if err != nil {
		t.Fatalf("RecordEvent bad_debt: %v", err)
	}
	// Duplicate idempotency key must return same event without doubling savings.
	evDup, err := svc.RecordEvent(ctx, roi.RecordEventInput{
		TenantID: "sup-roi-1", Category: roi.CategoryBadDebtAvoided, SavingsMinor: 300000,
		IdempotencyKey: "k1",
	})
	if err != nil || evDup.SavingsMinor != 300000 {
		t.Fatalf("idempotency replay failed: %+v err=%v", evDup, err)
	}

	_, err = svc.RecordEvent(ctx, roi.RecordEventInput{
		TenantID: "sup-roi-1", Category: roi.CategoryStockoutAvoided, SavingsMinor: 400000,
		HoldoutControl: false, EvidenceRefID: "sku-1", IdempotencyKey: "k2",
	})
	if err != nil {
		t.Fatalf("RecordEvent stockout treatment: %v", err)
	}
	_, err = svc.RecordEvent(ctx, roi.RecordEventInput{
		TenantID: "sup-roi-1", Category: roi.CategoryStockoutAvoided, SavingsMinor: 100000,
		HoldoutControl: true, EvidenceRefID: "sku-holdout-1", IdempotencyKey: "k3",
	})
	if err != nil {
		t.Fatalf("RecordEvent stockout holdout: %v", err)
	}
	_, err = svc.RecordEvent(ctx, roi.RecordEventInput{
		TenantID: "sup-roi-1", Category: roi.CategoryExpirySaved, SavingsMinor: 150000,
		ProposalID: "prop-2", EvidenceRefType: "StockLot", EvidenceRefID: "lot-1", IdempotencyKey: "k4",
	})
	if err != nil {
		t.Fatalf("RecordEvent expiry: %v", err)
	}
	_, err = svc.RecordEvent(ctx, roi.RecordEventInput{
		TenantID: "sup-roi-1", Category: roi.CategoryLogisticsSaved, SavingsMinor: 50000,
		IdempotencyKey: "k5",
	})
	if err != nil {
		t.Fatalf("RecordEvent logistics: %v", err)
	}
	falseVal := false
	labourEv, err := svc.RecordEvent(ctx, roi.RecordEventInput{
		TenantID: "sup-roi-1", Category: roi.CategoryLabourSaved, SavingsMinor: 100000,
		IsEstimate: &falseVal, IdempotencyKey: "k6",
	})
	if err != nil {
		t.Fatalf("RecordEvent labour: %v", err)
	}
	if !labourEv.IsEstimate {
		t.Fatalf("expected LABOUR_SAVED to always enforce IsEstimate=true")
	}
	_, err = svc.RecordEvent(ctx, roi.RecordEventInput{
		TenantID: "sup-roi-1", Category: roi.CategoryWorkingCapitalSaved, SavingsMinor: 200000,
		IdempotencyKey: "k7",
	})
	if err != nil {
		t.Fatalf("RecordEvent working capital: %v", err)
	}

	sum, err := svc.GetSummary(ctx, "sup-roi-1", 90)
	if err != nil {
		t.Fatalf("GetSummary: %v", err)
	}
	// Treatment = 300k + 400k + 150k + 50k + 100k + 200k = 1,200,000; Holdout = 100,000; Fees = 200,000 -> ROI = 60,000 bps (6.0x)
	if sum.TotalSavingsMinor != 1200000 || sum.RealizedSavingsMinor != 1100000 || sum.EstimatedSavingsMinor != 100000 {
		t.Fatalf("unexpected totals: %+v", sum)
	}
	if sum.HoldoutSavingsMinor != 100000 || sum.PlatformFeesMinor != 200000 || sum.NetSavingsMinor != 1000000 || sum.RoiMultipleBps != 60000 {
		t.Fatalf("unexpected roi multiple/holdout: %+v", sum)
	}
	if sum.HoldoutStats.IncrementalLiftMinor != 1100000 {
		t.Fatalf("unexpected holdout lift: %+v", sum.HoldoutStats)
	}

	csvBytes, err := svc.ExportQBRCSV(ctx, "sup-roi-1", 90)
	if err != nil {
		t.Fatalf("ExportQBRCSV: %v", err)
	}
	recs, err := csv.NewReader(bytes.NewReader(csvBytes)).ReadAll()
	if err != nil || len(recs) < 9 {
		t.Fatalf("unexpected CSV output: rows=%d err=%v", len(recs), err)
	}

	// HTTP routes RBAC + float rejection + QBR download verification.
	r := chi.NewRouter()
	const jwtSecret = "roi-test-secret-32-bytes-minimum!!"
	r.Use(auth.SessionAuth(jwtSecret))
	r.Use(auth.AttachTenantFromClaims)
	roiroutes.RegisterRoutes(r, roiroutes.Deps{
		Service:   svc,
		JWTSecret: jwtSecret,
	})

	issueTok := func(sub string, supRole auth.Role, actorType string) string {
		tok, err := auth.Issue(auth.Claims{
			Subject:      sub,
			Role:         auth.RoleAdmin,
			SupplierID:   "sup-roi-1",
			SupplierRole: supRole,
			ActorType:    actorType,
		}, auth.IssueOptions{Secret: jwtSecret, TTL: time.Hour})
		if err != nil {
			t.Fatal(err)
		}
		return tok
	}
	ownerTok := issueTok("owner-1", auth.SupplierRoleOwner, auth.ActorTypeHuman)
	auditorTok := issueTok("auditor-1", auth.SupplierRoleAuditor, auth.ActorTypeHuman)
	agentTok := issueTok("agent-1", auth.SupplierRoleAgent, auth.ActorTypeAgent)

	// Auditor can read summary & report, CANNOT POST event.
	reqRead := httptest.NewRequest(http.MethodGet, "/v1/roi/summary", nil)
	reqRead.Header.Set("Authorization", "Bearer "+auditorTok)
	recRead := httptest.NewRecorder()
	r.ServeHTTP(recRead, reqRead)
	if recRead.Code != http.StatusOK {
		t.Fatalf("auditor GET /v1/roi/summary = %d %s", recRead.Code, recRead.Body.String())
	}

	reqAudMut := httptest.NewRequest(http.MethodPost, "/v1/roi/events", strings.NewReader(`{"category":"EXPIRY_SAVED","savings_minor":1000}`))
	reqAudMut.Header.Set("Authorization", "Bearer "+auditorTok)
	reqAudMut.Header.Set("Content-Type", "application/json")
	recAudMut := httptest.NewRecorder()
	r.ServeHTTP(recAudMut, reqAudMut)
	if recAudMut.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for auditor mutation, got %d", recAudMut.Code)
	}

	reqAgentMut := httptest.NewRequest(http.MethodPost, "/v1/roi/events", strings.NewReader(`{"category":"EXPIRY_SAVED","savings_minor":1000}`))
	reqAgentMut.Header.Set("Authorization", "Bearer "+agentTok)
	reqAgentMut.Header.Set("Content-Type", "application/json")
	recAgentMut := httptest.NewRecorder()
	r.ServeHTTP(recAgentMut, reqAgentMut)
	if recAgentMut.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for agent mutation, got %d", recAgentMut.Code)
	}

	// Owner float rejected with 400.
	reqFloat := httptest.NewRequest(http.MethodPost, "/v1/roi/events", strings.NewReader(`{"category":"EXPIRY_SAVED","savings_minor":100.5}`))
	reqFloat.Header.Set("Authorization", "Bearer "+ownerTok)
	reqFloat.Header.Set("Content-Type", "application/json")
	recFloat := httptest.NewRecorder()
	r.ServeHTTP(recFloat, reqFloat)
	if recFloat.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for float in ROI event, got %d %s", recFloat.Code, recFloat.Body.String())
	}

	// Owner CSV report download.
	reqCSV := httptest.NewRequest(http.MethodGet, "/v1/roi/report?format=csv&days=90", nil)
	reqCSV.Header.Set("Authorization", "Bearer "+ownerTok)
	recCSV := httptest.NewRecorder()
	r.ServeHTTP(recCSV, reqCSV)
	if recCSV.Code != http.StatusOK || !strings.Contains(recCSV.Header().Get("Content-Type"), "text/csv") {
		t.Fatalf("expected 200 CSV report, got %d %s", recCSV.Code, recCSV.Header().Get("Content-Type"))
	}
	var _ json.RawMessage
}
