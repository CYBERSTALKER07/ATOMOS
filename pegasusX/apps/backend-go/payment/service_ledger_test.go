package payment

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestHandleLedger_QueriesRepositoryWithSupplierScope(t *testing.T) {
	t.Parallel()

	repo := &paymentRepoStub{
		ledgerItems: []LedgerEntryRecord{{
			LedgerEntryID: "pledger_1",
			SupplierID:    "supplier-1",
			Gateway:       "ADYEN",
			EntryType:     "SESSION_PAYMENT_REQUIRED",
			AmountMinor:   1200,
			Currency:      "UZS",
			Source:        "payment.session",
			OccurredAt:    time.Unix(1_700_000_000, 0).UTC(),
			CreatedAt:     time.Unix(1_700_000_000, 0).UTC(),
		}},
	}
	svc := newPaymentServiceForExecutionTest(repo)

	req := httptest.NewRequest(http.MethodGet, "/v1/payment/ledger?order_id=order-1&session_id=session-1&gateway=adyen&entry_type=session_payment_required&occurred_from=2023-11-14T22:00:00Z&occurred_to=2023-11-14T22:30:00Z&limit=20", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{Role: auth.RoleAdmin, SupplierID: "supplier-1"}))
	res := httptest.NewRecorder()

	svc.HandleLedger(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	if repo.lastLedgerQuery.SupplierID != "supplier-1" {
		t.Fatalf("supplier scope = %q, want supplier-1", repo.lastLedgerQuery.SupplierID)
	}
	if repo.lastLedgerQuery.OrderID != "order-1" {
		t.Fatalf("order_id query = %q, want order-1", repo.lastLedgerQuery.OrderID)
	}
	if repo.lastLedgerQuery.SessionID != "session-1" {
		t.Fatalf("session_id query = %q, want session-1", repo.lastLedgerQuery.SessionID)
	}
	if repo.lastLedgerQuery.Limit != 20 {
		t.Fatalf("limit query = %d, want 20", repo.lastLedgerQuery.Limit)
	}
	if repo.lastLedgerQuery.Gateway != "ADYEN" {
		t.Fatalf("gateway query = %q, want ADYEN", repo.lastLedgerQuery.Gateway)
	}
	if repo.lastLedgerQuery.EntryType != "SESSION_PAYMENT_REQUIRED" {
		t.Fatalf("entry_type query = %q, want SESSION_PAYMENT_REQUIRED", repo.lastLedgerQuery.EntryType)
	}
	if repo.lastLedgerQuery.OccurredFrom == nil || repo.lastLedgerQuery.OccurredFrom.Format(time.RFC3339) != "2023-11-14T22:00:00Z" {
		t.Fatalf("occurred_from = %v, want 2023-11-14T22:00:00Z", repo.lastLedgerQuery.OccurredFrom)
	}
	if repo.lastLedgerQuery.OccurredTo == nil || repo.lastLedgerQuery.OccurredTo.Format(time.RFC3339) != "2023-11-14T22:30:00Z" {
		t.Fatalf("occurred_to = %v, want 2023-11-14T22:30:00Z", repo.lastLedgerQuery.OccurredTo)
	}

	var payload map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["count"] != float64(1) {
		t.Fatalf("count = %v, want 1", payload["count"])
	}
}

func TestHandleLedger_SupplierScopeMismatchForbidden(t *testing.T) {
	t.Parallel()

	repo := &paymentRepoStub{}
	svc := newPaymentServiceForExecutionTest(repo)

	req := httptest.NewRequest(http.MethodGet, "/v1/payment/ledger?supplier_id=supplier-2", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{Role: auth.RoleAdmin, SupplierID: "supplier-1"}))
	res := httptest.NewRecorder()

	svc.HandleLedger(res, req)

	if res.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusForbidden)
	}
}

func TestHandleLedger_InvalidLimitBadRequest(t *testing.T) {
	t.Parallel()

	repo := &paymentRepoStub{}
	svc := newPaymentServiceForExecutionTest(repo)

	req := httptest.NewRequest(http.MethodGet, "/v1/payment/ledger?limit=0", nil)
	res := httptest.NewRecorder()

	svc.HandleLedger(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusBadRequest)
	}
}

func TestHandleLedger_InvalidOccurredRangeBadRequest(t *testing.T) {
	t.Parallel()

	repo := &paymentRepoStub{}
	svc := newPaymentServiceForExecutionTest(repo)

	req := httptest.NewRequest(http.MethodGet, "/v1/payment/ledger?occurred_from=2023-11-14T22:30:00Z&occurred_to=2023-11-14T22:00:00Z", nil)
	res := httptest.NewRecorder()

	svc.HandleLedger(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusBadRequest)
	}
}
