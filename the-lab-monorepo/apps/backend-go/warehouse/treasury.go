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
	InvoiceID    string `json:"invoice_id"`
	OrderID      string `json:"order_id"`
	Amount       int64  `json:"amount"`
	Status       string `json:"status"`
	RetailerID   string `json:"retailer_id"`
	RetailerName string `json:"retailer_name,omitempty"`
	CreatedAt    string `json:"created_at"`
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
	stmt := spanner.Statement{
		SQL: `SELECT COALESCE(SUM(mi.TotalAmount), 0),
		             COALESCE(SUM(CASE WHEN mi.Status = 'PENDING' THEN mi.TotalAmount ELSE 0 END), 0),
		             COALESCE(SUM(CASE WHEN mi.Status = 'SETTLED' THEN mi.TotalAmount ELSE 0 END), 0),
		             COUNT(*),
		             COUNTIF(mi.Status = 'PENDING')
		      FROM MasterInvoices mi
		      JOIN Orders o ON mi.OrderId = o.OrderId
		      WHERE o.SupplierId = @sid AND o.WarehouseId = @whId`,
		Params: map[string]interface{}{"sid": ops.SupplierID, "whId": ops.WarehouseID},
	}
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

	stmt := spanner.Statement{
		SQL: `SELECT mi.InvoiceId, mi.OrderId, mi.TotalAmount, mi.Status,
		             COALESCE(o.RetailerId, ''), COALESCE(rt.StoreName, ''),
		             mi.CreatedAt
		      FROM MasterInvoices mi
		      JOIN Orders o ON mi.OrderId = o.OrderId
		      LEFT JOIN Retailers rt ON o.RetailerId = rt.RetailerId
		      WHERE o.SupplierId = @sid AND o.WarehouseId = @whId
		      ORDER BY mi.CreatedAt DESC
		      LIMIT 200`,
		Params: map[string]interface{}{"sid": ops.SupplierID, "whId": ops.WarehouseID},
	}

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
		if err := row.Columns(&si.InvoiceID, &si.OrderID, &si.Amount,
			&si.Status, &si.RetailerID, &si.RetailerName, &createdAt); err != nil {
			log.Printf("[WH TREASURY] parse: %v", err)
			continue
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
