package supplier

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"backend-go/auth"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

type onboardingStep struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Complete bool   `json:"complete"`
}

type onboardingProgressResponse struct {
	SupplierID      string           `json:"supplier_id"`
	Complete        bool             `json:"complete"`
	CompletedSteps  int              `json:"completed_steps"`
	TotalSteps      int              `json:"total_steps"`
	Steps           []onboardingStep `json:"steps"`
	CheckedAt       string           `json:"checked_at"`
}

// HandleOnboardingProgress serves GET /v1/supplier/onboarding-progress.
func HandleOnboardingProgress(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims.UserID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		supplierID := claims.ResolveSupplierID()

		profile, warehouseCount, gatewayCount, err := loadOnboardingSignals(r.Context(), client, supplierID)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		steps := []onboardingStep{
			{Key: "tax_profile", Label: "Tax profile", Complete: profile.TaxID != ""},
			{Key: "categories", Label: "Operating categories", Complete: len(profile.Categories) > 0},
			{Key: "billing", Label: "Billing setup", Complete: profile.BankName != "" && profile.PaymentGateway != ""},
			{Key: "warehouse", Label: "Warehouse configured", Complete: warehouseCount > 0},
			{Key: "payment_gateway", Label: "Payment gateway connected", Complete: gatewayCount > 0 || profile.PaymentGateway != ""},
		}

		completed := 0
		for _, step := range steps {
			if step.Complete {
				completed++
			}
		}

		resp := onboardingProgressResponse{
			SupplierID:     supplierID,
			Complete:       profile.IsConfigured && completed == len(steps),
			CompletedSteps: completed,
			TotalSteps:     len(steps),
			Steps:          steps,
			CheckedAt:      time.Now().UTC().Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

type onboardingProfile struct {
	TaxID          string
	BankName       string
	PaymentGateway string
	IsConfigured   bool
	Categories     []string
}

func loadOnboardingSignals(ctx context.Context, client *spanner.Client, supplierID string) (onboardingProfile, int64, int64, error) {
	var profile onboardingProfile
	stmt := spanner.Statement{
		SQL: `SELECT COALESCE(TaxId, ''), COALESCE(BankName, ''), COALESCE(PaymentGateway, ''),
		             IFNULL(IsConfigured, false), COALESCE(OperatingCategories, [])
		      FROM Suppliers WHERE SupplierId = @id`,
		Params: map[string]interface{}{"id": supplierID},
	}
	row, err := client.Single().Query(ctx, stmt).Next()
	if err != nil {
		return profile, 0, 0, err
	}
	if err := row.Columns(&profile.TaxID, &profile.BankName, &profile.PaymentGateway, &profile.IsConfigured, &profile.Categories); err != nil {
		return profile, 0, 0, err
	}

	warehouseCount, err := countRows(ctx, client,
		`SELECT COUNT(*) FROM Warehouses WHERE SupplierId = @id AND IFNULL(IsActive, true)`,
		map[string]interface{}{"id": supplierID})
	if err != nil {
		return profile, 0, 0, err
	}
	gatewayCount, err := countRows(ctx, client,
		`SELECT COUNT(*) FROM GatewayOnboardingSessions WHERE SupplierId = @id AND Status = 'COMPLETED'`,
		map[string]interface{}{"id": supplierID})
	if err != nil {
		return profile, warehouseCount, 0, err
	}
	return profile, warehouseCount, gatewayCount, nil
}

func countRows(ctx context.Context, client *spanner.Client, sql string, params map[string]interface{}) (int64, error) {
	iter := client.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return 0, err
	}
	var count int64
	if err := row.Columns(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// DriverScorecard is the MVP supplier-facing driver performance snapshot.
type DriverScorecard struct {
	DriverID        string  `json:"driver_id"`
	Deliveries30d   int64   `json:"deliveries_30d"`
	OnTimeRate      float64 `json:"on_time_rate"`
	AvgRating       float64 `json:"avg_rating"`
	IncidentCount   int64   `json:"incident_count"`
	GeneratedAt     string  `json:"generated_at"`
}

// HandleDriverScorecard serves GET /v1/supplier/drivers/{id}/scorecard.
func HandleDriverScorecard(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims.UserID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		supplierID := claims.ResolveSupplierID()
		driverID := driverIDFromScorecardPath(r.URL.Path)
		if driverID == "" {
			http.Error(w, `{"error":"driver id required"}`, http.StatusBadRequest)
			return
		}

		scorecard, err := buildDriverScorecard(r.Context(), client, supplierID, driverID)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(scorecard)
	}
}

func driverIDFromScorecardPath(path string) string {
	const prefix = "/v1/supplier/drivers/"
	if len(path) <= len(prefix) {
		return ""
	}
	rest := path[len(prefix):]
	for i := 0; i < len(rest); i++ {
		if rest[i] == '/' {
			if rest[i+1:] == "scorecard" {
				return rest[:i]
			}
			return ""
		}
	}
	return ""
}

func buildDriverScorecard(ctx context.Context, client *spanner.Client, supplierID, driverID string) (DriverScorecard, error) {
	stmt := spanner.Statement{
		SQL: `SELECT
			COUNTIF(o.State = 'COMPLETED') AS Deliveries,
			SAFE_DIVIDE(
				COUNTIF(o.State = 'COMPLETED' AND TIMESTAMP_DIFF(o.UpdatedAt, o.CreatedAt, HOUR) <= 24),
				NULLIF(COUNTIF(o.State = 'COMPLETED'), 0)
			) AS OnTimeRate
		FROM Orders o
		WHERE o.SupplierId = @supplierID
		  AND o.DriverId = @driverID
		  AND o.CreatedAt >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 30 DAY)`,
		Params: map[string]interface{}{
			"supplierID": supplierID,
			"driverID":   driverID,
		},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return DriverScorecard{
			DriverID:      driverID,
			OnTimeRate:    0,
			GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		}, nil
	}
	if err != nil {
		return DriverScorecard{}, err
	}
	var deliveries int64
	var onTime spanner.NullFloat64
	if err := row.Columns(&deliveries, &onTime); err != nil {
		return DriverScorecard{}, err
	}
	rate := 0.0
	if onTime.Valid {
		rate = onTime.Float64
	}
	return DriverScorecard{
		DriverID:      driverID,
		Deliveries30d: deliveries,
		OnTimeRate:    rate,
		AvgRating:     0,
		IncidentCount: 0,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// HandleMonthlySummaryReport serves GET /v1/supplier/reports/monthly-summary?format=csv
func HandleMonthlySummaryReport(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Query().Get("format") != "csv" {
			http.Error(w, `{"error":"only format=csv is supported in MVP"}`, http.StatusBadRequest)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims.UserID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		supplierID := claims.ResolveSupplierID()

		month := time.Now().UTC().Format("2006-01")
		orders, revenue, err := monthlyOrderStats(r.Context(), client, supplierID)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", `attachment; filename="monthly-summary-`+month+`.csv"`)
		_, _ = w.Write([]byte("month,supplier_id,orders_completed,revenue_uzs\n"))
		_, _ = w.Write([]byte(month + "," + supplierID + "," + strconv.FormatInt(orders, 10) + "," + strconv.FormatInt(revenue, 10) + "\n"))
	}
}

func monthlyOrderStats(ctx context.Context, client *spanner.Client, supplierID string) (int64, int64, error) {
	stmt := spanner.Statement{
		SQL: `SELECT COUNT(*), IFNULL(SUM(TotalAmount), 0)
		      FROM Orders
		      WHERE SupplierId = @supplierID
		        AND State = 'COMPLETED'
		        AND CreatedAt >= TIMESTAMP_TRUNC(CURRENT_TIMESTAMP(), MONTH)`,
		Params: map[string]interface{}{"supplierID": supplierID},
	}
	row, err := client.Single().Query(ctx, stmt).Next()
	if err != nil {
		return 0, 0, err
	}
	var orders, revenue int64
	if err := row.Columns(&orders, &revenue); err != nil {
		return 0, 0, err
	}
	return orders, revenue, nil
}
