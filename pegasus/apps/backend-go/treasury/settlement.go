package treasury

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"backend-go/auth"
	kafkaEvents "backend-go/kafka"
	"backend-go/outbox"
	"backend-go/telemetry"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

// ── Settlement Report ───────────────────────────────────────────────────────
// GET /v1/supplier/settlement-report — Per-order settlement detail with payment proof.
//   Query params: ?from=2025-01-01&to=2025-12-31&status=PAID

type SettlementRow struct {
	OrderID          string `json:"order_id"`
	InvoiceID        string `json:"invoice_id"`
	RetailerID       string `json:"retailer_id"`
	Amount           int64  `json:"amount"`
	DeliveryFee      int64  `json:"delivery_fee"`
	PaymentMode      string `json:"payment_mode"`
	InvoiceStatus    string `json:"invoice_status"`
	FeeAmount        int64  `json:"fee_amount,omitempty"`
	NetPayoutAmount  int64  `json:"net_payout_amount,omitempty"`
	PayoutOwnerType  string `json:"payout_owner_type,omitempty"`
	PayoutOwnerID    string `json:"payout_owner_id,omitempty"`
	FeePolicy        string `json:"fee_policy_version,omitempty"`
	SettlementTarget string `json:"settlement_target,omitempty"`
	Currency         string `json:"currency"`
	PaidAt           string `json:"paid_at,omitempty"`
	CreatedAt        string `json:"created_at"`
}

type SettlementSummary struct {
	TotalPaid        int64  `json:"total_paid"`
	TotalPending     int64  `json:"total_pending"`
	TotalDeliveryFee int64  `json:"total_delivery_fee"`
	PaidCount        int    `json:"paid_count"`
	PendingCount     int    `json:"pending_count"`
	PeriodFrom       string `json:"period_from"`
	PeriodTo         string `json:"period_to"`
}

type SettlementReportResponse struct {
	Summary SettlementSummary `json:"summary"`
	Rows    []SettlementRow   `json:"rows"`
}

func HandleSettlementReport(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims == nil || claims.UserID == "" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		supplierID := claims.ResolveSupplierID()

		// Parse date range
		fromStr := r.URL.Query().Get("from")
		toStr := r.URL.Query().Get("to")

		from := time.Now().AddDate(0, -3, 0) // default: last 3 months
		to := time.Now()

		if fromStr != "" {
			if t, err := time.Parse("2006-01-02", fromStr); err == nil {
				from = t
			}
		}
		if toStr != "" {
			if t, err := time.Parse("2006-01-02", toStr); err == nil {
				to = t.Add(24*time.Hour - time.Second) // end of day
			}
		}

		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		sql := `SELECT mi.OrderId, mi.InvoiceId, o.RetailerId, mi.Total,
			     COALESCE(o.DeliveryFee, 0) AS DeliveryFee,
			     mi.PaymentMode, mi.CustodyStatus, mi.CollectedAt, mi.CreatedAt,
			     COALESCE(iss.FeeAmount, 0),
			     COALESCE(iss.NetPayoutAmount, 0),
			     COALESCE(iss.PayoutOwnerType, ''),
			     COALESCE(iss.PayoutOwnerId, ''),
			     COALESCE(iss.FeePolicyVersion, ''),
			     COALESCE(mi.SettlementTarget, ''),
			     COALESCE(o.Currency, 'UZS')
			FROM MasterInvoices mi
			JOIN Orders o ON mi.OrderId = o.OrderId
			LEFT JOIN (
				SELECT InvoiceId, SupplierId,
				       COALESCE(SUM(FeeAmount), 0) AS FeeAmount,
				       COALESCE(SUM(NetPayoutAmount), 0) AS NetPayoutAmount,
				       MAX(PayoutOwnerType) AS PayoutOwnerType,
				       MAX(PayoutOwnerId) AS PayoutOwnerId,
				       MAX(FeePolicyVersion) AS FeePolicyVersion
				FROM InvoiceSettlementSlices iss_outer
				WHERE NOT EXISTS (
				  SELECT 1 FROM InvoiceSettlementSlices iss_rev
				  WHERE iss_rev.InvoiceId = iss_outer.InvoiceId
				    AND iss_rev.RevisionOf = iss_outer.SliceId
				)
				GROUP BY InvoiceId, SupplierId
			) iss ON iss.InvoiceId = mi.InvoiceId AND iss.SupplierId = o.SupplierId
			LEFT JOIN Retailers ret ON o.RetailerId = ret.RetailerId
			WHERE o.SupplierId = @supplierId
			  AND mi.CreatedAt >= @fromDate
			  AND mi.CreatedAt <= @toDate`
		params := map[string]interface{}{
			"supplierId": supplierID,
			"fromDate":   from,
			"toDate":     to,
		}
		sql, params = auth.AppendRegionFilter(r.Context(), sql, params, "ret")
		sql += ` ORDER BY mi.CreatedAt DESC
				LIMIT 500`

		stmt := spanner.Statement{SQL: sql, Params: params}

		resp := SettlementReportResponse{
			Rows: []SettlementRow{},
			Summary: SettlementSummary{
				PeriodFrom: from.Format("2006-01-02"),
				PeriodTo:   to.Format("2006-01-02"),
			},
		}

		iter := client.Single().Query(ctx, stmt)
		defer iter.Stop()
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				http.Error(w, `{"error":"query_failed"}`, http.StatusInternalServerError)
				return
			}

			var sr SettlementRow
			var amount spanner.NullInt64
			var deliveryFee spanner.NullInt64
			var feeAmount spanner.NullInt64
			var netPayoutAmount spanner.NullInt64
			var invoiceID spanner.NullString
			var paymentMode spanner.NullString
			var custodyStatus spanner.NullString
			var payoutOwnerType spanner.NullString
			var payoutOwnerID spanner.NullString
			var feePolicyVersion spanner.NullString
			var settlementTarget spanner.NullString
			var currency spanner.NullString
			var collectedAt spanner.NullTime
			var createdAt time.Time

			if err := row.Columns(
				&sr.OrderID,
				&invoiceID,
				&sr.RetailerID,
				&amount,
				&deliveryFee,
				&paymentMode,
				&custodyStatus,
				&collectedAt,
				&createdAt,
				&feeAmount,
				&netPayoutAmount,
				&payoutOwnerType,
				&payoutOwnerID,
				&feePolicyVersion,
				&settlementTarget,
				&currency,
			); err != nil {
				slog.Error("treasury.settlement.decode_failed", "supplier_id", supplierID, "err", err)
				continue
			}

			sr.InvoiceID = invoiceID.StringVal
			sr.Amount = amount.Int64
			sr.DeliveryFee = deliveryFee.Int64
			sr.PaymentMode = paymentMode.StringVal
			sr.InvoiceStatus = custodyStatus.StringVal
			sr.FeeAmount = feeAmount.Int64
			sr.NetPayoutAmount = netPayoutAmount.Int64
			sr.PayoutOwnerType = payoutOwnerType.StringVal
			sr.PayoutOwnerID = payoutOwnerID.StringVal
			sr.FeePolicy = feePolicyVersion.StringVal
			sr.SettlementTarget = settlementTarget.StringVal
			sr.Currency = "UZS"
			if currency.Valid && strings.TrimSpace(currency.StringVal) != "" {
				sr.Currency = strings.ToUpper(strings.TrimSpace(currency.StringVal))
			}
			sr.CreatedAt = createdAt.Format(time.RFC3339)
			if collectedAt.Valid {
				sr.PaidAt = collectedAt.Time.Format(time.RFC3339)
			}

			// Aggregate summaries
			resp.Summary.TotalDeliveryFee += sr.DeliveryFee
			if custodyStatus.StringVal == "SETTLED" || custodyStatus.StringVal == "COLLECTED" {
				resp.Summary.TotalPaid += sr.Amount
				resp.Summary.PaidCount++
			} else {
				resp.Summary.TotalPending += sr.Amount
				resp.Summary.PendingCount++
			}

			resp.Rows = append(resp.Rows, sr)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

// ── Batch Settlement ────────────────────────────────────────────────────────
// POST /v1/treasury/batch-settle — Mark a batch of invoices as settled.

type BatchSettleRequest struct {
	InvoiceIDs []string `json:"invoice_ids"`
	Reference  string   `json:"reference"` // Bank transfer ref or payout batch ID
}

func HandleBatchSettle(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims == nil || claims.UserID == "" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		var req BatchSettleRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid_json"}`, http.StatusBadRequest)
			return
		}
		if len(req.InvoiceIDs) == 0 {
			http.Error(w, `{"error":"invoice_ids required"}`, http.StatusBadRequest)
			return
		}
		if len(req.InvoiceIDs) > 500 {
			http.Error(w, `{"error":"max 500 invoices per batch"}`, http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		now := time.Now()
		settled := 0

		_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			var mutations []*spanner.Mutation

			// 1. Bulk read invoices to verify ownership and current status
			keys := make([]spanner.KeySet, 0, len(req.InvoiceIDs))
			for _, id := range req.InvoiceIDs {
				keys = append(keys, spanner.Key{id})
			}

			iter := txn.Read(ctx, "MasterInvoices", spanner.KeySets(keys...), []string{"InvoiceId", "SupplierId", "CustodyStatus"})
			defer iter.Stop()

			foundCount := 0
			for {
				row, readErr := iter.Next()
				if readErr == iterator.Done {
					break
				}
				if readErr != nil {
					return readErr
				}

				var id, supplierID string
				var custodyStatus spanner.NullString
				if colErr := row.Columns(&id, &supplierID, &custodyStatus); colErr != nil {
					return colErr
				}

				foundCount++

				if supplierID != claims.ResolveSupplierID() {
					return fmt.Errorf("invoice %s does not belong to this supplier", id)
				}
				if custodyStatus.StringVal != "PENDING" {
					return fmt.Errorf("invoice %s is not PENDING (current: %s)", id, custodyStatus.StringVal)
				}
			}

			if foundCount != len(req.InvoiceIDs) {
				return fmt.Errorf("one or more invoices not found")
			}

			for _, invoiceID := range req.InvoiceIDs {
				invoiceSuffix := invoiceID
				if len(invoiceSuffix) > 8 {
					invoiceSuffix = invoiceSuffix[:8]
				}
				mutations = append(mutations,
					spanner.Update("MasterInvoices",
						[]string{"InvoiceId", "CustodyStatus", "CollectedAt", "UpdatedAt"},
						[]interface{}{invoiceID, "SETTLED", now, spanner.CommitTimestamp},
					),
				)
				// Audit log entry
				mutations = append(mutations,
					spanner.Insert("AuditLog",
						[]string{"LogId", "ActorId", "ActorRole", "Action", "ResourceType", "ResourceId", "Metadata", "CreatedAt"},
						[]interface{}{
							"AUDIT-" + now.Format("20060102150405") + "-" + invoiceSuffix + "-" + uuid.New().String()[:8],
							claims.UserID, claims.Role, "STATE_CHANGE", "INVOICE", invoiceID,
							`{"old_status":"PENDING","new_status":"SETTLED","reference":"` + req.Reference + `"}`,
							spanner.CommitTimestamp,
						},
					),
				)
			}
			settled = len(req.InvoiceIDs)
			return txn.BufferWrite(mutations)
		})

		if err != nil {
			slog.Error("treasury.batch_settle_failed", "supplier_id", claims.ResolveSupplierID(), "invoice_count", len(req.InvoiceIDs), "err", err)
			http.Error(w, `{"error":"settlement_failed"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "settled",
			"count":     settled,
			"reference": req.Reference,
		})
	}
}

// ── Invoice Status Override ─────────────────────────────────────────────────
// PATCH /v1/treasury/invoice/status — Manual status override with audit trail.

type InvoiceStatusRequest struct {
	InvoiceID string `json:"invoice_id"`
	Status    string `json:"status"` // SETTLED | PENDING | DISPUTED | WRITTEN_OFF
	Reason    string `json:"reason"`
}

func HandleInvoiceStatusOverride(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims == nil || claims.UserID == "" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		var req InvoiceStatusRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid_json"}`, http.StatusBadRequest)
			return
		}
		if req.InvoiceID == "" || req.Status == "" {
			http.Error(w, `{"error":"invoice_id and status required"}`, http.StatusBadRequest)
			return
		}

		validStatuses := map[string]bool{
			"SETTLED": true, "PENDING": true, "DISPUTED": true, "WRITTEN_OFF": true,
		}
		if !validStatuses[req.Status] {
			http.Error(w, `{"error":"invalid status — must be SETTLED|PENDING|DISPUTED|WRITTEN_OFF"}`, http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		now := time.Now()
		invoiceSuffix := req.InvoiceID
		if len(invoiceSuffix) > 8 {
			invoiceSuffix = invoiceSuffix[:8]
		}

		_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			// Read current status for audit
			row, err := txn.ReadRow(ctx, "MasterInvoices", spanner.Key{req.InvoiceID}, []string{"SupplierId", "CustodyStatus", "OrderId", "RetailerId"})
			if err != nil {
				return err
			}
			var oldStatus spanner.NullString
			var orderID spanner.NullString
			var retailerID spanner.NullString
			var supplierID string
			if err := row.Columns(&supplierID, &oldStatus, &orderID, &retailerID); err != nil {
				return err
			}

			if supplierID != claims.ResolveSupplierID() {
				return fmt.Errorf("invoice does not belong to this supplier")
			}

			mutations := []*spanner.Mutation{
				spanner.Update("MasterInvoices",
					[]string{"InvoiceId", "CustodyStatus"},
					[]interface{}{req.InvoiceID, req.Status},
				),
				spanner.Insert("AuditLog",
					[]string{"LogId", "ActorId", "ActorRole", "Action", "ResourceType", "ResourceId", "Metadata", "CreatedAt"},
					[]interface{}{
						"AUDIT-ISO-" + now.Format("20060102150405") + "-" + invoiceSuffix,
						claims.UserID, claims.Role, "STATE_CHANGE", "INVOICE", req.InvoiceID,
						`{"old_status":"` + oldStatus.StringVal + `","new_status":"` + req.Status + `","reason":"` + req.Reason + `"}`,
						spanner.CommitTimestamp,
					},
				),
			}
			if err := txn.BufferWrite(mutations); err != nil {
				return err
			}

			if req.Status != "DISPUTED" || oldStatus.StringVal == "DISPUTED" {
				return nil
			}

			orderIDVal := strings.TrimSpace(orderID.StringVal)
			retailerIDVal := strings.TrimSpace(retailerID.StringVal)
			var supplierIDVal string
			var driverIDVal string
			var sessionIDVal string

			if orderIDVal != "" {
				orderIter := txn.Query(ctx, spanner.Statement{
					SQL: `SELECT SupplierId, DriverId
					      FROM Orders
					      WHERE OrderId = @orderId
					      LIMIT 1`,
					Params: map[string]interface{}{"orderId": orderIDVal},
				})
				defer orderIter.Stop()
				orow, oerr := orderIter.Next()
				if oerr != nil && oerr != iterator.Done {
					return fmt.Errorf("load order context for invoice dispute: %w", oerr)
				}
				if oerr == nil {
					var supplierID spanner.NullString
					var driverID spanner.NullString
					if err := orow.Columns(&supplierID, &driverID); err != nil {
						return fmt.Errorf("decode order context for invoice dispute: %w", err)
					}
					supplierIDVal = strings.TrimSpace(supplierID.StringVal)
					driverIDVal = strings.TrimSpace(driverID.StringVal)
				}

				sessionIter := txn.Query(ctx, spanner.Statement{
					SQL: `SELECT SessionId
					      FROM DeliverySessions
					      WHERE OrderId = @orderId
					      ORDER BY UpdatedAt DESC
					      LIMIT 1`,
					Params: map[string]interface{}{"orderId": orderIDVal},
				})
				defer sessionIter.Stop()
				srow, serr := sessionIter.Next()
				if serr != nil && serr != iterator.Done {
					return fmt.Errorf("load delivery session context for invoice dispute: %w", serr)
				}
				if serr == nil {
					var sessionID spanner.NullString
					if err := srow.Columns(&sessionID); err != nil {
						return fmt.Errorf("decode delivery session context for invoice dispute: %w", err)
					}
					sessionIDVal = strings.TrimSpace(sessionID.StringVal)
				}
			}

			aggregateType := "DeliverySession"
			aggregateID := sessionIDVal
			if aggregateID == "" {
				aggregateType = "Order"
				aggregateID = orderIDVal
			}
			if aggregateID == "" {
				aggregateType = "Invoice"
				aggregateID = req.InvoiceID
			}

			if err := outbox.EmitJSON(
				txn,
				aggregateType,
				aggregateID,
				kafkaEvents.EventDeliveryDisputed,
				kafkaEvents.TopicMain,
				kafkaEvents.DeliveryDisputedEvent{
					SessionID:  sessionIDVal,
					OrderID:    orderIDVal,
					RetailerID: retailerIDVal,
					DriverID:   driverIDVal,
					SupplierID: supplierIDVal,
					Reason:     req.Reason,
					DisputedBy: claims.UserID,
					Timestamp:  now.UTC(),
				},
				telemetry.TraceIDFromContext(ctx),
			); err != nil {
				return fmt.Errorf("outbox emit DELIVERY_DISPUTED: %w", err)
			}

			return nil
		})

		if err != nil {
			slog.Error("treasury.invoice_status_override_failed", "invoice_id", req.InvoiceID, "status", req.Status, "err", err)
			http.Error(w, `{"error":"update_failed"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":     req.Status,
			"invoice_id": req.InvoiceID,
		})
	}
}
