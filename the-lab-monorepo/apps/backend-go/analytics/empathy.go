package analytics

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"backend-go/auth"
	"backend-go/proximity"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// EmpathyAdoption represents the auto-order adoption metrics for admin dashboard.
type EmpathyAdoption struct {
	TotalRetailers      int64 `json:"total_retailers"`
	GlobalEnabled       int64 `json:"global_enabled"`
	SupplierOverrides   int64 `json:"supplier_overrides"`
	ProductOverrides    int64 `json:"product_overrides"`
	VariantOverrides    int64 `json:"variant_overrides"`
	PredictionsDormant  int64 `json:"predictions_dormant"`
	PredictionsWaiting  int64 `json:"predictions_waiting"`
	PredictionsFired    int64 `json:"predictions_fired"`
	PredictionsRejected int64 `json:"predictions_rejected"`
}

// HandleEmpathyAdoption returns adoption metrics for the Empathy Engine (GLOBAL_ADMIN only).
func HandleEmpathyAdoption(client *spanner.Client, readRouter proximity.ReadRouter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// SOVEREIGN ACTION: System-wide analytics requires GLOBAL_ADMIN
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
		if !ok || claims.UserID == "" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		if err := auth.RequireGlobalAdmin(w, claims); err != nil {
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		var adoption EmpathyAdoption

		readClient := getReadClient(r.Context(), client, readRouter, nil)

		// Total retailers
		countQuery := func(sql string) int64 {
			iter := readClient.Single().Query(ctx, spanner.Statement{SQL: sql})
			defer iter.Stop()
			row, err := iter.Next()
			if err != nil {
				return 0
			}
			var cnt int64
			if err := row.Columns(&cnt); err != nil {
				return 0
			}
			return cnt
		}

		adoption.TotalRetailers = countQuery("SELECT COUNT(*) FROM RetailerGlobalSettings")
		adoption.GlobalEnabled = countQuery("SELECT COUNT(*) FROM RetailerGlobalSettings WHERE GlobalAutoOrderEnabled = TRUE")
		adoption.SupplierOverrides = countQuery("SELECT COUNT(*) FROM RetailerSupplierSettings WHERE AutoOrderEnabled = TRUE")
		adoption.ProductOverrides = countQuery("SELECT COUNT(*) FROM RetailerProductSettings WHERE AutoOrderEnabled = TRUE")
		adoption.VariantOverrides = countQuery("SELECT COUNT(*) FROM RetailerVariantSettings WHERE AutoOrderEnabled = TRUE")

		// Prediction status counts
		statusQuery := `SELECT IFNULL(Status, 'UNKNOWN') as s, COUNT(*) as c FROM AIPredictions GROUP BY s`
		iter := readClient.Single().Query(ctx, spanner.Statement{SQL: statusQuery})
		defer iter.Stop()
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				break
			}
			var status string
			var cnt int64
			if err := row.Columns(&status, &cnt); err != nil {
				continue
			}
			switch status {
			case "DORMANT":
				adoption.PredictionsDormant = cnt
			case "WAITING":
				adoption.PredictionsWaiting = cnt
			case "FIRED":
				adoption.PredictionsFired = cnt
			case "REJECTED":
				adoption.PredictionsRejected = cnt
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(adoption)
	}
}
