package analytics

import (
	"context"
	"net/http"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"

	"backend-go/auth"
	"backend-go/proximity"
)

// ═══════════════════════════════════════════════════════════════════════════════
// FACTORY ANALYTICS — Production throughput + transfer velocity
//
// GET /v1/factory/analytics/overview?from=&to=
// Auth: FACTORY + RequireFactoryScope
// ═══════════════════════════════════════════════════════════════════════════════

type FactoryDayBucket struct {
	Date             string  `json:"date"`
	TransfersCreated int64   `json:"transfers_created"`
	TransfersShipped int64   `json:"transfers_shipped"`
	UnitsProduced    float64 `json:"units_produced"`
}

type FactoryStatusSummary struct {
	State string `json:"state"`
	Count int64  `json:"count"`
}

type FactoryOverviewResponse struct {
	DailyActivity    []FactoryDayBucket     `json:"daily_activity"`
	TransfersByState []FactoryStatusSummary `json:"transfers_by_state"`
	TotalTransfers   int64                  `json:"total_transfers"`
	AvgLeadTimeMins  float64                `json:"avg_lead_time_mins"`
}

func HandleFactoryAnalytics(client *spanner.Client, readRouter proximity.ReadRouter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		scope := auth.GetFactoryScope(r.Context())
		if scope == nil {
			http.Error(w, "Factory scope required", http.StatusForbidden)
			return
		}

		dr := ParseDateRange(r, 30)
		params := map[string]interface{}{
			"factoryId": scope.FactoryID,
			"_from":     dr.From,
			"_to":       dr.To,
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		resp := FactoryOverviewResponse{}

		// 1. Daily activity — transfers created/shipped + units
		dailySql := `
			SELECT
				FORMAT_TIMESTAMP('%Y-%m-%d', t.CreatedAt) as Day,
				COUNT(*) as Created,
				COUNTIF(t.State = 'SHIPPED') as Shipped,
				COALESCE(SUM(t.TotalVU), 0) as Units
			FROM TransferOrders t
			WHERE t.SourceFactoryId = @factoryId
			AND t.CreatedAt >= @_from AND t.CreatedAt <= @_to
			GROUP BY Day
			ORDER BY Day`

		readClient := getReadClient(r.Context(), client, readRouter, nil)
		dailyIter := readClient.Single().Query(ctx, spanner.Statement{SQL: dailySql, Params: params})
		defer dailyIter.Stop()

		for {
			row, err := dailyIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				http.Error(w, "Factory daily activity query fault", http.StatusInternalServerError)
				return
			}
			var b FactoryDayBucket
			var day spanner.NullString
			var created, shipped spanner.NullInt64
			var units spanner.NullFloat64
			if err := row.Columns(&day, &created, &shipped, &units); err != nil {
				continue
			}
			b.Date = day.StringVal
			b.TransfersCreated = created.Int64
			b.TransfersShipped = shipped.Int64
			b.UnitsProduced = units.Float64
			resp.DailyActivity = append(resp.DailyActivity, b)
		}

		// 2. Transfers by state
		stateSql := `
			SELECT t.State, COUNT(*) as Cnt
			FROM TransferOrders t
			WHERE t.SourceFactoryId = @factoryId
			AND t.CreatedAt >= @_from AND t.CreatedAt <= @_to
			GROUP BY t.State
			ORDER BY Cnt DESC`

		stateIter := readClient.Single().Query(ctx, spanner.Statement{SQL: stateSql, Params: params})
		defer stateIter.Stop()

		for {
			row, err := stateIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				break
			}
			var state spanner.NullString
			var count spanner.NullInt64
			if err := row.Columns(&state, &count); err != nil {
				continue
			}
			resp.TransfersByState = append(resp.TransfersByState, FactoryStatusSummary{
				State: state.StringVal,
				Count: count.Int64,
			})
			resp.TotalTransfers += count.Int64
		}

		// 3. Avg lead time
		ltSql := `
			SELECT AVG(TIMESTAMP_DIFF(t.UpdatedAt, t.CreatedAt, MINUTE)) as AvgLead
			FROM TransferOrders t
			WHERE t.SourceFactoryId = @factoryId
			AND t.State IN ('SHIPPED', 'RECEIVED')
			AND t.CreatedAt >= @_from AND t.CreatedAt <= @_to`

		ltIter := readClient.Single().Query(ctx, spanner.Statement{SQL: ltSql, Params: params})
		defer ltIter.Stop()

		if row, err := ltIter.Next(); err == nil {
			var avg spanner.NullFloat64
			if err := row.Columns(&avg); err == nil {
				resp.AvgLeadTimeMins = avg.Float64
			}
		}

		if resp.DailyActivity == nil {
			resp.DailyActivity = []FactoryDayBucket{}
		}
		if resp.TransfersByState == nil {
			resp.TransfersByState = []FactoryStatusSummary{}
		}

		writeJSON(w, resp)
	}
}
