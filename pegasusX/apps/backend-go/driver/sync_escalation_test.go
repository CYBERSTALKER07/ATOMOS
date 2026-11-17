package driver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func driverAuthedReq(method, path, body, driverID, supplierID string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	claims := auth.Claims{
		Subject:    driverID,
		Role:       auth.RoleDriver,
		SupplierID: supplierID,
	}
	ctx := auth.WithClaims(req.Context(), claims)
	ctx = auth.WithTenant(ctx, auth.TenantContext{SupplierID: supplierID, Source: "jwt"})
	return req.WithContext(ctx)
}

func TestHandleOpenFiscal_AndReturnComplete_NonBlockingRetryQueued(t *testing.T) {
	svc := NewService(ServiceConfig{
		OpenFiscal: func(_ context.Context, driverID string) (OpenFiscalSnapshot, error) {
			return OpenFiscalSnapshot{
				Count:            0,
				Frozen:           false,
				RetryQueuedCount: 2,
				FiscalIssueCount: 2,
				RetryOrderIDs:    []string{"ord-retry-1", "ord-retry-2"},
			}, nil
		},
		ReturnComplete: func(_ context.Context, driverID string) (ReturnCompleteResult, bool, error) {
			return ReturnCompleteResult{ManifestID: "man-1", Count: 2}, true, nil
		},
	})

	// 1. Open fiscal banner shows retry_queued_count=2 without freezing cash bag.
	rr := httptest.NewRecorder()
	svc.HandleOpenFiscal(rr, driverAuthedReq(http.MethodGet, "/v1/driver/open-fiscal", "", "drv-1", "sup-1"))
	if rr.Code != http.StatusOK {
		t.Fatalf("open-fiscal status=%d body=%s", rr.Code, rr.Body.String())
	}
	var openBody map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &openBody); err != nil {
		t.Fatalf("decode open-fiscal: %v", err)
	}
	if openBody["cash_bag_frozen"] != false {
		t.Fatalf("cash_bag_frozen=%v want false", openBody["cash_bag_frozen"])
	}
	if openBody["retry_queued_count"] != float64(2) || openBody["fiscal_issue_count"] != float64(2) {
		t.Fatalf("unexpected open-fiscal counts: %+v", openBody)
	}

	// 2. Return-complete succeeds even with retryable fiscal orders pending OFD recovery.
	rc := httptest.NewRecorder()
	req := driverAuthedReq(http.MethodPost, "/v1/fleet/driver/return-complete", `{}`, "drv-1", "sup-1")
	req.Header.Set("Idempotency-Key", "idem-rc-nonblock-1")
	svc.HandleDriverReturnComplete(rc, req)
	if rc.Code != http.StatusOK {
		t.Fatalf("return-complete status=%d want 200; body=%s", rc.Code, rc.Body.String())
	}
}

func TestHandleOpenFiscal_StrictOrUncollectedFreezes(t *testing.T) {
	svc := NewService(ServiceConfig{
		OpenFiscal: func(_ context.Context, driverID string) (OpenFiscalSnapshot, error) {
			return OpenFiscalSnapshot{
				Count:            1,
				OrderIDs:         []string{"ord-uncollected"},
				Frozen:           true,
				RetryQueuedCount: 1,
				FiscalIssueCount: 2,
				RetryOrderIDs:    []string{"ord-retry"},
			}, nil
		},
	})
	rc := httptest.NewRecorder()
	req := driverAuthedReq(http.MethodPost, "/v1/fleet/driver/return-complete", `{}`, "drv-1", "sup-1")
	req.Header.Set("Idempotency-Key", "idem-rc-block-1")
	svc.HandleDriverReturnComplete(rc, req)
	if rc.Code != http.StatusConflict {
		t.Fatalf("expected 409 conflict when blocking order exists, got %d body=%s", rc.Code, rc.Body.String())
	}
	var errBody map[string]any
	if err := json.Unmarshal(rc.Body.Bytes(), &errBody); err != nil {
		t.Fatalf("decode block response: %v", err)
	}
	if errBody["cash_bag_frozen"] != true || errBody["open_fiscal_count"] != float64(1) {
		t.Fatalf("unexpected block payload: %+v", errBody)
	}
}

func TestHandleSyncEscalation_PersistsAuditRecords(t *testing.T) {
	svc := NewService(ServiceConfig{
		Now: func() time.Time { return time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC) },
	})
	payload := `{
		"items": [
			{
				"id": "sq-1",
				"endpoint": "/v1/order/collect-cash",
				"order_id": "ord-99",
				"last_error": "HTTP 500 gateway timeout",
				"attempt_count": 5,
				"client_timestamp_iso": "2026-11-13T08:55:00Z"
			}
		]
	}`
	rr := httptest.NewRecorder()
	svc.HandleSyncEscalation(rr, driverAuthedReq(http.MethodPost, "/v1/driver/sync-escalation", payload, "drv-7", "sup-1"))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp struct {
		OK           bool   `json:"ok"`
		Escalated    int    `json:"escalated"`
		EscalationID string `json:"escalation_id"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !resp.OK || resp.Escalated != 1 || resp.EscalationID == "" {
		t.Fatalf("unexpected resp: %+v", resp)
	}
	recs := svc.ListSyncEscalations()
	if len(recs) != 1 || recs[0].DriverID != "drv-7" || recs[0].Item.OrderID != "ord-99" {
		t.Fatalf("unexpected stored escalation records: %+v", recs)
	}
}
