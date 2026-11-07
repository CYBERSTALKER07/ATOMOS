package payload

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/order"
)

type mockOrderExpectationReader struct {
	orders map[string]order.Order
}

func (m *mockOrderExpectationReader) GetOrder(ctx context.Context, orderID string) (order.Order, bool, error) {
	ord, ok := m.orders[orderID]
	return ord, ok, nil
}

func TestHandleRecommendReassign_PhysicalVolumeDerivation(t *testing.T) {
	repo := &payloadRepoSpy{}
	svc := newPayloadTestService(repo, &payloadCacheBackendSpy{})

	svc.mu.Lock()
	svc.ensureDemoDataLocked()
	// Clear any manifest-assigned volume for ord_payload_1
	for k := range svc.manifestOrders {
		for i := range svc.manifestOrders[k] {
			if svc.manifestOrders[k][i].OrderID == "ord_payload_1" {
				svc.manifestOrders[k][i].VolumeVU = 0
			}
		}
	}
	svc.orders[0].VolumeVU = 0
	svc.mu.Unlock()

	// Setup mock reader with 2 line items:
	// item 1: qty=2, unit_volume_vu=1.5 -> 3.0 VU
	// item 2: qty=1, unit_volume_vu=0.5 -> 0.5 VU
	// Total volume should be 3.5 VU (NOT total_minor=100_000_000 tiyin)
	reader := &mockOrderExpectationReader{
		orders: map[string]order.Order{
			"ord_payload_1": {
				OrderID:    "ord_payload_1",
				RetailerID: "ret_enterprise_42",
				TotalMinor: 100000000, // 1,000,000 UZS
				LineItems: []order.LineItem{
					{
						SKU:          "SKU-BEV-001",
						Quantity:     2,
						UnitVolumeVU: 1.5,
					},
					{
						SKU:          "SKU-BEV-002",
						Quantity:     1,
						UnitVolumeVU: 0.5,
					},
				},
			},
		},
	}
	svc.SetOrderExpectationReader(reader)

	req := httptest.NewRequest(http.MethodPost, "/v1/payloader/recommend-reassign", strings.NewReader(`{"order_id":"ord_payload_1"}`))
	req.Header.Set("Idempotency-Key", "test-reassign-vol-1")
	rr := httptest.NewRecorder()

	svc.HandleRecommendReassign(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	vol, ok := resp["order_volume_vu"].(float64)
	if !ok {
		t.Fatalf("order_volume_vu missing or not float64: %v", resp["order_volume_vu"])
	}
	if vol != 3.5 {
		t.Fatalf("expected physical volume 3.5 VU, got %v (must never equal financial minor units or arbitrary 10.0)", vol)
	}

	retailer, _ := resp["retailer_name"].(string)
	if retailer != "ret_enterprise_42" {
		t.Fatalf("expected retailer ret_enterprise_42, got %q", retailer)
	}
}

func TestHandleRecommendReassign_VolumeFloorFallback(t *testing.T) {
	repo := &payloadRepoSpy{}
	svc := newPayloadTestService(repo, &payloadCacheBackendSpy{})

	svc.mu.Lock()
	svc.ensureDemoDataLocked()
	for k := range svc.manifestOrders {
		for i := range svc.manifestOrders[k] {
			if svc.manifestOrders[k][i].OrderID == "ord_payload_1" {
				svc.manifestOrders[k][i].VolumeVU = 0
			}
		}
	}
	svc.orders[0].VolumeVU = 0
	svc.mu.Unlock()

	// No reader or empty line items -> must fall back to 1.0 VU floor (never 10.0, never money)
	svc.SetOrderExpectationReader(nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/payloader/recommend-reassign", strings.NewReader(`{"order_id":"ord_payload_1"}`))
	req.Header.Set("Idempotency-Key", "test-reassign-floor-1")
	rr := httptest.NewRecorder()

	svc.HandleRecommendReassign(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	vol, ok := resp["order_volume_vu"].(float64)
	if !ok || vol != 1.0 {
		t.Fatalf("expected 1.0 VU minimum floor, got %v", resp["order_volume_vu"])
	}
}
