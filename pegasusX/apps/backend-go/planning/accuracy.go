package planning

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ForecastAccuracyEnabled gates the nightly accuracy pass and related API enrichment.
func ForecastAccuracyEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("FORECAST_ACCURACY_ENABLED")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// AccuracyNotifyFunc fans out |TS|>4 alerts (inbox / ops).
type AccuracyNotifyFunc func(ctx context.Context, supplierID, warehouseID, productID string, ts float64, day civil.Date) error

// AccuracyDailyRow is one persisted ForecastAccuracyDaily record.
type AccuracyDailyRow struct {
	SupplierID     string
	ForecastDate   civil.Date
	WarehouseID    string
	ProductID      string
	ForecastQty    int64
	ActualQty      int64
	AbsError       int64
	SignedError    int64
	Wape7          float64
	Wape28         float64
	Bias7          float64
	Bias28         float64
	TrackingSignal float64
	SampleDays7    int64
	SampleDays28   int64
	AlertTs        bool
	ComputedAt     time.Time
}

// SeriesPoint is one day of forecast vs actual for rolling metrics.
type SeriesPoint struct {
	Day         civil.Date
	ForecastQty int64
	ActualQty   int64
}

// SeriesMetrics holds rolling WAPE / bias / tracking signal.
type SeriesMetrics struct {
	Wape7          float64
	Wape28         float64
	Bias7          float64
	Bias28         float64
	TrackingSignal float64
	SampleDays7    int64
	SampleDays28   int64
	AlertTs        bool
}

// ComputeSeriesMetrics scores points ending at asOf (inclusive). Points may be unsorted.
func ComputeSeriesMetrics(points []SeriesPoint, asOf civil.Date) SeriesMetrics {
	byDay := make(map[civil.Date]SeriesPoint, len(points))
	for _, p := range points {
		byDay[p.Day] = p
	}
	window28 := collectWindow(byDay, asOf, 28)
	window7 := collectWindow(byDay, asOf, 7)

	wape7, bias7, n7 := wapeBias(window7)
	wape28, bias28, n28 := wapeBias(window28)
	ts := trackingSignal(window28)

	return SeriesMetrics{
		Wape7:          wape7,
		Wape28:         wape28,
		Bias7:          bias7,
		Bias28:         bias28,
		TrackingSignal: ts,
		SampleDays7:    n7,
		SampleDays28:   n28,
		AlertTs:        math.Abs(ts) > 4,
	}
}

func collectWindow(byDay map[civil.Date]SeriesPoint, asOf civil.Date, days int) []SeriesPoint {
	out := make([]SeriesPoint, 0, days)
	for i := days - 1; i >= 0; i-- {
		d := asOf.AddDays(-i)
		if p, ok := byDay[d]; ok {
			out = append(out, p)
		}
	}
	return out
}

func wapeBias(points []SeriesPoint) (wape, bias float64, sampleDays int64) {
	var sumAbs, sumSigned, sumActual int64
	for _, p := range points {
		if p.ForecastQty == 0 && p.ActualQty == 0 {
			continue
		}
		sampleDays++
		err := p.ForecastQty - p.ActualQty
		if err < 0 {
			sumAbs += -err
		} else {
			sumAbs += err
		}
		sumSigned += err
		sumActual += p.ActualQty
	}
	if sumActual <= 0 {
		return 0, 0, sampleDays
	}
	wape = float64(sumAbs) / float64(sumActual)
	bias = float64(sumSigned) / float64(sumActual)
	return wape, bias, sampleDays
}

// trackingSignal = cumulative signed error / MAD (mean absolute deviation of errors).
func trackingSignal(points []SeriesPoint) float64 {
	var errs []float64
	var cum float64
	for _, p := range points {
		if p.ForecastQty == 0 && p.ActualQty == 0 {
			continue
		}
		e := float64(p.ForecastQty - p.ActualQty)
		errs = append(errs, e)
		cum += e
	}
	if len(errs) == 0 {
		return 0
	}
	var absSum float64
	for _, e := range errs {
		absSum += math.Abs(e)
	}
	mad := absSum / float64(len(errs))
	if mad < 1e-9 {
		if cum == 0 {
			return 0
		}
		// Zero MAD with non-zero cum → treat as large |TS|.
		if cum > 0 {
			return 5
		}
		return -5
	}
	return cum / mad
}

// ConfidencePctFromWape maps measured WAPE28 → ConfidencePct when sample is sufficient.
func ConfidencePctFromWape(wape28 float64, sampleDays28 int64) (pct int64, ok bool) {
	if sampleDays28 < 7 {
		return 0, false
	}
	pct = int64(math.Round(100 * (1 - wape28)))
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	return pct, true
}

// AccuracyService runs the nightly join + persist pass.
type AccuracyService struct {
	Client *spanner.Client
	Log    *slog.Logger
	Notify AccuracyNotifyFunc
	Now    func() time.Time
}

func (s *AccuracyService) now() time.Time {
	if s != nil && s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func (s *AccuracyService) log() *slog.Logger {
	if s != nil && s.Log != nil {
		return s.Log
	}
	return slog.Default()
}

// RunAccuracyPass scores baselines vs actuals for lookbackDays ending today (UTC).
func (s *AccuracyService) RunAccuracyPass(ctx context.Context, supplierID string, lookbackDays int) (written int, alerts int, err error) {
	if s == nil || s.Client == nil {
		return 0, 0, nil
	}
	if !ForecastAccuracyEnabled() {
		s.log().Info("forecast accuracy pass skipped", "reason", "FORECAST_ACCURACY_ENABLED off")
		return 0, 0, nil
	}
	if lookbackDays < 1 {
		lookbackDays = 28
	}
	// Need prior days for rolling windows.
	historyDays := lookbackDays + 27
	end := s.now().Truncate(24 * time.Hour)
	start := end.AddDate(0, 0, -(historyDays - 1))

	actuals, err := LoadCompletedActuals(ctx, s.Client, supplierID, start, end)
	if err != nil {
		return 0, 0, fmt.Errorf("load actuals: %w", err)
	}
	baselines, err := loadBaselinesInRange(ctx, s.Client, supplierID, start, end)
	if err != nil {
		return 0, 0, fmt.Errorf("load baselines: %w", err)
	}

	// Group series history by (supplier, warehouse, product).
	type seriesKey struct {
		SupplierID, WarehouseID, ProductID string
	}
	series := map[seriesKey][]SeriesPoint{}
	seenDays := map[seriesKey]map[civil.Date]struct{}{}

	for _, b := range baselines {
		sk := seriesKey{b.SupplierID, b.WarehouseID, b.ProductID}
		if seenDays[sk] == nil {
			seenDays[sk] = map[civil.Date]struct{}{}
		}
		ak := ActualKey{b.SupplierID, b.WarehouseID, b.ProductID, b.ForecastDate}
		actual := actuals[ak]
		series[sk] = append(series[sk], SeriesPoint{
			Day: b.ForecastDate, ForecastQty: b.BaselineQty, ActualQty: actual,
		})
		seenDays[sk][b.ForecastDate] = struct{}{}
	}
	// Include actual-only days so bias/WAPE see demand with zero forecast.
	for ak, qty := range actuals {
		if supplierID != "" && ak.SupplierID != supplierID {
			continue
		}
		sk := seriesKey{ak.SupplierID, ak.WarehouseID, ak.ProductID}
		if seenDays[sk] != nil {
			if _, ok := seenDays[sk][ak.Day]; ok {
				continue
			}
		}
		series[sk] = append(series[sk], SeriesPoint{Day: ak.Day, ForecastQty: 0, ActualQty: qty})
	}

	scoreStart := end.AddDate(0, 0, -(lookbackDays - 1))
	mutations := make([]*spanner.Mutation, 0, 256)
	for sk, points := range series {
		for d := scoreStart; !d.After(end); d = d.AddDate(0, 0, 1) {
			day := civil.DateOf(d)
			m := ComputeSeriesMetrics(points, day)
			var fQty, aQty int64
			for _, p := range points {
				if p.Day == day {
					fQty, aQty = p.ForecastQty, p.ActualQty
					break
				}
			}
			if fQty == 0 && aQty == 0 && m.SampleDays28 == 0 {
				continue
			}
			absErr := fQty - aQty
			if absErr < 0 {
				absErr = -absErr
			}
			row := map[string]any{
				"SupplierId":     sk.SupplierID,
				"ForecastDate":   day,
				"WarehouseId":    sk.WarehouseID,
				"ProductId":      sk.ProductID,
				"ForecastQty":    fQty,
				"ActualQty":      aQty,
				"AbsError":       absErr,
				"SignedError":    fQty - aQty,
				"Wape7":          m.Wape7,
				"Wape28":         m.Wape28,
				"Bias7":          m.Bias7,
				"Bias28":         m.Bias28,
				"TrackingSignal": m.TrackingSignal,
				"SampleDays7":    m.SampleDays7,
				"SampleDays28":   m.SampleDays28,
				"AlertTs":        m.AlertTs,
				"ComputedAt":     spanner.CommitTimestamp,
			}
			mutations = append(mutations, spanner.InsertOrUpdateMap("ForecastAccuracyDaily", row))
			written++
			if m.AlertTs {
				alerts++
				if s.Notify != nil {
					if nerr := s.Notify(ctx, sk.SupplierID, sk.WarehouseID, sk.ProductID, m.TrackingSignal, day); nerr != nil {
						s.log().Warn("accuracy TS alert notify failed", "err", nerr, "product_id", sk.ProductID)
					}
				} else {
					s.log().Warn("forecast tracking signal out of control",
						"supplier_id", sk.SupplierID, "warehouse_id", sk.WarehouseID,
						"product_id", sk.ProductID, "day", day.String(), "ts", m.TrackingSignal)
				}
			}
			if len(mutations) >= 400 {
				if _, err := s.Client.Apply(ctx, mutations); err != nil {
					return written, alerts, err
				}
				mutations = mutations[:0]
			}
		}
	}
	if len(mutations) > 0 {
		if _, err := s.Client.Apply(ctx, mutations); err != nil {
			return written, alerts, err
		}
	}
	s.log().Info("forecast accuracy pass complete", "written", written, "alerts", alerts)
	return written, alerts, nil
}

type baselineRow struct {
	SupplierID   string
	WarehouseID  string
	ProductID    string
	ForecastDate civil.Date
	BaselineQty  int64
}

func loadBaselinesInRange(ctx context.Context, client *spanner.Client, supplierID string, start, end time.Time) ([]baselineRow, error) {
	sql := `SELECT SupplierId, WarehouseId, ProductId, ForecastDate, BaselineQty
		FROM DemandForecastBaseline
		WHERE ForecastDate BETWEEN @start AND @end`
	params := map[string]any{
		"start": civil.DateOf(start.UTC()),
		"end":   civil.DateOf(end.UTC()),
	}
	if sid := strings.TrimSpace(supplierID); sid != "" {
		sql += ` AND SupplierId = @sid`
		params["sid"] = sid
	}
	iter := client.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	var rows []baselineRow
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var r baselineRow
		if err := row.Columns(&r.SupplierID, &r.WarehouseID, &r.ProductID, &r.ForecastDate, &r.BaselineQty); err != nil {
			return nil, err
		}
		rows = append(rows, r)
	}
	return rows, nil
}

// LoadLatestSeriesWape28 returns average Wape28 / sample for supplier (optional warehouse) on/near day.
func LoadLatestSeriesWape28(ctx context.Context, client *spanner.Client, supplierID, warehouseID string, day time.Time) (wape float64, sampleDays int64, ok bool, err error) {
	if client == nil || strings.TrimSpace(supplierID) == "" {
		return 0, 0, false, nil
	}
	asOf := civil.DateOf(day.UTC())
	sql := `SELECT COALESCE(AVG(Wape28), 0), COALESCE(AVG(SampleDays28), 0), COUNT(*)
		FROM ForecastAccuracyDaily
		WHERE SupplierId = @sid AND ForecastDate = @day`
	params := map[string]any{"sid": supplierID, "day": asOf}
	if wh := strings.TrimSpace(warehouseID); wh != "" {
		sql += ` AND WarehouseId = @wh`
		params["wh"] = wh
	}
	iter := client.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return 0, 0, false, nil
	}
	if err != nil {
		return 0, 0, false, err
	}
	var avgWape, avgSample float64
	var n int64
	if err := row.Columns(&avgWape, &avgSample, &n); err != nil {
		return 0, 0, false, err
	}
	if n == 0 {
		return 0, 0, false, nil
	}
	return avgWape, int64(math.Round(avgSample)), true, nil
}

// ListAccuracyRows returns recent ForecastAccuracyDaily rows for API.
func ListAccuracyRows(ctx context.Context, client *spanner.Client, supplierID, warehouseID, productID string, days int) ([]AccuracyDailyRow, error) {
	if client == nil || supplierID == "" {
		return nil, nil
	}
	if days < 1 {
		days = 28
	}
	end := civil.DateOf(time.Now().UTC())
	start := end.AddDays(-(days - 1))
	sql := `SELECT SupplierId, ForecastDate, WarehouseId, ProductId,
		ForecastQty, ActualQty, AbsError, SignedError,
		COALESCE(Wape7, 0), COALESCE(Wape28, 0), COALESCE(Bias7, 0), COALESCE(Bias28, 0),
		COALESCE(TrackingSignal, 0), SampleDays7, SampleDays28, AlertTs, ComputedAt
		FROM ForecastAccuracyDaily
		WHERE SupplierId = @sid AND ForecastDate BETWEEN @start AND @end`
	params := map[string]any{"sid": supplierID, "start": start, "end": end}
	if wh := strings.TrimSpace(warehouseID); wh != "" {
		sql += ` AND WarehouseId = @wh`
		params["wh"] = wh
	}
	if pid := strings.TrimSpace(productID); pid != "" {
		sql += ` AND ProductId = @pid`
		params["pid"] = pid
	}
	sql += ` ORDER BY ForecastDate DESC, WarehouseId, ProductId LIMIT 2000`
	iter := client.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	var out []AccuracyDailyRow
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var r AccuracyDailyRow
		if err := row.Columns(
			&r.SupplierID, &r.ForecastDate, &r.WarehouseID, &r.ProductID,
			&r.ForecastQty, &r.ActualQty, &r.AbsError, &r.SignedError,
			&r.Wape7, &r.Wape28, &r.Bias7, &r.Bias28, &r.TrackingSignal,
			&r.SampleDays7, &r.SampleDays28, &r.AlertTs, &r.ComputedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}
