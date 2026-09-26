package driver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestHandleHistoryUsesQueryNotMemory(t *testing.T) {
	t.Parallel()
	called := false
	svc := NewService(ServiceConfig{
		Now: func() time.Time { return time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC) },
		HistoryQuery: func(_ context.Context, driverID string, since time.Time, limit int) ([]HistoryRow, error) {
			called = true
			if driverID != "drv-1" {
				t.Fatalf("driverID=%s", driverID)
			}
			if limit != driverHistoryLimit {
				t.Fatalf("limit=%d", limit)
			}
			if since.After(time.Date(2026, 4, 23, 0, 0, 0, 0, time.UTC)) {
				t.Fatalf("since too recent: %s", since)
			}
			return []HistoryRow{{
				OrderID:     "ord-9",
				Status:      "COMPLETED",
				TotalMinor:  1200,
				Currency:    "UZS",
				CompletedAt: "2026-05-20T00:00:00Z",
			}}, nil
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/driver/history", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{Role: auth.RoleDriver, Subject: "drv-1"}))
	rr := httptest.NewRecorder()
	svc.HandleHistory(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if !called {
		t.Fatal("expected history query")
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	rows, _ := body["rows"].([]any)
	if len(rows) != 1 {
		t.Fatalf("rows=%v", body["rows"])
	}
	row := rows[0].(map[string]any)
	if row["order_id"] != "ord-9" {
		t.Fatalf("row=%v", row)
	}
}

func TestHandleHistoryQueryErrorIsUnavailable(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{
		HistoryQuery: func(context.Context, string, time.Time, int) ([]HistoryRow, error) {
			return nil, errors.New("spanner down")
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/driver/history", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{Role: auth.RoleDriver, Subject: "drv-1"}))
	rr := httptest.NewRecorder()
	svc.HandleHistory(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d want 503 body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleHistoryNilQueryHonestEmpty(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{})
	req := httptest.NewRequest(http.MethodGet, "/v1/driver/history", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{Role: auth.RoleDriver, Subject: "drv-1"}))
	rr := httptest.NewRecorder()
	svc.HandleHistory(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	rows, _ := body["rows"].([]any)
	if len(rows) != 0 {
		t.Fatalf("rows=%v", rows)
	}
}
