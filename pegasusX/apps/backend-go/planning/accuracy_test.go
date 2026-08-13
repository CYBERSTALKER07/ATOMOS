package planning

import (
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"cloud.google.com/go/civil"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestComputeSeriesMetricsWapeBias(t *testing.T) {
	asOf := civil.Date{Year: 2026, Month: 8, Day: 6}
	points := []SeriesPoint{
		{Day: asOf.AddDays(-2), ForecastQty: 10, ActualQty: 10},
		{Day: asOf.AddDays(-1), ForecastQty: 12, ActualQty: 10},
		{Day: asOf, ForecastQty: 8, ActualQty: 10},
	}
	m := ComputeSeriesMetrics(points, asOf)
	// abs errors: 0 + 2 + 2 = 4; actual sum = 30; WAPE = 4/30
	wantWape := 4.0 / 30.0
	if math.Abs(m.Wape7-wantWape) > 1e-9 {
		t.Fatalf("Wape7=%v want %v", m.Wape7, wantWape)
	}
	// signed: 0 + 2 + (-2) = 0
	if math.Abs(m.Bias7) > 1e-9 {
		t.Fatalf("Bias7=%v want 0", m.Bias7)
	}
	if m.SampleDays7 != 3 {
		t.Fatalf("SampleDays7=%d want 3", m.SampleDays7)
	}
	if m.AlertTs {
		t.Fatal("expected no TS alert on balanced series")
	}
}

func TestComputeSeriesMetricsTrackingSignalAlert(t *testing.T) {
	asOf := civil.Date{Year: 2026, Month: 8, Day: 10}
	points := make([]SeriesPoint, 0, 10)
	for i := 9; i >= 0; i-- {
		// Consistently over-forecast by 10 → large |TS|.
		points = append(points, SeriesPoint{
			Day: asOf.AddDays(-i), ForecastQty: 20, ActualQty: 10,
		})
	}
	m := ComputeSeriesMetrics(points, asOf)
	if math.Abs(m.TrackingSignal) <= 4 {
		t.Fatalf("TrackingSignal=%v want |TS|>4", m.TrackingSignal)
	}
	if !m.AlertTs {
		t.Fatal("expected AlertTs")
	}
	// bias = +1.0 (always +10 on actual 10)
	if math.Abs(m.Bias28-1.0) > 1e-9 {
		t.Fatalf("Bias28=%v want 1", m.Bias28)
	}
}

func TestConfidencePctFromWape(t *testing.T) {
	pct, ok := ConfidencePctFromWape(0.25, 7)
	if !ok || pct != 75 {
		t.Fatalf("got pct=%d ok=%v want 75 true", pct, ok)
	}
	if _, ok := ConfidencePctFromWape(0.1, 6); ok {
		t.Fatal("expected insufficient sample")
	}
	pct, ok = ConfidencePctFromWape(1.5, 10)
	if !ok || pct != 0 {
		t.Fatalf("got pct=%d ok=%v want 0 true", pct, ok)
	}
}

func TestComputeMAPE(t *testing.T) {
	// |10-10|/10 + |12-10|/10 + |8-10|/10 = 0 + 0.2 + 0.2 = 0.4; mean = 0.4/3
	points := []SeriesPoint{
		{ForecastQty: 10, ActualQty: 10},
		{ForecastQty: 12, ActualQty: 10},
		{ForecastQty: 8, ActualQty: 10},
		{ForecastQty: 5, ActualQty: 0}, // skipped (a<=0)
	}
	got := computeMAPE(points)
	want := (0.0 + 0.2 + 0.2) / 3.0
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("MAPE=%v want %v", got, want)
	}
	m := ComputeSeriesMetrics([]SeriesPoint{
		{Day: civil.Date{Year: 2026, Month: 8, Day: 4}, ForecastQty: 10, ActualQty: 10},
		{Day: civil.Date{Year: 2026, Month: 8, Day: 5}, ForecastQty: 12, ActualQty: 10},
		{Day: civil.Date{Year: 2026, Month: 8, Day: 6}, ForecastQty: 8, ActualQty: 10},
	}, civil.Date{Year: 2026, Month: 8, Day: 6})
	if math.Abs(m.Mape28-want) > 1e-9 {
		t.Fatalf("Mape28=%v want %v", m.Mape28, want)
	}
}

func TestShouldDemote(t *testing.T) {
	t.Setenv("FORECAST_DEMOTE_ENABLED", "false")
	if ShouldDemote(0.9, 20) {
		t.Fatal("flag off → no demote")
	}
	t.Setenv("FORECAST_DEMOTE_ENABLED", "true")
	t.Setenv("FORECAST_DEMOTE_WAPE28_MAX", "0.45")
	if ShouldDemote(0.50, 10) {
		t.Fatal("sample < 14 → no demote")
	}
	if !ShouldDemote(0.50, 14) {
		t.Fatal("WAPE>max + sample>=14 → demote")
	}
	if ShouldDemote(0.40, 20) {
		t.Fatal("WAPE under max → no demote")
	}
	if AccuracyDemotedReason != "accuracy_demoted" {
		t.Fatalf("reason=%q", AccuracyDemotedReason)
	}
}

func TestHandleListAccuracyMethodNotAllowed(t *testing.T) {
	svc := &AccuracyService{}
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/planning/accuracy", nil)
	rec := httptest.NewRecorder()
	svc.HandleListAccuracy(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("got %d want 405", rec.Code)
	}
}

func TestHandleListAccuracyForbidden(t *testing.T) {
	svc := &AccuracyService{}
	// No claims.
	rec := httptest.NewRecorder()
	svc.HandleListAccuracy(rec, httptest.NewRequest(http.MethodGet, "/v1/admin/planning/accuracy", nil))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("no-claims: got %d want 403", rec.Code)
	}
	// Non-admin role.
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/planning/accuracy", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{Subject: "u1", Role: auth.RoleRetailer}))
	rec = httptest.NewRecorder()
	svc.HandleListAccuracy(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("non-admin: got %d want 403", rec.Code)
	}
}

func TestHandleListAccuracyRequiresSupplier(t *testing.T) {
	svc := &AccuracyService{}
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/planning/accuracy", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{Subject: "u1", Role: auth.RoleAdmin}))
	rec := httptest.NewRecorder()
	svc.HandleListAccuracy(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got %d want 400 (missing supplier_id)", rec.Code)
	}
}

func TestRegisterAccuracyRoutesNilSafe(t *testing.T) {
	// Must not panic on nil service.
	RegisterAccuracyRoutes(newTestRouter(), nil)
}

func newTestRouter() *chi.Mux {
	return chi.NewRouter()
}
