package analytics

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"backend-go/auth"
)

func TestNormalizeForecastTournamentRequest_Defaults(t *testing.T) {
	normalized, err := normalizeForecastTournamentRequest(ForecastTournamentRequest{}, nil)
	if err != nil {
		t.Fatalf("normalizeForecastTournamentRequest() error = %v", err)
	}

	if normalized.MinSampleSize != defaultForecastTournamentSampleSize {
		t.Fatalf("min sample size = %d, want %d", normalized.MinSampleSize, defaultForecastTournamentSampleSize)
	}
	if normalized.WarehouseID != "" {
		t.Fatalf("warehouse_id = %q, want empty", normalized.WarehouseID)
	}
	if !normalized.To.After(normalized.From) {
		t.Fatalf("time window invalid: from=%v to=%v", normalized.From, normalized.To)
	}
}

func TestNormalizeForecastTournamentRequest_ScopeWarehouseOverride(t *testing.T) {
	ws := &auth.WarehouseScope{WarehouseID: "wh-scope-1"}
	normalized, err := normalizeForecastTournamentRequest(ForecastTournamentRequest{
		WarehouseID: "wh-request-1",
	}, ws)
	if err != nil {
		t.Fatalf("normalizeForecastTournamentRequest() error = %v", err)
	}
	if normalized.WarehouseID != "wh-scope-1" {
		t.Fatalf("warehouse_id = %q, want wh-scope-1", normalized.WarehouseID)
	}
}

func TestNormalizeForecastTournamentRequest_MinSampleLimit(t *testing.T) {
	_, err := normalizeForecastTournamentRequest(ForecastTournamentRequest{
		MinSampleSize: maxForecastTournamentSampleSize + 1,
	}, nil)
	if err == nil {
		t.Fatal("expected error for oversized min_sample_size")
	}
}

func TestHandleForecastTournament_MethodNotAllowed(t *testing.T) {
	handler := HandleForecastTournament(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/v1/supplier/analytics/forecast/tournament", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleForecastTournament_UnauthorizedWithoutClaims(t *testing.T) {
	handler := HandleForecastTournament(nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/analytics/forecast/tournament", bytes.NewBufferString(`{}`))
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestHandleForecastTournament_InvalidJSONWithClaims(t *testing.T) {
	handler := HandleForecastTournament(nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/analytics/forecast/tournament", bytes.NewBufferString(`{"unknown":true}`))
	req = withSupplierClaims(req)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleForecastTournament_InvalidRequestWithClaims(t *testing.T) {
	handler := HandleForecastTournament(nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/analytics/forecast/tournament", bytes.NewBufferString(`{"min_sample_size":9999}`))
	req = withSupplierClaims(req)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
