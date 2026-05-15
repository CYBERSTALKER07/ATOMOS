package warehouse

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"backend-go/auth"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ─── Treasury ─────────────────────────────────────────────────────────────────
// Warehouse-scoped settlement and ledger view.

type SettlementItem struct {
	InvoiceID        string `json:"invoice_id"`
	OrderID          string `json:"order_id"`
	Amount           int64  `json:"amount"`
	AmountUZS        int64  `json:"amount_uzs"`
	Currency         string `json:"currency"`
	Status           string `json:"status"`
	RetailerID       string `json:"retailer_id"`
	RetailerName     string `json:"retailer_name,omitempty"`
	FeeAmount        int64  `json:"fee_amount,omitempty"`
	NetPayoutAmount  int64  `json:"net_payout_amount,omitempty"`
	PayoutOwnerType  string `json:"payout_owner_type,omitempty"`
	PayoutOwnerID    string `json:"payout_owner_id,omitempty"`
	FeePolicyVersion string `json:"fee_policy_version,omitempty"`
	SettlementTarget string `json:"settlement_target,omitempty"`
	CreatedAt        string `json:"created_at"`
}

type TreasuryOverview struct {
	TotalCollected int64 `json:"total_collected"`
	TotalPending   int64 `json:"total_pending"`
	TotalSettled   int64 `json:"total_settled"`
	InvoiceCount   int64 `json:"invoice_count"`
	PendingCount   int64 `json:"pending_count"`
}

// HandleOpsTreasury — GET for /v1/warehouse/ops/treasury
func HandleOpsTreasury(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		ops := auth.GetWarehouseOps(r.Context())
		if ops == nil {
			http.Error(w, "Warehouse scope required", http.StatusForbidden)
			return
		}

		view := r.URL.Query().Get("view") // "overview" or "invoices"
		if view == "overview" || view == "" {
			handleTreasuryOverview(w, r, spannerClient, ops)
			return
		}
		handleTreasuryInvoices(w, r, spannerClient, ops)
	}
}

func handleTreasuryOverview(w http.ResponseWriter, r *http.Request, client *spanner.Client, ops *auth.WarehouseOps) {
	ctx := r.Context()
	overview := TreasuryOverview{}

	// Aggregate from MasterInvoices joined with Orders
	sql := `SELECT COALESCE(SUM(mi.TotalAmount), 0),
	             COALESCE(SUM(CASE WHEN mi.Status = 'PENDING' THEN mi.TotalAmount ELSE 0 END), 0),
	             COALESCE(SUM(CASE WHEN mi.Status = 'SETTLED' THEN mi.TotalAmount ELSE 0 END), 0),
	             COUNT(*),
	             COUNTIF(mi.Status = 'PENDING')
	      FROM MasterInvoices mi
	      JOIN Orders o ON mi.OrderId = o.OrderId
	      LEFT JOIN Retailers rt ON o.RetailerId = rt.RetailerId
	      WHERE o.SupplierId = @sid AND o.WarehouseId = @whId`
	params := map[string]interface{}{"sid": ops.SupplierID, "whId": ops.WarehouseID}
	sql, params = auth.AppendRegionFilter(ctx, sql, params, "rt")
	stmt := spanner.Statement{SQL: sql, Params: params}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()
	if row, err := iter.Next(); err == nil {
		row.Columns(&overview.TotalCollected, &overview.TotalPending,
			&overview.TotalSettled, &overview.InvoiceCount, &overview.PendingCount)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(overview)
}

func handleTreasuryInvoices(w http.ResponseWriter, r *http.Request, client *spanner.Client, ops *auth.WarehouseOps) {
	ctx := r.Context()

	sql := `SELECT mi.InvoiceId, mi.OrderId, mi.TotalAmount, mi.Status,
	             COALESCE(o.RetailerId, ''), COALESCE(rt.StoreName, ''),
	             COALESCE(o.Currency, 'UZS'),
	             COALESCE(iss.FeeAmount, 0),
	             COALESCE(iss.NetPayoutAmount, 0),
	             COALESCE(iss.PayoutOwnerType, ''),
	             COALESCE(iss.PayoutOwnerId, ''),
	             COALESCE(iss.FeePolicyVersion, ''),
	             COALESCE(mi.SettlementTarget, ''),
	             mi.CreatedAt
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
	      LEFT JOIN Retailers rt ON o.RetailerId = rt.RetailerId
	      WHERE o.SupplierId = @sid AND o.WarehouseId = @whId`
	params := map[string]interface{}{"sid": ops.SupplierID, "whId": ops.WarehouseID}
	sql, params = auth.AppendRegionFilter(ctx, sql, params, "rt")
	sql += ` ORDER BY mi.CreatedAt DESC
	      LIMIT 200`

	stmt := spanner.Statement{SQL: sql, Params: params}

	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var invoices []SettlementItem
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("[WH TREASURY] list error: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		var si SettlementItem
		var createdAt time.Time
		var currency spanner.NullString
		var feeAmount spanner.NullInt64
		var netPayoutAmount spanner.NullInt64
		var payoutOwnerType spanner.NullString
		var payoutOwnerID spanner.NullString
		var feePolicyVersion spanner.NullString
		var settlementTarget spanner.NullString
		if err := row.Columns(&si.InvoiceID, &si.OrderID, &si.Amount,
			&si.Status, &si.RetailerID, &si.RetailerName, &currency,
			&feeAmount, &netPayoutAmount, &payoutOwnerType, &payoutOwnerID,
			&feePolicyVersion, &settlementTarget, &createdAt); err != nil {
			log.Printf("[WH TREASURY] parse: %v", err)
			continue
		}
		si.AmountUZS = si.Amount
		si.Currency = "UZS"
		si.FeeAmount = feeAmount.Int64
		si.NetPayoutAmount = netPayoutAmount.Int64
		si.PayoutOwnerType = payoutOwnerType.StringVal
		si.PayoutOwnerID = payoutOwnerID.StringVal
		si.FeePolicyVersion = feePolicyVersion.StringVal
		si.SettlementTarget = settlementTarget.StringVal
		if currency.Valid && currency.StringVal != "" {
			si.Currency = currency.StringVal
		}
		si.CreatedAt = createdAt.Format(time.RFC3339)
		invoices = append(invoices, si)
	}
	if invoices == nil {
		invoices = []SettlementItem{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"invoices": invoices, "total": len(invoices)})
}
