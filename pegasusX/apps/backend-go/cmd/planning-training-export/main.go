package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
	"github.com/pegasusx/pegasusx/apps/backend-go/planning"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// PX-PROD-3: daily export of forecast baseline vs completed order actuals (collect-only; no training).
func main() {
	supplierID := flag.String("supplier-id", "", "filter by supplier_id (optional)")
	days := flag.Int("days", 30, "lookback window in days")
	format := flag.String("format", "jsonl", "jsonl or csv")
	outPath := flag.String("out", "", "output file (default stdout)")
	minRows := flag.Int("min-rows", 0, "fail if exported row count is below this threshold")
	timeout := flag.Duration("timeout", 15*time.Minute, "job timeout")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	cfg, err := bootstrap.LoadConfig()
	if err != nil {
		slog.Error("load config", "err", err)
		os.Exit(1)
	}
	client, err := spanner.NewClient(ctx, spannerDatabasePath(cfg), spannerClientOptions(cfg)...)
	if err != nil {
		slog.Error("spanner client", "err", err)
		os.Exit(1)
	}
	defer client.Close()

	end := time.Now().UTC().Truncate(24 * time.Hour)
	start := end.AddDate(0, 0, -(*days)+1)

	rows, err := queryTrainingRows(ctx, client, strings.TrimSpace(*supplierID), start, end)
	if err != nil {
		slog.Error("export query failed", "err", err)
		os.Exit(1)
	}
	rows, quality := sanitizeTrainingRows(rows)
	if *minRows > 0 && len(rows) < *minRows {
		slog.Error("export below min-rows threshold",
			"rows", len(rows),
			"min_rows", *minRows,
			"quality", quality,
		)
		os.Exit(1)
	}
	if quality.MLSourceRows > 0 {
		slog.Warn("export contained legacy ml baseline_source rows (normalized on write)", "count", quality.MLSourceRows)
	}

	var out *os.File = os.Stdout
	if p := strings.TrimSpace(*outPath); p != "" {
		f, err := os.Create(p)
		if err != nil {
			slog.Error("create output", "path", p, "err", err)
			os.Exit(1)
		}
		defer f.Close()
		out = f
	}

	switch strings.ToLower(strings.TrimSpace(*format)) {
	case "csv":
		err = writeCSV(out, rows)
	default:
		err = writeJSONL(out, rows)
	}
	if err != nil {
		slog.Error("write export", "err", err)
		os.Exit(1)
	}

	slog.Info("planning training export complete",
		"rows", len(rows),
		"start", start.Format("2006-01-02"),
		"end", end.Format("2006-01-02"),
		"format", *format,
		"null_baseline_qty", quality.NullBaselineQty,
		"ml_source_rows_normalized", quality.MLSourceRows,
	)
}

type exportQuality struct {
	NullBaselineQty int
	MLSourceRows    int
}

func sanitizeTrainingRows(rows []trainingRow) ([]trainingRow, exportQuality) {
	var q exportQuality
	for i := range rows {
		if rows[i].BaselineQty <= 0 {
			q.NullBaselineQty++
		}
		if strings.EqualFold(strings.TrimSpace(rows[i].BaselineSource), "ml") {
			q.MLSourceRows++
		}
		rows[i].BaselineSource = planning.NormalizeBaselineSource(rows[i].BaselineSource)
	}
	return rows, q
}

type trainingRow struct {
	SupplierID           string `json:"supplier_id"`
	WarehouseID          string `json:"warehouse_id"`
	ProductID            string `json:"product_id"`
	ForecastDate         string `json:"forecast_date"`
	BaselineQty          int64  `json:"baseline_qty"`
	LowUnits             int64  `json:"low_units"`
	HighUnits            int64  `json:"high_units"`
	ConfidencePct        int64  `json:"confidence_pct"`
	BaselineSource       string `json:"baseline_source"`
	WarehouseDayOrders   int64  `json:"warehouse_day_completed_orders"`
	ExportAt             string `json:"export_at"`
}

func queryTrainingRows(ctx context.Context, client *spanner.Client, supplierID string, start, end time.Time) ([]trainingRow, error) {
	sql := `SELECT
		b.SupplierId, b.WarehouseId, b.ProductId, b.ForecastDate,
		b.BaselineQty, COALESCE(b.LowUnits, b.BaselineQty), COALESCE(b.HighUnits, b.BaselineQty),
		COALESCE(b.ConfidencePct, CAST(ROUND(b.Confidence * 100) AS INT64)),
		COALESCE(b.BaselineSource, b.Source),
		COALESCE(act.CompletedOrders, 0)
	FROM DemandForecastBaseline b
	LEFT JOIN (
		SELECT SupplierId, WarehouseId,
			DATE(TIMESTAMP_TRUNC(UpdatedAt, DAY, 'UTC')) AS OrderDay,
			COUNT(*) AS CompletedOrders
		FROM Orders
		WHERE Status = 'COMPLETED'
		GROUP BY SupplierId, WarehouseId, OrderDay
	) act ON b.SupplierId = act.SupplierId
		AND b.WarehouseId = act.WarehouseId
		AND b.ForecastDate = act.OrderDay
	WHERE b.ForecastDate BETWEEN @start AND @end`
	params := map[string]any{"start": start, "end": end}
	if supplierID != "" {
		sql += ` AND b.SupplierId = @sid`
		params["sid"] = supplierID
	}
	sql += ` ORDER BY b.ForecastDate DESC, b.SupplierId, b.WarehouseId, b.ProductId`

	iter := client.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()

	exportAt := time.Now().UTC().Format(time.RFC3339Nano)
	rows := make([]trainingRow, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var r trainingRow
		var forecastDate time.Time
		if err := row.Columns(
			&r.SupplierID, &r.WarehouseID, &r.ProductID, &forecastDate,
			&r.BaselineQty, &r.LowUnits, &r.HighUnits, &r.ConfidencePct, &r.BaselineSource,
			&r.WarehouseDayOrders,
		); err != nil {
			return nil, err
		}
		r.ForecastDate = forecastDate.Format("2006-01-02")
		r.ExportAt = exportAt
		rows = append(rows, r)
	}
	return rows, nil
}

func writeJSONL(w *os.File, rows []trainingRow) error {
	enc := json.NewEncoder(w)
	for _, row := range rows {
		if err := enc.Encode(row); err != nil {
			return err
		}
	}
	return nil
}

func writeCSV(w *os.File, rows []trainingRow) error {
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{
		"supplier_id", "warehouse_id", "product_id", "forecast_date",
		"baseline_qty", "low_units", "high_units", "confidence_pct", "baseline_source",
		"warehouse_day_completed_orders", "export_at",
	})
	for _, r := range rows {
		if err := cw.Write([]string{
			r.SupplierID, r.WarehouseID, r.ProductID, r.ForecastDate,
			fmt.Sprintf("%d", r.BaselineQty), fmt.Sprintf("%d", r.LowUnits), fmt.Sprintf("%d", r.HighUnits),
			fmt.Sprintf("%d", r.ConfidencePct), r.BaselineSource,
			fmt.Sprintf("%d", r.WarehouseDayOrders), r.ExportAt,
		}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

func spannerClientOptions(cfg *bootstrap.Config) []option.ClientOption {
	if strings.TrimSpace(cfg.SpannerEmulatorHost) == "" {
		return nil
	}
	return []option.ClientOption{
		option.WithEndpoint(cfg.SpannerEmulatorHost),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		option.WithoutAuthentication(),
	}
}

func spannerDatabasePath(cfg *bootstrap.Config) string {
	return fmt.Sprintf("projects/%s/instances/%s/databases/%s",
		cfg.SpannerProject, cfg.SpannerInstance, cfg.SpannerDatabase)
}
