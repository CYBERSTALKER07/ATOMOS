package retailer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPredictDemand_Success(t *testing.T) {
	expectedAPIKey := "test-secret-internal-key"
	receivedKey := ""
	receivedPath := ""

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedKey = r.Header.Get("X-Internal-Api-Key")
		receivedPath = r.URL.Path

		var req DemandPredictionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		resp := DemandPredictionResponse{
			SKU:                req.SKU,
			OptimalReorderQty:  42,
			PredictedDailyBurn: 7.5,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	t.Setenv("AI_DEMAND_SOLVER_URL", server.URL+"/")
	t.Setenv("INTERNAL_API_KEY", expectedAPIKey)

	res, err := PredictDemand(context.Background(), DemandPredictionRequest{
		SKU:                  "SKU-TEST-1",
		HistoricalDailySales: []int{10, 5},
		CurrentStock:         20,
		LeadTimeDays:         3,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedKey != expectedAPIKey {
		t.Fatalf("expected X-Internal-Api-Key %q, got %q", expectedAPIKey, receivedKey)
	}
	if receivedPath != "/predict-demand" {
		t.Fatalf("expected path /predict-demand, got %q", receivedPath)
	}
	if res.SKU != "SKU-TEST-1" {
		t.Fatalf("expected SKU SKU-TEST-1, got %q", res.SKU)
	}
	if res.OptimalReorderQty != 42 {
		t.Fatalf("expected optimal reorder qty 42, got %d", res.OptimalReorderQty)
	}
	if res.PredictedDailyBurn != 7.5 {
		t.Fatalf("expected predicted daily burn 7.5, got %f", res.PredictedDailyBurn)
	}
}

func TestPredictDemand_Non200ReturnsError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
	}{
		{name: "server error 500", statusCode: http.StatusInternalServerError, body: `{"detail":"internal solver failure"}`},
		{name: "validation error 422", statusCode: http.StatusUnprocessableEntity, body: `{"detail":"unprocessable entity"}`},
		{name: "bad request 400", statusCode: http.StatusBadRequest, body: `{"detail":"bad request"}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()

			t.Setenv("AI_DEMAND_SOLVER_URL", server.URL)
			t.Setenv("INTERNAL_API_KEY", "")

			res, err := PredictDemand(context.Background(), DemandPredictionRequest{
				SKU:                  "SKU-ERR",
				HistoricalDailySales: []int{5},
				CurrentStock:         10,
				LeadTimeDays:         1,
			})
			if err == nil {
				t.Fatalf("expected error for status %d, got nil response: %#v", tc.statusCode, res)
			}
			expectedMsg := fmt.Sprintf("demand prediction solver returned HTTP %d", tc.statusCode)
			if err.Error() != expectedMsg {
				t.Fatalf("expected error %q, got %q", expectedMsg, err.Error())
			}
		})
	}
}

func TestResolveDemandSolverBaseURL(t *testing.T) {
	t.Run("fails closed when unconfigured", func(t *testing.T) {
		t.Setenv("AI_DEMAND_SOLVER_URL", "")
		t.Setenv("DISPATCH_OPTIMIZER_URL", "")
		t.Setenv("OPTIMIZER_SIDECAR_URL", "")
		t.Setenv("OPTIMIZER_BASE_URL", "")
		t.Setenv("AI_DEMAND_SIDECAR_ALLOW_LOCALHOST", "")

		if got, err := resolveDemandSolverBaseURL(); !errors.Is(err, ErrDemandSolverNotConfigured) || got != "" {
			t.Fatalf("expected ErrDemandSolverNotConfigured and empty url, got url=%q err=%v", got, err)
		}

		res, err := PredictDemand(context.Background(), DemandPredictionRequest{
			SKU:                  "SKU-UNCONFIGURED",
			HistoricalDailySales: []int{5},
			CurrentStock:         10,
			LeadTimeDays:         1,
		})
		if !errors.Is(err, ErrDemandSolverNotConfigured) || res != nil {
			t.Fatalf("expected PredictDemand to fail closed with ErrDemandSolverNotConfigured, got res=%#v err=%v", res, err)
		}
	})

	t.Run("explicit non-prod localhost opt-in", func(t *testing.T) {
		t.Setenv("AI_DEMAND_SOLVER_URL", "")
		t.Setenv("DISPATCH_OPTIMIZER_URL", "")
		t.Setenv("OPTIMIZER_SIDECAR_URL", "")
		t.Setenv("OPTIMIZER_BASE_URL", "")
		t.Setenv("AI_DEMAND_SIDECAR_ALLOW_LOCALHOST", "true")
		t.Setenv("PEGASUSX_ENV", "local")

		got, err := resolveDemandSolverBaseURL()
		if err != nil || got != "http://localhost:8000" {
			t.Fatalf("expected http://localhost:8000 with non-prod opt-in, got url=%q err=%v", got, err)
		}

		// Production ignores AI_DEMAND_SIDECAR_ALLOW_LOCALHOST=true and fails closed.
		t.Setenv("PEGASUSX_ENV", "production")
		if gotProd, errProd := resolveDemandSolverBaseURL(); !errors.Is(errProd, ErrDemandSolverNotConfigured) || gotProd != "" {
			t.Fatalf("expected ErrDemandSolverNotConfigured in production, got url=%q err=%v", gotProd, errProd)
		}
	})

	t.Run("priority order", func(t *testing.T) {
		t.Setenv("AI_DEMAND_SOLVER_URL", "")
		t.Setenv("DISPATCH_OPTIMIZER_URL", "")
		t.Setenv("OPTIMIZER_SIDECAR_URL", "")
		t.Setenv("AI_DEMAND_SIDECAR_ALLOW_LOCALHOST", "")
		t.Setenv("OPTIMIZER_BASE_URL", "http://base:8000")
		if got, err := resolveDemandSolverBaseURL(); err != nil || got != "http://base:8000" {
			t.Fatalf("expected base, got %q err=%v", got, err)
		}

		t.Setenv("OPTIMIZER_BASE_URL", "http://core-solver:8082")
		if got, err := resolveDemandSolverBaseURL(); !errors.Is(err, ErrDemandSolverNotConfigured) || got != "" {
			t.Fatalf("expected 8082 to be ignored and fail closed, got %q err=%v", got, err)
		}

		t.Setenv("OPTIMIZER_SIDECAR_URL", "http://sidecar:8000")
		if got, err := resolveDemandSolverBaseURL(); err != nil || got != "http://sidecar:8000" {
			t.Fatalf("expected sidecar, got %q err=%v", got, err)
		}

		t.Setenv("DISPATCH_OPTIMIZER_URL", "http://dispatch-opt:8000")
		if got, err := resolveDemandSolverBaseURL(); err != nil || got != "http://dispatch-opt:8000" {
			t.Fatalf("expected dispatch-opt, got %q err=%v", got, err)
		}

		t.Setenv("AI_DEMAND_SOLVER_URL", "http://ai-demand:8000")
		if got, err := resolveDemandSolverBaseURL(); err != nil || got != "http://ai-demand:8000" {
			t.Fatalf("expected ai-demand, got %q err=%v", got, err)
		}
	})
}
