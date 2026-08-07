package order

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// HandleInitiateRefund: POST /v1/order/{orderID}/refunds
// Body: {"amount_minor": 0, "reason_code": "", "reason_text": "", "idempotency_key": ""}
// amount_minor 0/omitted = full remaining refundable. Supplier admin or
// warehouse admin only (route-gated); the supplier scope check happens against
// the order's SupplierId.
func (s *Service) HandleInitiateRefund(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	orderID := strings.TrimSpace(chiURLParam(r, "orderID"))
	var body struct {
		AmountMinor    int64  `json:"amount_minor"`
		ReasonCode     string `json:"reason_code"`
		ReasonText     string `json:"reason_text"`
		IdempotencyKey string `json:"idempotency_key"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	res, err := s.InitiateRefund(r.Context(), RefundRequest{
		OrderID:        orderID,
		AmountMinor:    body.AmountMinor,
		ReasonCode:     body.ReasonCode,
		ReasonText:     body.ReasonText,
		ActorID:        claims.Subject,
		IdempotencyKey: body.IdempotencyKey,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrRefundExceedsCaptured), errors.Is(err, ErrRefundCreditPortion), errors.Is(err, ErrRefundOrderState):
			writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		case strings.Contains(err.Error(), "not found"):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		default:
			s.log.ErrorContext(r.Context(), "refund failed", "order_id", orderID, "err", err)
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		}
		return
	}
	writeJSON(w, http.StatusOK, res)
}
