package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ReconciliationRecord exactly matches your Next.js TypeScript interface
type ReconciliationRecord struct {
	OrderID         string `json:"order_id"`
	RetailerID      string `json:"retailer_id"`
	SpannerAmount   int64  `json:"spanner_amount"`
	GatewayAmount   int64  `json:"gateway_amount"`
	GatewayProvider string `json:"gateway_provider"`
	Status          string `json:"status"`
	Timestamp       string `json:"timestamp"`
}

// HandleGetReconciliation feeds the Admin Omniscience Dashboard
func HandleGetReconciliation(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// In production, verify the pegasus_admin_jwt here

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		limit := int64(100)
		if raw := r.URL.Query().Get("limit"); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil {
				http.Error(w, "Invalid limit", http.StatusBadRequest)
				return
			}
			if parsed <= 0 {
				parsed = 100
			}
			if parsed > 500 {
				parsed = 500
			}
			limit = int64(parsed)
		}

		offset := int64(0)
		if raw := r.URL.Query().Get("offset"); raw != "" {
			parsed, err := strconv.ParseInt(raw, 10, 64)
			if err != nil {
				http.Error(w, "Invalid offset", http.StatusBadRequest)
				return
			}
			if parsed < 0 {
				parsed = 0
			}
			offset = parsed
		}

		// Fetch all unresolved anomalies
		stmt := spanner.Statement{
			SQL: `SELECT OrderId, RetailerId, SpannerAmount, GatewayAmount, GatewayProvider, Status, DetectedAt 
			      FROM LedgerAnomalies 
			      WHERE Status != 'MATCH' 
			      ORDER BY DetectedAt DESC LIMIT @limit OFFSET @offset`,
			Params: map[string]interface{}{
				"limit":  limit,
				"offset": offset,
			},
		}

		iter := client.Single().Query(ctx, stmt)
		defer iter.Stop()

		var anomalies []ReconciliationRecord
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				http.Error(w, "Ledger query fault", http.StatusInternalServerError)
				return
			}

			var rec ReconciliationRecord
			var detectedAt time.Time

			if err := row.Columns(&rec.OrderID, &rec.RetailerID, &rec.SpannerAmount, &rec.GatewayAmount, &rec.GatewayProvider, &rec.Status, &detectedAt); err != nil {
				http.Error(w, "Data extraction fault", http.StatusInternalServerError)
				return
			}
			rec.Timestamp = detectedAt.Format(time.RFC3339)
			anomalies = append(anomalies, rec)
		}

		// If nil, return an empty array to prevent Next.js UI crashes
		if anomalies == nil {
			anomalies = []ReconciliationRecord{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "SUCCESS",
			"data":   anomalies,
		})
	}
}
