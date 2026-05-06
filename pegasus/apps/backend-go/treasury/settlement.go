package treasury

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"backend-go/auth"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ── Settlement Report ───────────────────────────────────────────────────────
// GET /v1/supplier/settlement-report — Per-order settlement detail with payment proof.
//   Query params: ?from=2025-01-01&to=2025-12-31&status=PAID

type SettlementRow struct {
	OrderID       string `json:"order_id"`
	InvoiceID     string `json:"invoice_id"`
	RetailerID    string `json:"retailer_id"`
	Amount        int64  `json:"amount"`
	DeliveryFee   int64  `json:"delivery_fee"`
	PaymentMode   string `json:"payment_mode"`
	InvoiceStatus string `json:"invoice_status"`
	PaidAt        string `json:"paid_at,omitempty"`
	CreatedAt     string `json:"created_at"`
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

		stmt := spanner.Statement{
			SQL: `SELECT mi.OrderId, mi.InvoiceId, o.RetailerId, mi.Total,
				     COALESCE(o.DeliveryFee, 0) AS DeliveryFee,
				     mi.PaymentMode, mi.CustodyStatus, mi.CollectedAt, mi.CreatedAt
				FROM MasterInvoices mi
				JOIN Orders o ON mi.OrderId = o.OrderId
				WHERE o.SupplierId = @supplierId
				  AND mi.CreatedAt >= @fromDate
				  AND mi.CreatedAt <= @toDate
				ORDER BY mi.CreatedAt DESC
				LIMIT 500`,
			Params: map[string]interface{}{
				"supplierId": supplierID,
				"fromDate":   from,
				"toDate":     to,
			},
		}

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
			var invoiceID spanner.NullString
			var paymentMode spanner.NullString
			var custodyStatus spanner.NullString
			var collectedAt spanner.NullTime
			var createdAt time.Time

			if err := row.Columns(&sr.OrderID, &invoiceID, &sr.RetailerID, &amount, &deliveryFee, &paymentMode, &custodyStatus, &collectedAt, &createdAt); err != nil {
				slog.Error("treasury.settlement.decode_failed", "supplier_id", supplierID, "err", err)
				continue
			}

			sr.InvoiceID = invoiceID.StringVal
			sr.Amount = amount.Int64
			sr.DeliveryFee = deliveryFee.Int64
			sr.PaymentMode = paymentMode.StringVal
			sr.InvoiceStatus = custodyStatus.StringVal
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
			for _, invoiceID := range req.InvoiceIDs {
				mutations = append(mutations,
					spanner.Update("MasterInvoices",
						[]string{"InvoiceId", "CustodyStatus", "CollectedAt"},
						[]interface{}{invoiceID, "SETTLED", now},
					),
				)
				// Audit log entry
				mutations = append(mutations,
					spanner.Insert("AuditLog",
						[]string{"LogId", "ActorId", "ActorRole", "Action", "ResourceType", "ResourceId", "Metadata", "CreatedAt"},
						[]interface{}{
							"AUDIT-" + now.Format("20060102150405") + "-" + invoiceID[:8],
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
		_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			// Read current status for audit
			row, err := txn.ReadRow(ctx, "MasterInvoices", spanner.Key{req.InvoiceID}, []string{"CustodyStatus"})
			if err != nil {
				return err
			}
			var oldStatus spanner.NullString
			if err := row.Columns(&oldStatus); err != nil {
				return err
			}

			mutations := []*spanner.Mutation{
				spanner.Update("MasterInvoices",
					[]string{"InvoiceId", "CustodyStatus"},
					[]interface{}{req.InvoiceID, req.Status},
				),
				spanner.Insert("AuditLog",
					[]string{"LogId", "ActorId", "ActorRole", "Action", "ResourceType", "ResourceId", "Metadata", "CreatedAt"},
					[]interface{}{
						"AUDIT-ISO-" + now.Format("20060102150405") + "-" + req.InvoiceID[:8],
						claims.UserID, claims.Role, "STATE_CHANGE", "INVOICE", req.InvoiceID,
						`{"old_status":"` + oldStatus.StringVal + `","new_status":"` + req.Status + `","reason":"` + req.Reason + `"}`,
						spanner.CommitTimestamp,
					},
				),
			}
			return txn.BufferWrite(mutations)
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
