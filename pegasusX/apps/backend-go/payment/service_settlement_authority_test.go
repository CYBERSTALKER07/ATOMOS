package payment

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestHandleSettlementAuthority_QueriesRepositoryWithFilters(t *testing.T) {
	t.Parallel()

	repo := &paymentRepoStub{
		settlementRows: []SettlementAuthorityRow{
			{
				Gateway:          "ADYEN",
				EntryType:        "WEBHOOK_PAID",
				Currency:         "UZS",
				EntryCount:       2,
				AmountMinorTotal: 3400,
				FirstOccurredAt:  time.Unix(1_700_000_000, 0).UTC(),
				LastOccurredAt:   time.Unix(1_700_000_900, 0).UTC(),
			},
		},
	}
	svc := newPaymentServiceForExecutionTest(repo)

	req := httptest.NewRequest(http.MethodGet, "/v1/payment/settlement/authority?gateway=adyen&entry_type=webhook_paid&occurred_from=2023-11-14T22:00:00Z&occurred_to=2023-11-14T22:30:00Z&group_limit=50", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{Role: auth.RoleAdmin, SupplierID: "supplier-1"}))
	res := httptest.NewRecorder()

	svc.HandleSettlementAuthority(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	if repo.lastSettlementQuery.SupplierID != "supplier-1" {
		t.Fatalf("supplier scope = %q, want supplier-1", repo.lastSettlementQuery.SupplierID)
	}
	if repo.lastSettlementQuery.Gateway != "ADYEN" {
		t.Fatalf("gateway = %q, want ADYEN", repo.lastSettlementQuery.Gateway)
	}
	if repo.lastSettlementQuery.EntryType != "WEBHOOK_PAID" {
		t.Fatalf("entry_type = %q, want WEBHOOK_PAID", repo.lastSettlementQuery.EntryType)
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

	var payload map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["count"] != float64(1) {
		t.Fatalf("count = %v, want 1", payload["count"])
	}
	if payload["entry_count_total"] != float64(2) {
		t.Fatalf("entry_count_total = %v, want 2", payload["entry_count_total"])
	}
}

func TestHandleSettlementAuthority_SupplierScopeMismatchForbidden(t *testing.T) {
	t.Parallel()

	repo := &paymentRepoStub{}
	svc := newPaymentServiceForExecutionTest(repo)

	req := httptest.NewRequest(http.MethodGet, "/v1/payment/settlement/authority?supplier_id=supplier-2", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{Role: auth.RoleAdmin, SupplierID: "supplier-1"}))
	res := httptest.NewRecorder()

	svc.HandleSettlementAuthority(res, req)

	if res.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusForbidden)
	}
}

func TestHandleSettlementAuthority_InvalidGroupLimitBadRequest(t *testing.T) {
	t.Parallel()

	repo := &paymentRepoStub{}
	svc := newPaymentServiceForExecutionTest(repo)

	req := httptest.NewRequest(http.MethodGet, "/v1/payment/settlement/authority?group_limit=0", nil)
	res := httptest.NewRecorder()

	svc.HandleSettlementAuthority(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusBadRequest)
	}
}

func TestHandleSettlementAuthority_InvalidOccurredRangeBadRequest(t *testing.T) {
	t.Parallel()

	repo := &paymentRepoStub{}
	svc := newPaymentServiceForExecutionTest(repo)

	req := httptest.NewRequest(http.MethodGet, "/v1/payment/settlement/authority?occurred_from=2023-11-14T22:30:00Z&occurred_to=2023-11-14T22:00:00Z", nil)
	res := httptest.NewRecorder()

	svc.HandleSettlementAuthority(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusBadRequest)
	}
}
