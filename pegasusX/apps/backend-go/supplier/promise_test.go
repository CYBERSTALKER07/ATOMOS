package supplier

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestEvaluatePromise_SameDayCutoff(t *testing.T) {
	svc := &Service{}
	ctx := context.Background()

	// 07:00 AM UTC = 12:00 PM Tashkent (before 14:00 cutoff)
	now := time.Date(2026, 8, 19, 7, 0, 0, 0, time.UTC)
	req := PromiseEvaluationRequest{
		SupplierID: "sup-1",
		TotalMinor: 50000,
		Currency:   "UZS",
	}

	res, err := svc.EvaluatePromise(ctx, req, now)
	if err != nil {
		t.Fatalf("EvaluatePromise failed: %v", err)
	}

	if !res.Eligible {
		t.Fatalf("expected eligible, got false: %s", res.Reason)
	}
	if res.PromiseType != "SAME_DAY" {
		t.Fatalf("expected SAME_DAY promise before cutoff, got %s", res.PromiseType)
	}
	if res.SLAHours != 8 {
		t.Fatalf("expected 8 hour SLA for same day, got %d", res.SLAHours)
	}
}

func TestEvaluatePromise_NextDayAfterCutoff(t *testing.T) {
	svc := &Service{}
	ctx := context.Background()

	// 12:00 PM UTC = 17:00 PM Tashkent (after 14:00 cutoff)
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	req := PromiseEvaluationRequest{
		SupplierID: "sup-1",
		TotalMinor: 50000,
		Currency:   "UZS",
	}

	res, err := svc.EvaluatePromise(ctx, req, now)
	if err != nil {
		t.Fatalf("EvaluatePromise failed: %v", err)
	}

	if !res.Eligible {
		t.Fatalf("expected eligible, got false: %s", res.Reason)
	}
	if res.PromiseType != "NEXT_DAY" {
		t.Fatalf("expected NEXT_DAY promise after cutoff, got %s", res.PromiseType)
	}
	if res.SLAHours != 24 {
		t.Fatalf("expected 24 hour SLA for next day, got %d", res.SLAHours)
	}
}

func TestEvaluatePromise_ScheduledDelivery(t *testing.T) {
	svc := &Service{}
	ctx := context.Background()

	now := time.Date(2026, 8, 19, 10, 0, 0, 0, time.UTC)
	futureDate := now.Add(5 * 24 * time.Hour).Format(time.RFC3339)
	req := PromiseEvaluationRequest{
		SupplierID:            "sup-1",
		TotalMinor:            50000,
		Currency:              "UZS",
		RequestedDeliveryDate: futureDate,
	}

	res, err := svc.EvaluatePromise(ctx, req, now)
	if err != nil {
		t.Fatalf("EvaluatePromise failed: %v", err)
	}

	if !res.Eligible {
		t.Fatalf("expected eligible, got false: %s", res.Reason)
	}
	if res.PromiseType != "SCHEDULED" {
		t.Fatalf("expected SCHEDULED promise, got %s", res.PromiseType)
	}
}

func TestHandleGetServicePolicy_Unauthorized(t *testing.T) {
	svc := &Service{}
	req := httptest.NewRequest(http.MethodGet, "/v1/supplier/service-policy", nil)
	rec := httptest.NewRecorder()

	svc.HandleGetServicePolicy(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", rec.Code)
	}
}

func TestHandleEvaluateServicePromise_MissingSupplier(t *testing.T) {
	svc := &Service{}
	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/service-promise", nil)
	rec := httptest.NewRecorder()

	svc.HandleEvaluateServicePromise(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", rec.Code)
	}
}

func TestHandleEvaluateServicePromise_Success(t *testing.T) {
	svc := &Service{}
	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/service-promise?supplier_id=sup-1&total_minor=50000", nil)
	rec := httptest.NewRecorder()

	svc.HandleEvaluateServicePromise(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"eligible":true`) {
		t.Fatalf("expected response with eligible: true, got %s", rec.Body.String())
	}
}

func TestHandleGetServicePolicy_Authorized(t *testing.T) {
	svc := &Service{}
	req := httptest.NewRequest(http.MethodGet, "/v1/supplier/service-policy", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject:    "user-1",
		Role:       auth.RoleAdmin,
		SupplierID: "sup-1",
	}))
	rec := httptest.NewRecorder()

	svc.HandleGetServicePolicy(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"lead_time_days":1`) {
		t.Fatalf("expected default lead_time_days:1, got %s", rec.Body.String())
	}
}
