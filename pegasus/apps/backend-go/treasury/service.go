package treasury

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"backend-go/auth"
	"backend-go/finance"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

type TreasuryReport struct {
	PlatformRevenue  int64                 `json:"platform_revenue"`
	SupplierPayout   int64                 `json:"supplier_payout"`
	TotalVolume      int64                 `json:"total_volume"`
	BillingHistory   []BillingHistoryPoint `json:"billing_history,omitempty"`
	BillingMilestone BillingMilestone      `json:"billing_milestone"`
}

type BillingHistoryPoint struct {
	PeriodMonth string `json:"period_month"`
	Currency    string `json:"currency"`
	OrderCount  int64  `json:"order_count"`
	GrossAmount int64  `json:"gross_amount"`
	FeeAmount   int64  `json:"fee_amount"`
}

type BillingMilestone struct {
	CurrentFeeBasisPoints    int64   `json:"current_fee_basis_points"`
	CurrentFeePercent        float64 `json:"current_fee_percent"`
	GlobalOrderCount         int64   `json:"global_order_count"`
	MilestoneOrderCount      int64   `json:"milestone_order_count"`
	CurrentMilestoneIndex    int64   `json:"current_milestone_index"`
	LastMilestoneIndex       int64   `json:"last_milestone_index"`
	NextMilestoneOrderCount  int64   `json:"next_milestone_order_count"`
	OrdersToNextMilestone    int64   `json:"orders_to_next_milestone"`
	MilestoneStepBasisPoints int64   `json:"milestone_step_basis_points"`
	MinFeeBasisPoints        int64   `json:"min_fee_basis_points"`
}

const (
	billingAllTimePeriod                    = "ALL_TIME"
	billingAllTimeCurrency                  = "ALL"
	systemKeyPlatformFeePercent             = "platform_fee_percent"
	systemKeyPlatformFeeBasisPoints         = "platform_fee_basis_points"
	systemKeyMilestoneOrderCount            = "billing_milestone_order_count"
	systemKeyMilestoneStepBasisPoints       = "billing_milestone_step_basis_points"
	systemKeyMinFeeBasisPoints              = "billing_min_fee_basis_points"
	systemKeyLastMilestoneIndex             = "billing_last_milestone_index"
	defaultMilestoneOrderCount        int64 = 100000
	defaultMilestoneStepBP            int64 = 25
	defaultMinFeeBP                   int64 = 25
)

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
			slog.Error("treasury.metrics_row_decode_failed", "err", err)
			continue
		}

		if finance.IsPlatformAccount(accountId) {
			report.PlatformRevenue += amount.Int64
		} else {
			report.SupplierPayout += amount.Int64
		}
	}

	report.TotalVolume = report.PlatformRevenue + report.SupplierPayout

	history, milestone, err := loadBillingTelemetry(ctx, client)
	if err != nil {
		slog.Warn("treasury.billing_telemetry_unavailable", "err", err)
	} else {
		report.BillingHistory = history
		report.BillingMilestone = milestone
	}

	return report, nil
}

func loadBillingTelemetry(ctx context.Context, client *spanner.Client) ([]BillingHistoryPoint, BillingMilestone, error) {
	history, err := loadBillingHistory(ctx, client)
	if err != nil {
		return nil, BillingMilestone{}, err
	}
	milestone, err := loadBillingMilestone(ctx, client)
	if err != nil {
		return nil, BillingMilestone{}, err
	}
	return history, milestone, nil
}

func loadBillingHistory(ctx context.Context, client *spanner.Client) ([]BillingHistoryPoint, error) {
	sinceMonth := time.Now().UTC().AddDate(0, -11, 0).Format("2006-01")
	stmt := spanner.Statement{
		SQL: `SELECT PeriodMonth, Currency,
		             IFNULL(SUM(OrderCount), 0),
		             IFNULL(SUM(GrossAmount), 0),
		             IFNULL(SUM(FeeAmount), 0)
		      FROM BillingGlobalMeters
		      WHERE PeriodMonth != @allTime
		        AND PeriodMonth >= @sinceMonth
		      GROUP BY PeriodMonth, Currency
		      ORDER BY PeriodMonth ASC`,
		Params: map[string]interface{}{
			"allTime":    billingAllTimePeriod,
			"sinceMonth": sinceMonth,
		},
	}

	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	points := make([]BillingHistoryPoint, 0, 12)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var point BillingHistoryPoint
		if err := row.Columns(&point.PeriodMonth, &point.Currency, &point.OrderCount, &point.GrossAmount, &point.FeeAmount); err != nil {
			continue
		}
		points = append(points, point)
	}

	return points, nil
}

func loadBillingMilestone(ctx context.Context, client *spanner.Client) (BillingMilestone, error) {
	globalOrderCount, err := loadGlobalOrderCount(ctx, client)
	if err != nil {
		return BillingMilestone{}, err
	}

	feeBP, err := loadCurrentFeeBasisPoints(ctx, client)
	if err != nil {
		return BillingMilestone{}, err
	}

	milestoneOrders, err := loadSystemConfigInt64(ctx, client, systemKeyMilestoneOrderCount, defaultMilestoneOrderCount)
	if err != nil {
		return BillingMilestone{}, err
	}
	if milestoneOrders <= 0 {
		milestoneOrders = defaultMilestoneOrderCount
	}

	stepBP, err := loadSystemConfigInt64(ctx, client, systemKeyMilestoneStepBasisPoints, defaultMilestoneStepBP)
	if err != nil {
		return BillingMilestone{}, err
	}

	minFeeBP, err := loadSystemConfigInt64(ctx, client, systemKeyMinFeeBasisPoints, defaultMinFeeBP)
	if err != nil {
		return BillingMilestone{}, err
	}

	lastMilestoneIndex, err := loadSystemConfigInt64(ctx, client, systemKeyLastMilestoneIndex, 0)
	if err != nil {
		return BillingMilestone{}, err
	}

	currentMilestoneIndex := int64(0)
	if milestoneOrders > 0 {
		currentMilestoneIndex = globalOrderCount / milestoneOrders
	}
	nextMilestoneOrderCount := (currentMilestoneIndex + 1) * milestoneOrders
	ordersToNext := nextMilestoneOrderCount - globalOrderCount
	if ordersToNext < 0 {
		ordersToNext = 0
	}

	return BillingMilestone{
		CurrentFeeBasisPoints:    feeBP,
		CurrentFeePercent:        float64(feeBP) / 100.0,
		GlobalOrderCount:         globalOrderCount,
		MilestoneOrderCount:      milestoneOrders,
		CurrentMilestoneIndex:    currentMilestoneIndex,
		LastMilestoneIndex:       lastMilestoneIndex,
		NextMilestoneOrderCount:  nextMilestoneOrderCount,
		OrdersToNextMilestone:    ordersToNext,
		MilestoneStepBasisPoints: stepBP,
		MinFeeBasisPoints:        minFeeBP,
	}, nil
}

func loadGlobalOrderCount(ctx context.Context, client *spanner.Client) (int64, error) {
	stmt := spanner.Statement{
		SQL: `SELECT IFNULL(SUM(OrderCount), 0)
		      FROM BillingGlobalMeters
		      WHERE PeriodMonth = @periodMonth AND Currency = @currency`,
		Params: map[string]interface{}{
			"periodMonth": billingAllTimePeriod,
			"currency":    billingAllTimeCurrency,
		},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		if err == iterator.Done {
			return 0, nil
		}
		return 0, err
	}

	var total spanner.NullInt64
	if err := row.Columns(&total); err != nil {
		return 0, err
	}
	if !total.Valid {
		return 0, nil
	}
	return total.Int64, nil
}

func loadCurrentFeeBasisPoints(ctx context.Context, client *spanner.Client) (int64, error) {
	bp, err := loadSystemConfigInt64(ctx, client, systemKeyPlatformFeeBasisPoints, -1)
	if err != nil {
		return 0, err
	}
	if bp >= 0 {
		return bp, nil
	}

	percent, err := loadSystemConfigInt64(ctx, client, systemKeyPlatformFeePercent, 0)
	if err != nil {
		return 0, err
	}
	return percent * 100, nil
}

func loadSystemConfigInt64(ctx context.Context, client *spanner.Client, key string, fallback int64) (int64, error) {
	row, err := client.Single().ReadRow(ctx, "SystemConfig", spanner.Key{key}, []string{"ConfigValue"})
	if err != nil {
		if errors.Is(err, spanner.ErrRowNotFound) {
			return fallback, nil
		}
		return 0, err
	}

	var value string
	if err := row.Columns(&value); err != nil {
		return 0, err
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fallback, nil
	}
	return parsed, nil
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
			slog.Error("treasury.metrics_encode_failed", "err", err)
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
	Currency      string  `json:"currency"`
	CustodyStatus string  `json:"custody_status"`
	CollectedAt   string  `json:"collected_at,omitempty"`
	GeofenceDistM float64 `json:"geofence_dist_m"`
}

type CashHoldingsReport struct {
	TotalPending   int64            `json:"total_pending"`
	TotalCollected int64            `json:"total_collected"`
	PendingCount   int              `json:"pending_count"`
	CollectedCount int              `json:"collected_count"`
	Currency       string           `json:"currency"`
	Holdings       []CashHoldingRow `json:"holdings"`
}

func GetCashHoldings(ctx context.Context, client *spanner.Client, supplierID string) (*CashHoldingsReport, error) {
	report := &CashHoldingsReport{Currency: "UZS", Holdings: []CashHoldingRow{}}

	sql := `SELECT mi.InvoiceId, mi.OrderId, mi.CollectorDriverId, o.RetailerId,
	             mi.Total, mi.CustodyStatus, mi.CollectedAt, mi.GeofenceDistanceM
	      FROM MasterInvoices mi
	      JOIN Orders o ON mi.OrderId = o.OrderId
	      LEFT JOIN Retailers ret ON o.RetailerId = ret.RetailerId
	      WHERE mi.PaymentMode = 'CASH'
	        AND o.SupplierId = @supplierId`
	params := map[string]interface{}{"supplierId": supplierID}
	sql, params = auth.AppendRegionFilter(ctx, sql, params, "ret")
	sql += ` ORDER BY mi.CreatedAt DESC
	      LIMIT 200`

	stmt := spanner.Statement{SQL: sql, Params: params}

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
			slog.Error("treasury.cash_holdings_row_decode_failed", "supplier_id", supplierID, "err", err)
			continue
		}

		h := CashHoldingRow{
			OrderID:       orderID,
			InvoiceID:     invoiceID,
			DriverID:      driverID.StringVal,
			RetailerID:    retailerID.StringVal,
			Amount:        amount.Int64,
			Currency:      report.Currency,
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
			slog.Error("treasury.cash_holdings_encode_failed", "err", err)
		}
	}
}
