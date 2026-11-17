package order

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/soliq"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockCorrectiveProvider struct {
	lastReq FiscalCorrectiveRequest
	res     FiscalCreateResult
	err     error
}

func (m *mockCorrectiveProvider) Name() string {
	return "MOCK_CORRECTIVE"
}

func (m *mockCorrectiveProvider) CreateReceipt(_ context.Context, _ FiscalCreateRequest) (FiscalCreateResult, error) {
	return FiscalCreateResult{}, nil
}

func (m *mockCorrectiveProvider) GetSoliqClient() soliq.SoliqClient {
	return nil
}

func (m *mockCorrectiveProvider) CreateCorrectiveReceipt(_ context.Context, req FiscalCorrectiveRequest) (FiscalCreateResult, error) {
	m.lastReq = req
	if m.err != nil {
		return FiscalCreateResult{}, m.err
	}
	return m.res, nil
}

// 1. Unit test for tenant isolation in HandleGenerateCorrectiveFiscalReceipt (403 on cross-tenant).
func TestHandleGenerateCorrectiveFiscalReceipt_CrossTenantForbidden(t *testing.T) {
	repo := &testRepo{
		order: Order{
			OrderID:               "ord-tenant-victim",
			SupplierID:            "sup-victim-999",
			RetailerID:            "ret-victim-111",
			LatestFiscalReceiptID: "EHF-ORIG-12345",
			TotalMinor:            10000000,
			Currency:              "UZS",
		},
		found: true,
	}

	provider := &mockCorrectiveProvider{
		res: FiscalCreateResult{
			FiscalReceiptID: "EHF-CORR-0001",
			FiscalQR:        "https://ofd.soliq.uz/check?id=EHF-CORR-0001",
		},
	}

	svc := &Service{
		repo: repo,
		ofd:  provider,
		now:  func() time.Time { return time.Now().UTC() },
	}

	router := chi.NewRouter()
	router.Post("/v1/supplier/orders/{orderId}/fiscal/corrective", svc.HandleGenerateCorrectiveFiscalReceipt)

	// Attacker belongs to supplier "sup-attacker-666" (different tenant)
	claims := auth.Claims{
		Subject:    "user-attacker-1",
		SupplierID: "sup-attacker-666",
		Role:       auth.RoleAdmin,
	}

	reqBody := `{"correction_amount_minor": 5000000, "reason": "item_damage"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/orders/ord-tenant-victim/fiscal/corrective", bytes.NewBufferString(reqBody))
	req = req.WithContext(auth.WithClaims(req.Context(), claims))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Cross-tenant attempt must return 403 Forbidden
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for cross-tenant corrective receipt, got %d: %s", w.Code, w.Body.String())
	}

	var res map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to parse json response: %v", err)
	}
	if res["error"] != "forbidden" {
		t.Fatalf("expected error 'forbidden', got %q", res["error"])
	}
}

// 2. Unit test verifying GenerateCorrectiveFiscalReceipt fails with FailedPrecondition when no original receipt exists.
func TestGenerateCorrectiveFiscalReceipt_RequiresOriginalReceipt(t *testing.T) {
	repo := &testRepo{
		order: Order{
			OrderID:               "ord-no-original-receipt",
			SupplierID:            "sup-legit-123",
			RetailerID:            "ret-legit-456",
			LatestFiscalReceiptID: "", // No original fiscal receipt!
			TotalMinor:            15000000,
			Currency:              "UZS",
		},
		found: true,
	}

	provider := &mockCorrectiveProvider{
		res: FiscalCreateResult{
			FiscalReceiptID: "EHF-CORR-0002",
			FiscalQR:        "https://ofd.soliq.uz/check?id=EHF-CORR-0002",
		},
	}

	svc := &Service{
		repo: repo,
		ofd:  provider,
		now:  func() time.Time { return time.Now().UTC() },
	}

	ctx := auth.WithClaims(context.Background(), auth.Claims{
		Subject:    "admin-user",
		SupplierID: "sup-legit-123",
		Role:       auth.RoleAdmin,
	})

	_, err := svc.GenerateCorrectiveFiscalReceipt(ctx, "ord-no-original-receipt", CorrectiveFiscalReceiptRequest{
		CorrectionAmountMinor: 5000000,
		Reason:                "return",
	}, "admin-user")

	if err == nil {
		t.Fatal("expected GenerateCorrectiveFiscalReceipt to fail when no original receipt exists, got nil")
	}

	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition, got code=%v err=%v", st.Code(), err)
	}
	if !strings.Contains(st.Message(), "no_original_fiscal_receipt_to_correct") {
		t.Fatalf("expected 'no_original_fiscal_receipt_to_correct', got %q", st.Message())
	}
}

// 3. Contract test verifying provider delegation with real receipt ID/QR.
func TestGenerateCorrectiveFiscalReceipt_ProviderDelegation(t *testing.T) {
	const origReceipt = "EHF-REAL-ORIG-998877"
	const expectedCorrReceipt = "EHF-SOLIQ-CORR-554433"
	const expectedQR = "https://ofd.soliq.uz/check?id=EHF-SOLIQ-CORR-554433&corr=true"

	repo := &testRepo{
		order: Order{
			OrderID:               "ord-legit-receipt",
			SupplierID:            "sup-legit-123",
			RetailerID:            "ret-legit-456",
			LatestFiscalReceiptID: origReceipt,
			TotalMinor:            20000000,
			Currency:              "UZS",
		},
		found: true,
	}

	provider := &mockCorrectiveProvider{
		res: FiscalCreateResult{
			FiscalReceiptID: expectedCorrReceipt,
			FiscalQR:        expectedQR,
		},
	}

	svc := &Service{
		repo: repo,
		ofd:  provider,
		now:  func() time.Time { return time.Now().UTC() },
	}

	ctx := auth.WithClaims(context.Background(), auth.Claims{
		Subject:    "admin-user",
		SupplierID: "sup-legit-123",
		Role:       auth.RoleAdmin,
	})

	correctiveRow, err := svc.GenerateCorrectiveFiscalReceipt(ctx, "ord-legit-receipt", CorrectiveFiscalReceiptRequest{
		CorrectionAmountMinor: 8000000,
		Reason:                "partial_offload_shortage",
	}, "admin-user")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify provider was delegated to with correct parameters
	if provider.lastReq.OriginalReceiptID != origReceipt {
		t.Fatalf("provider called with OriginalReceiptID = %q, want %q", provider.lastReq.OriginalReceiptID, origReceipt)
	}
	if provider.lastReq.AmountMinor != 8000000 {
		t.Fatalf("provider called with AmountMinor = %d, want 8000000", provider.lastReq.AmountMinor)
	}
	if provider.lastReq.ReasonCode != "CORRECTIVE_PARTIAL_OFFLOAD_SHORTAGE" {
		t.Fatalf("provider called with ReasonCode = %q, want 'CORRECTIVE_PARTIAL_OFFLOAD_SHORTAGE'", provider.lastReq.ReasonCode)
	}

	// Verify returned receipt stores real Soliq receipt ID and QR, NOT synthetic "CORR-" string
	if correctiveRow.FiscalReceiptID != expectedCorrReceipt {
		t.Fatalf("expected real receipt id %q, got %q", expectedCorrReceipt, correctiveRow.FiscalReceiptID)
	}
	if correctiveRow.FiscalQR != expectedQR {
		t.Fatalf("expected real QR %q, got %q", expectedQR, correctiveRow.FiscalQR)
	}
	if strings.HasPrefix(correctiveRow.FiscalReceiptID, "CORR-") {
		t.Fatalf("EXPLOIT DETECTED: synthetic fake CORR- receipt ID generated instead of provider ID")
	}
}

// 4. Happy-path HTTP test for HandleGenerateCorrectiveFiscalReceipt.
func TestHandleGenerateCorrectiveFiscalReceipt_HappyPath(t *testing.T) {
	repo := &testRepo{
		order: Order{
			OrderID:               "ord-happy",
			SupplierID:            "sup-test",
			RetailerID:            "ret-test",
			LatestFiscalReceiptID: "EHF-ORIG-100",
			TotalMinor:            10000000,
			Currency:              "UZS",
		},
		found: true,
	}

	provider := &mockCorrectiveProvider{
		res: FiscalCreateResult{
			FiscalReceiptID: "EHF-CORR-100",
			FiscalQR:        "https://ofd.soliq.uz/check?id=EHF-CORR-100",
		},
	}

	svc := &Service{
		repo: repo,
		ofd:  provider,
		now:  func() time.Time { return time.Now().UTC() },
	}

	router := chi.NewRouter()
	router.Post("/v1/supplier/orders/{orderId}/fiscal/corrective", svc.HandleGenerateCorrectiveFiscalReceipt)

	claims := auth.Claims{
		Subject:    "admin-user",
		SupplierID: "sup-test",
		Role:       auth.RoleAdmin,
	}

	reqBody := `{"correction_amount_minor": 2000000, "reason": "damaged_packaging"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/orders/ord-happy/fiscal/corrective", bytes.NewBufferString(reqBody))
	req = req.WithContext(auth.WithClaims(req.Context(), claims))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	var res FiscalReceiptRow
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if res.FiscalReceiptID != "EHF-CORR-100" {
		t.Fatalf("expected FiscalReceiptID 'EHF-CORR-100', got %q", res.FiscalReceiptID)
	}
}
