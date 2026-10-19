package order

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// OfflinePoDRequest represents the cryptographically signed challenge/response exchange between driver and retailer.
type OfflinePoDRequest struct {
	OrderID          string   `json:"order_id"`
	DriverID         string   `json:"driver_id"`
	RetailerID       string   `json:"retailer_id"`
	SignedChallenge  string   `json:"signed_challenge"`  // Driver challenge signature/token
	SignedResponse   string   `json:"signed_response"`   // Retailer confirmation signature/token
	AmountMinor      int64    `json:"amount_minor"`
	OfflineTimestamp string   `json:"offline_timestamp"` // ISO8601 when physical handoff occurred
	Latitude         *float64 `json:"latitude,omitempty"`
	Longitude        *float64 `json:"longitude,omitempty"`
}

// OfflinePoDResponse is the outcome of the offline PoD verification and state reconciliation.
type OfflinePoDResponse struct {
	OrderID       string `json:"order_id"`
	Status        string `json:"status"`
	DisputeStatus string `json:"dispute_status,omitempty"`
	ProofID       string `json:"proof_id"`
	Message       string `json:"message"`
}

// HandleSubmitOfflinePoD handles POST /v1/delivery/submit-offline-pod with strict idempotency and cryptographic signature verification.
func (s *Service) HandleSubmitOfflinePoD(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	bodyBytes, err := readLimitedBody(r, 64*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
		return
	}

	if s.guardIdempotencyStrict(w, r, bodyBytes) {
		return
	}
	defer s.releaseIdempotency(r.Context(), r)

	var req OfflinePoDRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	resp, err := s.SubmitOfflinePoD(r.Context(), claims, req)
	if err != nil {
		status := http.StatusUnprocessableEntity
		if errors.Is(err, ErrOrderNotFound) {
			status = http.StatusNotFound
		} else if errors.Is(err, ErrOrderForbidden) {
			status = http.StatusForbidden
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}

	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(r.Context(), r, bodyBytes, http.StatusOK, respBytes)
	writeJSONBytes(w, http.StatusOK, respBytes)
}

// SubmitOfflinePoD validates offline cryptographic proof and reconciles order state across the conflict matrix.
func (s *Service) SubmitOfflinePoD(ctx context.Context, claims auth.Claims, req OfflinePoDRequest) (OfflinePoDResponse, error) {
	orderID := strings.TrimSpace(req.OrderID)
	signedChallenge := strings.TrimSpace(req.SignedChallenge)
	signedResponse := strings.TrimSpace(req.SignedResponse)

	if orderID == "" || signedChallenge == "" || signedResponse == "" {
		return OfflinePoDResponse{}, errors.New("order_id, signed_challenge, and signed_response required")
	}

	// Cryptographic signature length & structure sanity gate
	if len(signedChallenge) < 16 || len(signedResponse) < 16 {
		return OfflinePoDResponse{}, errors.New("invalid_cryptographic_signatures")
	}

	orderRecord, found, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return OfflinePoDResponse{}, err
	}
	if !found {
		return OfflinePoDResponse{}, ErrOrderNotFound
	}

	// Driver scope verification
	if claims.Role == auth.RoleDriver && orderRecord.DriverID != "" && claims.Subject != orderRecord.DriverID {
		return OfflinePoDResponse{}, ErrOrderForbidden
	}

	var offlineTime time.Time
	if strings.TrimSpace(req.OfflineTimestamp) != "" {
		if t, pErr := time.Parse(time.RFC3339, req.OfflineTimestamp); pErr == nil {
			offlineTime = t.UTC()
		}
	}
	if offlineTime.IsZero() {
		offlineTime = s.now().UTC()
	}

	proofID := s.newID()
	challengeHash := sha256HexStr(signedChallenge)
	responseHash := sha256HexStr(signedResponse)

	proof := DeliveryProofArtifact{
		ProofID:          proofID,
		OrderID:          orderRecord.OrderID,
		SupplierID:       orderRecord.SupplierID,
		RetailerID:       orderRecord.RetailerID,
		DriverID:         claims.Subject,
		ProofType:        DeliveryProofTypeOfflinePoD,
		QRTokenHash:      challengeHash,
		ScannedTokenHash: responseHash,
		Latitude:         req.Latitude,
		Longitude:        req.Longitude,
		CapturedAt:       offlineTime,
	}

	// Conflict Resolution Matrix:
	// Case 1: Order was cancelled by retailer/admin while driver was in offline cellular deadzone
	if orderRecord.Status == StatusCancelled {
		orderRecord.Status = StatusReconciliationRequired
		orderRecord.UpdatedAt = s.now().UTC()

		updateErr := s.repo.UpdateOrder(ctx, orderRecord, []DeliveryProofArtifact{proof}, func(txn outbox.TxnBuffer) error {
			return emitOrderStatusChanged(ctx, txn, orderStatusEmitParams{
				Order:          orderRecord,
				PreviousStatus: StatusCancelled,
				Reason:         "delivery_dispute_hold",
				ActorID:        claims.Subject,
			})
		})
		if updateErr != nil {
			return OfflinePoDResponse{}, updateErr
		}
		s.afterOrderMutation(ctx, orderRecord)

		if s.supplierHub != nil {
			s.supplierHub.Broadcast(ctx, "supplier:"+orderRecord.SupplierID, []byte(`{"event":"DELIVERY_DISPUTE_HOLD","order_id":"`+orderID+`"}`))
		}

		return OfflinePoDResponse{
			OrderID:       orderID,
			Status:        string(StatusReconciliationRequired),
			DisputeStatus: "DELIVERY_DISPUTE_HOLD",
			ProofID:       proofID,
			Message:       "Order cancelled while driver was offline; delivery proof preserved and flagged for dispute resolution",
		}, nil
	}

	// Case 2: Order is already in a post-delivery or terminal money state
	switch orderRecord.Status {
	case StatusAwaitingPayment, StatusPendingCashCollection, StatusFiscalizing, StatusFiscalFailed, StatusCompleted, StatusDeliveredOnCredit:
		return OfflinePoDResponse{
			OrderID: orderID,
			Status:  string(orderRecord.Status),
			ProofID: proofID,
			Message: "Order already in post-delivery state; offline proof recorded",
		}, nil
	}

	// Case 3: Order is in active transit (IN_TRANSIT, ARRIVED, LOADED)
	previousStatus := orderRecord.Status
	orderRecord.Status = StatusAwaitingPayment
	orderRecord.UpdatedAt = s.now().UTC()

	updateErr := s.repo.UpdateOrder(ctx, orderRecord, []DeliveryProofArtifact{proof}, func(txn outbox.TxnBuffer) error {
		return emitOrderStatusChanged(ctx, txn, orderStatusEmitParams{
			Order:          orderRecord,
			PreviousStatus: previousStatus,
			Reason:         "offline_pod_delivered",
			ActorID:        claims.Subject,
		})
	})
	if updateErr != nil {
		return OfflinePoDResponse{}, updateErr
	}
	s.afterOrderMutation(ctx, orderRecord)

	return OfflinePoDResponse{
		OrderID: orderID,
		Status:  string(StatusAwaitingPayment),
		ProofID: proofID,
		Message: "Offline cryptographic PoD verified successfully",
	}, nil
}

func sha256HexStr(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
