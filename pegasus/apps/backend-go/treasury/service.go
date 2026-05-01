package treasury

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"backend-go/auth"
	"backend-go/finance"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

type TreasuryReport struct {
	PlatformRevenue int64 `json:"platform_revenue"`
	SupplierPayout  int64 `json:"supplier_payout"`
	TotalVolume     int64 `json:"total_volume"`
}

// GetTreasuryMetrics runs the aggregation directly on Spanner's distributed nodes
func GetTreasuryMetrics(ctx context.Context, client *spanner.Client) (*TreasuryReport, error) {
	report := &TreasuryReport{}

	stmt := spanner.Statement{
		SQL: `SELECT AccountId, SUM(Amount) 
		      FROM LedgerEntries 
		      GROUP BY AccountId`,
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("ledger aggregation failed: %v", err)
		}

		var accountId string
		var amount spanner.NullInt64
		if err := row.Columns(&accountId, &amount); err != nil {
			log.Printf("[TREASURY ERROR] Failed to decode row: %v", err)
			continue
		}

		if finance.IsPlatformAccount(accountId) {
			report.PlatformRevenue += amount.Int64
		} else {
			report.SupplierPayout += amount.Int64
		}
	}

	report.TotalVolume = report.PlatformRevenue + report.SupplierPayout
	return report, nil
}

// TreasuryHandler exposes the Treasury Report as JSON
func TreasuryHandler(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		report, err := GetTreasuryMetrics(r.Context(), client)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(report); err != nil {
			log.Printf("[TREASURY ERROR] Failed to encode response: %v", err)
		}
	}
}

// ── Cash Holdings ───────────────────────────────────────────────

type CashHoldingRow struct {
	OrderID       string  `json:"order_id"`
	InvoiceID     string  `json:"invoice_id"`
	DriverID      string  `json:"driver_id"`
	RetailerID    string  `json:"retailer_id"`
	Amount        int64   `json:"amount"`
	CustodyStatus string  `json:"custody_status"`
	CollectedAt   string  `json:"collected_at,omitempty"`
	GeofenceDistM float64 `json:"geofence_dist_m"`
}

type CashHoldingsReport struct {
	TotalPending   int64            `json:"total_pending"`
	TotalCollected int64            `json:"total_collected"`
	PendingCount   int              `json:"pending_count"`
	CollectedCount int              `json:"collected_count"`
	Holdings       []CashHoldingRow `json:"holdings"`
}

func GetCashHoldings(ctx context.Context, client *spanner.Client, supplierID string) (*CashHoldingsReport, error) {
	report := &CashHoldingsReport{Holdings: []CashHoldingRow{}}

	stmt := spanner.Statement{
		SQL: `SELECT mi.InvoiceId, mi.OrderId, mi.CollectorDriverId, o.RetailerId,
		             mi.Total, mi.CustodyStatus, mi.CollectedAt, mi.GeofenceDistanceM
		      FROM MasterInvoices mi
		      JOIN Orders o ON mi.OrderId = o.OrderId
		      WHERE mi.PaymentMode = 'CASH'
		        AND o.SupplierId = @supplierId
		      ORDER BY mi.CreatedAt DESC
		      LIMIT 200`,
		Params: map[string]interface{}{"supplierId": supplierID},
	}

	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("cash holdings query failed: %v", err)
		}

		var invoiceID, orderID string
		var driverID, retailerID spanner.NullString
		var amount spanner.NullInt64
		var custodyStatus spanner.NullString
		var collectedAt spanner.NullTime
		var geoDist spanner.NullFloat64

		if err := row.Columns(&invoiceID, &orderID, &driverID, &retailerID,
			&amount, &custodyStatus, &collectedAt, &geoDist); err != nil {
			log.Printf("[CASH_HOLDINGS] row decode error: %v", err)
			continue
		}

		h := CashHoldingRow{
			OrderID:       orderID,
			InvoiceID:     invoiceID,
			DriverID:      driverID.StringVal,
			RetailerID:    retailerID.StringVal,
			Amount:        amount.Int64,
			CustodyStatus: custodyStatus.StringVal,
			GeofenceDistM: geoDist.Float64,
		}
		if collectedAt.Valid {
			h.CollectedAt = collectedAt.Time.Format("2006-01-02T15:04:05Z")
		}

		report.Holdings = append(report.Holdings, h)

		if custodyStatus.StringVal == "PENDING" {
			report.TotalPending += amount.Int64
			report.PendingCount++
		} else {
			report.TotalCollected += amount.Int64
			report.CollectedCount++
		}
	}

	return report, nil
}

// CashHoldingsHandler — GET /v1/treasury/cash-holdings
func CashHoldingsHandler(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		report, err := GetCashHoldings(r.Context(), client, claims.ResolveSupplierID())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(report); err != nil {
			log.Printf("[CASH_HOLDINGS] encode error: %v", err)
		}
	}
}
