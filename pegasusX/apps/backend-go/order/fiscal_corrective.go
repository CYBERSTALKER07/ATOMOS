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
	"google.golang.org/api/iterator"
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
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return nil, status.Error(codes.InvalidArgument, "order_id_required")
	}
	if req.CorrectionAmountMinor <= 0 {
		return nil, status.Error(codes.InvalidArgument, "correction_amount_must_be_positive")
	}

	if s == nil || s.repo == nil {
		return nil, status.Error(codes.Internal, "repository_required")
	}

	orderRow, ok, err := s.repo.GetOrder(ctx, orderID)
	if err != nil || !ok {
		return nil, status.Error(codes.NotFound, "order_not_found")
	}
	if !auth.EntitySupplierAllowed(ctx, orderRow.SupplierID) {
		return nil, status.Error(codes.PermissionDenied, "forbidden_cross_tenant")
	}

	originalReceiptID := strings.TrimSpace(orderRow.LatestFiscalReceiptID)
	if originalReceiptID == "" {
		if s.spannerClient != nil {
			stmt := spanner.Statement{
				SQL: `SELECT FiscalReceiptId FROM OrderFiscalReceipts 
				      WHERE OrderId = @order_id AND Status = 'SUCCESS' 
				      ORDER BY CreatedAt DESC LIMIT 1`,
				Params: map[string]interface{}{"order_id": orderID},
			}
			iter := s.spannerClient.Single().Query(ctx, stmt)
			defer iter.Stop()
			row, err := iter.Next()
			if err == iterator.Done {
				return nil, status.Error(codes.FailedPrecondition, "no_original_fiscal_receipt_to_correct")
			}
			if err != nil {
				return nil, status.Errorf(codes.Internal, "query original receipt: %v", err)
			}
			var origID spanner.NullString
			if err := row.Columns(&origID); err != nil || !origID.Valid || strings.TrimSpace(origID.StringVal) == "" {
				return nil, status.Error(codes.FailedPrecondition, "no_original_fiscal_receipt_to_correct")
			}
			originalReceiptID = strings.TrimSpace(origID.StringVal)
		} else {
			return nil, status.Error(codes.FailedPrecondition, "no_original_fiscal_receipt_to_correct")
		}
	}

	cp, ok := s.fiscalProvider().(CorrectiveFiscalProvider)
	if !ok {
		return nil, status.Error(codes.FailedPrecondition, "fiscal_provider_no_corrective_support")
	}
	attemptID := uuid.NewString()
	reasonCode := fmt.Sprintf("CORRECTIVE_%s", strings.ToUpper(strings.TrimSpace(req.Reason)))

	res, cErr := cp.CreateCorrectiveReceipt(ctx, FiscalCorrectiveRequest{
		AttemptID:         attemptID,
		OrderID:           orderID,
		SupplierID:        orderRow.SupplierID,
		RetailerID:        orderRow.RetailerID,
		OriginalReceiptID: originalReceiptID,
		AmountMinor:       req.CorrectionAmountMinor,
		Currency:          orderRow.Currency,
		ReasonCode:        reasonCode,
	})
	if cErr != nil {
		return nil, status.Errorf(codes.Internal, "corrective_fiscal_submission_failed: %v", cErr)
	}

	method := strings.TrimSpace(req.PaymentMethod)
	if method == "" {
		method = string(MethodCreditNote)
	}

	now := s.now().UTC()
	correctiveRow := FiscalReceiptRow{
		OrderID:         orderID,
		AttemptID:       attemptID,
		SupplierID:      orderRow.SupplierID,
		RetailerID:      orderRow.RetailerID,
		Provider:        s.ProviderName(),
		Status:          FiscalAttemptSuccess,
		FiscalReceiptID: res.FiscalReceiptID,
		FiscalQR:        res.FiscalQR,
		AmountMinor:     req.CorrectionAmountMinor,
		Currency:        orderRow.Currency,
		PaymentMethod:   method,
		ReasonCode:      reasonCode,
		ActorID:         actorID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if s.spannerClient != nil {
		err = spannerutils.RunReadWriteTransaction(ctx, s.spannerClient, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			cols := map[string]any{
				"OrderId":         orderID,
				"AttemptId":       attemptID,
				"SupplierId":      orderRow.SupplierID,
				"RetailerId":      orderRow.RetailerID,
				"Provider":        s.ProviderName(),
				"Status":          FiscalAttemptSuccess,
				"FiscalReceiptId": res.FiscalReceiptID,
				"FiscalQR":        nullableString(res.FiscalQR),
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
				"type":                    events.EventFiscalReceiptSucceeded,
				"order_id":                orderID,
				"attempt_id":              attemptID,
				"supplier_id":             orderRow.SupplierID,
				"retailer_id":             orderRow.RetailerID,
				"status":                  FiscalAttemptSuccess,
				"fiscal_receipt_id":       res.FiscalReceiptID,
				"correction_amount_minor": req.CorrectionAmountMinor,
				"reason_code":             reasonCode,
				"is_corrective":           true,
				"timestamp":               now.Format(time.RFC3339Nano),
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
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	res, err := s.GenerateCorrectiveFiscalReceipt(r.Context(), orderID, req, claims.Subject)
	if err != nil {
		st, _ := status.FromError(err)
		switch st.Code() {
		case codes.PermissionDenied:
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		case codes.NotFound:
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "order_not_found"})
		case codes.InvalidArgument:
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": st.Message()})
		case codes.FailedPrecondition:
			writeJSON(w, http.StatusPreconditionFailed, map[string]string{"error": st.Message()})
		default:
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		}
		return
	}
	writeJSON(w, http.StatusOK, res)
}
