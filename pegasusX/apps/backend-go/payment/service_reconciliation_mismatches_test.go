package payment

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestHandleReconciliationMismatches_UsesSummaryRowsAndFilters(t *testing.T) {
	t.Parallel()

	repo := &paymentRepoStub{
		settlementRows: []SettlementAuthorityRow{
			{
				Gateway:          "ADYEN",
				EntryType:        "WEBHOOK_PAID",
				Currency:         "UZS",
				EntryCount:       3,
				AmountMinorTotal: 10000,
				FirstOccurredAt:  time.Unix(1_700_000_000, 0).UTC(),
				LastOccurredAt:   time.Unix(1_700_000_400, 0).UTC(),
			},
			{
				Gateway:          "ADYEN",
				EntryType:        "CHARGEBACK_RECORDED",
				Currency:         "UZS",
				EntryCount:       1,
				AmountMinorTotal: 2500,
				FirstOccurredAt:  time.Unix(1_700_000_100, 0).UTC(),
				LastOccurredAt:   time.Unix(1_700_000_100, 0).UTC(),
			},
			{
				Gateway:          "ADYEN",
				EntryType:        "CHARGEBACK_REVERSAL_RECORDED",
				Currency:         "UZS",
				EntryCount:       1,
				AmountMinorTotal: 500,
				FirstOccurredAt:  time.Unix(1_700_000_250, 0).UTC(),
				LastOccurredAt:   time.Unix(1_700_000_250, 0).UTC(),
			},
		},
	}
	svc := newPaymentServiceForExecutionTest(repo)

	req := httptest.NewRequest(http.MethodGet, "/v1/payment/reconciliation/mismatches?gateway=adyen&occurred_from=2023-11-14T22:00:00Z&occurred_to=2023-11-14T22:30:00Z&group_limit=50&mismatch_threshold_minor=1000", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{Role: auth.RoleAdmin, SupplierID: "supplier-1"}))
	res := httptest.NewRecorder()

	svc.HandleReconciliationMismatches(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	if repo.lastSettlementQuery.SupplierID != "supplier-1" {
		t.Fatalf("supplier scope = %q, want supplier-1", repo.lastSettlementQuery.SupplierID)
	}
	if repo.lastSettlementQuery.Gateway != "ADYEN" {
		t.Fatalf("gateway filter = %q, want ADYEN", repo.lastSettlementQuery.Gateway)
	}
	if repo.lastSettlementQuery.GroupLimit != 50 {
		t.Fatalf("group_limit = %d, want 50", repo.lastSettlementQuery.GroupLimit)
	}
	if repo.lastSettlementQuery.OccurredFrom == nil || repo.lastSettlementQuery.OccurredFrom.Format(time.RFC3339) != "2023-11-14T22:00:00Z" {
		t.Fatalf("occurred_from = %v, want 2023-11-14T22:00:00Z", repo.lastSettlementQuery.OccurredFrom)
	}
	if repo.lastSettlementQuery.OccurredTo == nil || repo.lastSettlementQuery.OccurredTo.Format(time.RFC3339) != "2023-11-14T22:30:00Z" {
		t.Fatalf("occurred_to = %v, want 2023-11-14T22:30:00Z", repo.lastSettlementQuery.OccurredTo)
	}

	var payload struct {
		Count int                         `json:"count"`
		Items []ReconciliationMismatchRow `json:"items"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Count != 1 {
		t.Fatalf("count = %d, want 1", payload.Count)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("items len = %d, want 1", len(payload.Items))
	}
	if payload.Items[0].Gateway != "ADYEN" {
		t.Fatalf("item.gateway = %q, want ADYEN", payload.Items[0].Gateway)
	}
	if payload.Items[0].NetAmount != 8000 {
		t.Fatalf("item.net_amount_minor = %d, want 8000", payload.Items[0].NetAmount)
	}
	if payload.Items[0].CreditAmount != 10500 {
		t.Fatalf("item.credit_amount_minor_total = %d, want 10500", payload.Items[0].CreditAmount)
	}
	if payload.Items[0].DebitAmount != 2500 {
		t.Fatalf("item.debit_amount_minor_total = %d, want 2500", payload.Items[0].DebitAmount)
	}
}

func TestHandleReconciliationMismatches_ThresholdFiltersOutBalancedRows(t *testing.T) {
	t.Parallel()

	repo := &paymentRepoStub{
		settlementRows: []SettlementAuthorityRow{
			{
				Gateway:          "GLOBAL_PAY",
				EntryType:        "WEBHOOK_PAID",
				Currency:         "UZS",
				EntryCount:       1,
				AmountMinorTotal: 2000,
				FirstOccurredAt:  time.Unix(1_700_000_000, 0).UTC(),
				LastOccurredAt:   time.Unix(1_700_000_000, 0).UTC(),
			},
			{
				Gateway:          "GLOBAL_PAY",
				EntryType:        "WEBHOOK_REFUND",
				Currency:         "UZS",
				EntryCount:       1,
				AmountMinorTotal: 2000,
				FirstOccurredAt:  time.Unix(1_700_000_100, 0).UTC(),
				LastOccurredAt:   time.Unix(1_700_000_100, 0).UTC(),
			},
		},
	}
	svc := newPaymentServiceForExecutionTest(repo)

	req := httptest.NewRequest(http.MethodGet, "/v1/payment/reconciliation/mismatches?mismatch_threshold_minor=100", nil)
	res := httptest.NewRecorder()

	svc.HandleReconciliationMismatches(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}

	var payload map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["count"] != float64(0) {
		t.Fatalf("count = %v, want 0", payload["count"])
	}
}

func TestHandleReconciliationMismatches_SupplierScopeMismatchForbidden(t *testing.T) {
	t.Parallel()

	repo := &paymentRepoStub{}
	svc := newPaymentServiceForExecutionTest(repo)

	req := httptest.NewRequest(http.MethodGet, "/v1/payment/reconciliation/mismatches?supplier_id=supplier-2", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{Role: auth.RoleAdmin, SupplierID: "supplier-1"}))
	res := httptest.NewRecorder()

	svc.HandleReconciliationMismatches(res, req)

	if res.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusForbidden)
	}
}

func TestHandleReconciliationMismatches_InvalidThresholdBadRequest(t *testing.T) {
	t.Parallel()

	repo := &paymentRepoStub{}
	svc := newPaymentServiceForExecutionTest(repo)

	req := httptest.NewRequest(http.MethodGet, "/v1/payment/reconciliation/mismatches?mismatch_threshold_minor=-1", nil)
	res := httptest.NewRecorder()

	svc.HandleReconciliationMismatches(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusBadRequest)
	}
}
