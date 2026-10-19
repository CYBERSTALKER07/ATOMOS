package warehouse

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleOpsDispatchRescuePreview_Validation(t *testing.T) {
	s := &Service{}

	t.Run("MethodNotAllowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/warehouse/ops/dispatch/rescue/preview", nil)
		rr := httptest.NewRecorder()
		s.HandleOpsDispatchRescuePreview(rr, req)
		if rr.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want 405", rr.Code)
		}
	})

	t.Run("MissingWarehouseID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/v1/warehouse/ops/dispatch/rescue/preview", bytes.NewBufferString(`{"broken_driver_id":"drv-1"}`))
		rr := httptest.NewRecorder()
		s.HandleOpsDispatchRescuePreview(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rr.Code)
		}
		if !strings.Contains(rr.Body.String(), "warehouse_id_required") {
			t.Fatalf("expected warehouse_id_required, got %s", rr.Body.String())
		}
	})

	t.Run("MissingBrokenDriverID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/v1/warehouse/ops/dispatch/rescue/preview?warehouse_id=wh-1", bytes.NewBufferString(`{}`))
		rr := httptest.NewRecorder()
		s.HandleOpsDispatchRescuePreview(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rr.Code)
		}
		if !strings.Contains(rr.Body.String(), "invalid_request") {
			t.Fatalf("expected invalid_request, got %s", rr.Body.String())
		}
	})
}

func TestHandleOpsDispatchRescuePropose_Validation(t *testing.T) {
	s := &Service{}

	t.Run("MethodNotAllowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/warehouse/ops/dispatch/rescue/propose", nil)
		rr := httptest.NewRecorder()
		s.HandleOpsDispatchRescuePropose(rr, req)
		if rr.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want 405", rr.Code)
		}
	})

	t.Run("MissingWarehouseID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/v1/warehouse/ops/dispatch/rescue/propose", bytes.NewBufferString(`{"broken_driver_id":"drv-1","rescue_driver_id":"drv-2","rescue_id":"resc-1"}`))
		rr := httptest.NewRecorder()
		s.HandleOpsDispatchRescuePropose(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rr.Code)
		}
		if !strings.Contains(rr.Body.String(), "warehouse_id_required") {
			t.Fatalf("expected warehouse_id_required, got %s", rr.Body.String())
		}
	})

	t.Run("IncompletePayload", func(t *testing.T) {
		cases := []struct {
			name string
			body string
		}{
			{"missing_all", `{}`},
			{"missing_rescue_id", `{"broken_driver_id":"drv-1","rescue_driver_id":"drv-2"}`},
			{"missing_rescue_driver", `{"broken_driver_id":"drv-1","rescue_id":"resc-1"}`},
			{"missing_broken_driver", `{"rescue_driver_id":"drv-2","rescue_id":"resc-1"}`},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodPost, "/v1/warehouse/ops/dispatch/rescue/propose?warehouse_id=wh-1", bytes.NewBufferString(tc.body))
				rr := httptest.NewRecorder()
				s.HandleOpsDispatchRescuePropose(rr, req)
				if rr.Code != http.StatusBadRequest {
					t.Fatalf("status = %d, want 400", rr.Code)
				}
				if !strings.Contains(rr.Body.String(), "invalid_request") {
					t.Fatalf("expected invalid_request, got %s", rr.Body.String())
				}
			})
		}
	})
}
