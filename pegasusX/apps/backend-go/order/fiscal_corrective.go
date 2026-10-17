package order

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CorrectiveFiscalReceiptRequest specifies corrective receipt parameters.
type CorrectiveFiscalReceiptRequest struct {
	CorrectionAmountMinor int64  `json:"correction_amount_minor"`
	Reason                string `json:"reason"`
	PaymentMethod         string `json:"payment_method,omitempty"`
}

// GenerateCorrectiveFiscalReceipt generates an audited negative/corrective fiscal receipt row linked to the order.
func (s *Service) GenerateCorrectiveFiscalReceipt(ctx context.Context, orderID string, req CorrectiveFiscalReceiptRequest, actorID string) (*FiscalReceiptRow, error) {
	if s.spannerClient == nil {
		return nil, fmt.Errorf("spanner required")
	}
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return nil, status.Error(codes.InvalidArgument, "order_id_required")
	}
	if req.CorrectionAmountMinor <= 0 {
		return nil, status.Error(codes.InvalidArgument, "correction_amount_must_be_positive")
	}

	var correctiveRow FiscalReceiptRow
	now := s.now().UTC()

	err := spannerutils.RunReadWriteTransaction(ctx, s.spannerClient, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		orderRow, ok, err := s.repo.GetOrderTxn(ctx, txn, orderID)
		if err != nil || !ok {
			return status.Error(codes.NotFound, "order_not_found")
		}

		attemptID := uuid.NewString()
		receiptID := fmt.Sprintf("CORR-%s-%d", orderID, now.Unix())
		qrCode := fmt.Sprintf("https://ofd.soliq.uz/check?id=%s&corr=true", receiptID)
		reasonCode := fmt.Sprintf("CORRECTIVE_%s", strings.ToUpper(strings.TrimSpace(req.Reason)))
		method := strings.TrimSpace(req.PaymentMethod)
		if method == "" {
			method = string(MethodCreditNote)
		}

		correctiveRow = FiscalReceiptRow{
			OrderID:         orderID,
			AttemptID:       attemptID,
			SupplierID:      orderRow.SupplierID,
			RetailerID:      orderRow.RetailerID,
			Provider:        s.ProviderName(),
			Status:          FiscalAttemptSuccess,
			FiscalReceiptID: receiptID,
			FiscalQR:        qrCode,
			AmountMinor:     req.CorrectionAmountMinor,
			Currency:        orderRow.Currency,
			PaymentMethod:   method,
			ReasonCode:      reasonCode,
			ActorID:         actorID,
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		cols := map[string]any{
			"OrderId":         orderID,
			"AttemptId":       attemptID,
			"SupplierId":      orderRow.SupplierID,
			"RetailerId":      orderRow.RetailerID,
			"Provider":        s.ProviderName(),
			"Status":          FiscalAttemptSuccess,
			"FiscalReceiptId": receiptID,
			"FiscalQR":        qrCode,
			"AmountMinor":     req.CorrectionAmountMinor,
			"Currency":        orderRow.Currency,
			"PaymentMethod":   method,
			"ReasonCode":      reasonCode,
			"ActorId":         actorID,
			"CreatedAt":       spanner.CommitTimestamp,
			"UpdatedAt":       spanner.CommitTimestamp,
		}

		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.InsertMap("OrderFiscalReceipts", cols),
		}); err != nil {
			return err
		}

		buf := &spannerTxnBuffer{}
		if err := outbox.EmitJSON(ctx, buf, events.AggregateOrder, orderID, events.TopicMain, map[string]any{
			"type":                     events.EventFiscalReceiptSucceeded,
			"order_id":                 orderID,
			"attempt_id":               attemptID,
			"supplier_id":              orderRow.SupplierID,
			"retailer_id":              orderRow.RetailerID,
			"status":                   FiscalAttemptSuccess,
			"fiscal_receipt_id":        receiptID,
			"correction_amount_minor":  req.CorrectionAmountMinor,
			"reason_code":              reasonCode,
			"is_corrective":            true,
			"timestamp":                now.Format(time.RFC3339Nano),
		}); err != nil {
			return err
		}

		for _, m := range bufferedOutboxMutations(buf, now) {
			if err := txn.BufferWrite([]*spanner.Mutation{m}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &correctiveRow, nil
}

// HandleGenerateCorrectiveFiscalReceipt serves POST /v1/supplier/orders/{orderId}/fiscal/corrective.
func (s *Service) HandleGenerateCorrectiveFiscalReceipt(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || (claims.Role != auth.RoleAdmin && claims.Role != auth.RoleWarehouseAdmin) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	orderID := chi.URLParam(r, "orderId")
	if orderID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id_required"})
		return
	}

	var req CorrectiveFiscalReceiptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	res, err := s.GenerateCorrectiveFiscalReceipt(r.Context(), orderID, req, claims.Subject)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, res)
}
