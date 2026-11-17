package order

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
)

func TestSubmitOfflinePoD_SuccessAndProofCapture(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	o := deliveryTestOrder(StatusInTransit)
	o.OrderID = "ord-offline-pod-1"
	o.DriverID = "drv-pod-1"

	repo := &testRepo{
		found: true,
		order: o,
	}
	svc := newTestService(repo, now)
	idem := idempotency.NewInMemoryStore()
	svc.idem = idem

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := auth.WithClaims(req.Context(), auth.Claims{
				Subject:    "drv-pod-1",
				Role:       auth.RoleDriver,
				SupplierID: o.SupplierID,
			})
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	r.Post("/v1/delivery/submit-offline-pod", svc.HandleSubmitOfflinePoD)

	podReq := OfflinePoDRequest{
		OrderID:          o.OrderID,
		DriverID:         "drv-pod-1",
		RetailerID:       o.RetailerID,
		SignedChallenge:  "challenge_sig_valid_driver_token_12345",
		SignedResponse:   "response_sig_valid_retailer_token_67890",
		AmountMinor:      10000,
		OfflineTimestamp: now.Format(time.RFC3339),
	}
	bodyBytes, _ := json.Marshal(podReq)
	idemKey := "key-offline-pod-001"

	// 1. Missing Idempotency-Key -> 400
	reqMissing := httptest.NewRequest(http.MethodPost, "/v1/delivery/submit-offline-pod", bytes.NewReader(bodyBytes))
	reqMissing.Header.Set("Content-Type", "application/json")
	wMissing := httptest.NewRecorder()
	r.ServeHTTP(wMissing, reqMissing)
	if wMissing.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing idempotency key, got %d body=%s", wMissing.Code, wMissing.Body.String())
	}

	// 2. Submit with Idempotency-Key -> 200 OK & AWAITING_PAYMENT
	req1 := httptest.NewRequest(http.MethodPost, "/v1/delivery/submit-offline-pod", bytes.NewReader(bodyBytes))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Idempotency-Key", idemKey)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d body=%s", w1.Code, w1.Body.String())
	}

	var resp1 OfflinePoDResponse
	if err := json.Unmarshal(w1.Body.Bytes(), &resp1); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if resp1.Status != string(StatusAwaitingPayment) {
		t.Fatalf("expected status AWAITING_PAYMENT, got %s", resp1.Status)
	}
	if resp1.ProofID == "" {
		t.Fatalf("expected proof_id to be populated")
	}

	// Verify proof was captured in repository
	if len(repo.lastProofs) == 0 {
		t.Fatalf("expected delivery proof artifact to be saved in repository")
	}
	proof := repo.lastProofs[0]
	if proof.ProofType != DeliveryProofTypeOfflinePoD {
		t.Fatalf("expected proof type %s, got %s", DeliveryProofTypeOfflinePoD, proof.ProofType)
	}
	if proof.QRTokenHash != sha256HexStr(podReq.SignedChallenge) {
		t.Fatalf("mismatched challenge hash")
	}
	if proof.ScannedTokenHash != sha256HexStr(podReq.SignedResponse) {
		t.Fatalf("mismatched response hash")
	}

	// 3. Replay with same Idempotency-Key -> Cached 200 response
	req2 := httptest.NewRequest(http.MethodPost, "/v1/delivery/submit-offline-pod", bytes.NewReader(bodyBytes))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Idempotency-Key", idemKey)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected replayed 200 OK, got %d body=%s", w2.Code, w2.Body.String())
	}
}

func TestSubmitOfflinePoD_CancelledOrderDisputeHold(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	// Order was cancelled while driver was offline
	o := deliveryTestOrder(StatusCancelled)
	o.OrderID = "ord-cancelled-conflict"
	o.DriverID = "drv-pod-1"

	repo := &testRepo{
		found: true,
		order: o,
	}
	svc := newTestService(repo, now)

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := auth.WithClaims(req.Context(), auth.Claims{
				Subject:    "drv-pod-1",
				Role:       auth.RoleDriver,
				SupplierID: o.SupplierID,
			})
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	r.Post("/v1/delivery/submit-offline-pod", svc.HandleSubmitOfflinePoD)

	podReq := OfflinePoDRequest{
		OrderID:          o.OrderID,
		DriverID:         "drv-pod-1",
		RetailerID:       o.RetailerID,
		SignedChallenge:  "driver_offline_challenge_xyz_12345",
		SignedResponse:   "retailer_offline_response_abc_67890",
		AmountMinor:      15000,
		OfflineTimestamp: now.Format(time.RFC3339),
	}
	bodyBytes, _ := json.Marshal(podReq)

	req := httptest.NewRequest(http.MethodPost, "/v1/delivery/submit-offline-pod", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "key-dispute-001")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for dispute hold, got %d body=%s", w.Code, w.Body.String())
	}

	var resp OfflinePoDResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if resp.Status != string(StatusReconciliationRequired) {
		t.Fatalf("expected RECONCILIATION_REQUIRED, got %s", resp.Status)
	}
	if resp.DisputeStatus != "DELIVERY_DISPUTE_HOLD" {
		t.Fatalf("expected DELIVERY_DISPUTE_HOLD, got %s", resp.DisputeStatus)
	}
	if repo.captured.Status != StatusReconciliationRequired {
		t.Fatalf("captured status want RECONCILIATION_REQUIRED, got %s", repo.captured.Status)
	}
}

func TestSubmitOfflinePoD_InvalidSignatures(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	o := deliveryTestOrder(StatusInTransit)
	repo := &testRepo{found: true, order: o}
	svc := newTestService(repo, now)

	claims := auth.Claims{Subject: o.DriverID, Role: auth.RoleDriver, SupplierID: o.SupplierID}

	// Short / invalid signatures
	_, err := svc.SubmitOfflinePoD(context.Background(), claims, OfflinePoDRequest{
		OrderID:         o.OrderID,
		SignedChallenge: "short",
		SignedResponse:  "short",
	})
	if err == nil || err.Error() != "invalid_cryptographic_signatures" {
		t.Fatalf("expected invalid_cryptographic_signatures, got %v", err)
	}
}
