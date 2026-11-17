package controltower

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestObservabilityHandlers_Success(t *testing.T) {
	h := &Handlers{}

	t.Run("HandleCells", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/control-tower/cells", nil)
		rec := httptest.NewRecorder()
		h.HandleCells(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var cells []CellHealth
		if err := json.Unmarshal(rec.Body.Bytes(), &cells); err != nil {
			t.Fatalf("failed to decode cells: %v", err)
		}
		if len(cells) != 3 {
			t.Fatalf("expected 3 cells, got %d", len(cells))
		}
		if cells[0].CellID != "cell-uz-tashkent" || cells[0].Status != "HEALTHY" {
			t.Fatalf("unexpected cell data: %+v", cells[0])
		}
	})

	t.Run("HandleSpannerMetrics", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/control-tower/metrics/spanner", nil)
		rec := httptest.NewRecorder()
		h.HandleSpannerMetrics(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var m SpannerClusterMetrics
		if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
			t.Fatalf("failed to decode spanner metrics: %v", err)
		}
		if m.P99ReadMs <= 0 || len(m.HourlyHistory) == 0 {
			t.Fatalf("unexpected spanner metrics: %+v", m)
		}
	})

	t.Run("HandleKafkaMetrics", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/control-tower/metrics/kafka", nil)
		rec := httptest.NewRecorder()
		h.HandleKafkaMetrics(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var lags []KafkaPartitionLag
		if err := json.Unmarshal(rec.Body.Bytes(), &lags); err != nil {
			t.Fatalf("failed to decode kafka lags: %v", err)
		}
		if len(lags) != 4 {
			t.Fatalf("expected 4 kafka topics, got %d", len(lags))
		}
	})

	t.Run("HandleRedisMetrics", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/control-tower/metrics/redis", nil)
		rec := httptest.NewRecorder()
		h.HandleRedisMetrics(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var pools []RedisPermitPool
		if err := json.Unmarshal(rec.Body.Bytes(), &pools); err != nil {
			t.Fatalf("failed to decode redis pools: %v", err)
		}
		if len(pools) != 4 {
			t.Fatalf("expected 4 redis pools, got %d", len(pools))
		}
	})

	t.Run("HandleWaterfall_RolePerspectives", func(t *testing.T) {
		roles := []struct {
			role          string
			expectedCount int
		}{
			{"Warehouse Perspective", 4},
			{"Retailer Perspective", 4},
			{"Fleet & Carrier", 4},
			{"Overall Ecosystem", 6},
			{"", 6},
		}

		for _, tc := range roles {
			path := "/v1/control-tower/waterfall"
			if tc.role != "" {
				path += "?role=" + url.QueryEscape(tc.role)
			}
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()
			h.HandleWaterfall(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("role %q: expected 200, got %d", tc.role, rec.Code)
			}
			var slices []TenderWaterfallSlice
			if err := json.Unmarshal(rec.Body.Bytes(), &slices); err != nil {
				t.Fatalf("role %q: failed to decode waterfall: %v", tc.role, err)
			}
			if len(slices) != tc.expectedCount {
				t.Fatalf("role %q: expected %d slices, got %d", tc.role, tc.expectedCount, len(slices))
			}
		}
	})

	t.Run("HandleAttributedCosts", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/control-tower/costs/attributed", nil)
		rec := httptest.NewRecorder()
		h.HandleAttributedCosts(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var costs []AttributedCostItem
		if err := json.Unmarshal(rec.Body.Bytes(), &costs); err != nil {
			t.Fatalf("failed to decode attributed costs: %v", err)
		}
		if len(costs) != 6 {
			t.Fatalf("expected 6 cost items, got %d", len(costs))
		}
	})

	t.Run("HandleScorecards", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/control-tower/scorecards", nil)
		rec := httptest.NewRecorder()
		h.HandleScorecards(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var scorecards []RolePerformanceScorecard
		if err := json.Unmarshal(rec.Body.Bytes(), &scorecards); err != nil {
			t.Fatalf("failed to decode scorecards: %v", err)
		}
		if len(scorecards) != 4 {
			t.Fatalf("expected 4 scorecards, got %d", len(scorecards))
		}
	})
}

func TestObservabilityHandlers_MethodNotAllowed(t *testing.T) {
	h := &Handlers{}
	endpoints := []func(w http.ResponseWriter, r *http.Request){
		h.HandleCells,
		h.HandleSpannerMetrics,
		h.HandleKafkaMetrics,
		h.HandleRedisMetrics,
		h.HandleWaterfall,
		h.HandleAttributedCosts,
		h.HandleScorecards,
	}

	for i, handler := range endpoints {
		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		rec := httptest.NewRecorder()
		handler(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("handler %d: expected 405 Method Not Allowed, got %d", i, rec.Code)
		}
	}
}
