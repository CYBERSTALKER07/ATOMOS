package planning

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/seasonalcore"
	"google.golang.org/api/iterator"
)

// SeasonalEstimateSuggestion is a history-seeded calendar multiplier draft.
type SeasonalEstimateSuggestion struct {
	TemplateID      string  `json:"template_id"`
	Name            string  `json:"name"`
	StartDate       string  `json:"start_date"`
	EndDate         string  `json:"end_date"`
	Multiplier      float64 `json:"multiplier"`
	Basis           string  `json:"basis"`
	SampleDays      int     `json:"sample_days,omitempty"`
	DraftOverrideID string  `json:"draft_override_id,omitempty"`
}

// SeasonalEstimateResult is the estimate API response.
type SeasonalEstimateResult struct {
	Suggestions     []SeasonalEstimateSuggestion `json:"suggestions"`
	PersistedDrafts int                          `json:"persisted_drafts"`
}

// SeasonalEstimateEnabled gates history-seeded calendar multiplier jobs/API.
func SeasonalEstimateEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("FORECAST_SEASONAL_ESTIMATE_ENABLED")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// EstimateCalendarMultipliers computes YoY ratios for builtin windows and month
// buckets from COMPLETED order actuals. When persistDrafts is true, upserts
// inactive draft overrides (template_id estimate:<id>) for supplier review —
// never auto-activates.
func (s *Service) EstimateCalendarMultipliers(
	ctx context.Context,
	supplierID string,
	asOf time.Time,
	persistDrafts bool,
) (SeasonalEstimateResult, error) {
	out := SeasonalEstimateResult{Suggestions: []SeasonalEstimateSuggestion{}}
	if s == nil || s.Spanner == nil {
		return out, errors.New("planning unavailable")
	}
	client := s.Spanner
	sid := strings.TrimSpace(supplierID)
	if sid == "" {
		return out, errors.New("supplier_id_required")
	}
	if asOf.IsZero() {
		asOf = time.Now().UTC()
	}
	asOf = asOf.UTC().Truncate(24 * time.Hour)
	year := asOf.Year()

	// Load ~2 years of actuals for YoY window + month ratios.
	histStart := asOf.AddDate(-2, 0, 0)
	actuals, err := LoadCompletedActuals(ctx, client, sid, histStart, asOf)
	if err != nil {
		return out, err
	}
	dayTotals := aggregateActualsByDay(actuals)

	for _, tpl := range seasonalcore.Builtins {
		sug := estimateBuiltinYoY(tpl, year, dayTotals)
		if sug.Multiplier <= 0 {
			continue
		}
		out.Suggestions = append(out.Suggestions, sug)
	}
	out.Suggestions = append(out.Suggestions, estimateMonthBuckets(year, dayTotals)...)

	if !persistDrafts || len(out.Suggestions) == 0 {
		return out, nil
	}
	persisted, err := s.persistEstimateDrafts(ctx, sid, out.Suggestions)
	if err != nil {
		return out, err
	}
	out.PersistedDrafts = persisted
	return out, nil
}

func aggregateActualsByDay(actuals ActualQtyMap) map[civil.Date]int64 {
	out := make(map[civil.Date]int64)
	for k, qty := range actuals {
		out[k.Day] += qty
	}
	return out
}

func sumWindow(dayTotals map[civil.Date]int64, start, end time.Time) (total int64, days int) {
	for d := start; !d.After(end); d = d.Add(24 * time.Hour) {
		cd := civil.DateOf(d)
		if qty, ok := dayTotals[cd]; ok {
			total += qty
			days++
		}
	}
	return total, days
}

func estimateBuiltinYoY(tpl seasonalcore.Template, year int, dayTotals map[civil.Date]int64) SeasonalEstimateSuggestion {
	curStart, curEnd := seasonalcore.WindowBounds(tpl, year)
	prevStart, prevEnd := seasonalcore.WindowBounds(tpl, year-1)
	curTotal, curDays := sumWindow(dayTotals, curStart, curEnd)
	prevTotal, prevDays := sumWindow(dayTotals, prevStart, prevEnd)
	mult := seasonalcore.DefaultOverrideMultiplier
	basis := "insufficient_history"
	sample := curDays + prevDays
	if prevTotal > 0 && curDays > 0 && prevDays > 0 {
		ratio := float64(curTotal) / float64(prevTotal)
		// Normalize by days present when coverage differs.
		if prevDays != curDays && prevDays > 0 && curDays > 0 {
			ratio = (float64(curTotal) / float64(curDays)) / (float64(prevTotal) / float64(prevDays))
		}
		mult = seasonalcore.ClampMultiplier(ratio)
		basis = "yoy_window"
	} else if tpl.Multiplier > 0 {
		mult = tpl.Multiplier
		basis = "builtin_fallback"
	}
	return SeasonalEstimateSuggestion{
		TemplateID: "estimate:" + tpl.ID,
		Name:       tpl.Name + " (estimated)",
		StartDate:  curStart.Format("2006-01-02"),
		EndDate:    curEnd.Format("2006-01-02"),
		Multiplier: mult,
		Basis:      basis,
		SampleDays: sample,
	}
}

func estimateMonthBuckets(year int, dayTotals map[civil.Date]int64) []SeasonalEstimateSuggestion {
	var offSum float64
	var offDays int
	monthSum := make([]float64, 13)
	monthDays := make([]int, 13)
	for d, qty := range dayTotals {
		if d.Year != year && d.Year != year-1 {
			continue
		}
		m := int(d.Month)
		monthSum[m] += float64(qty)
		monthDays[m]++
		// Off-window mean: months outside Jun–Aug and Nov–Jan.
		if (m >= 2 && m <= 5) || (m >= 9 && m <= 10) {
			offSum += float64(qty)
			offDays++
		}
	}
	offMean := 0.0
	if offDays > 0 {
		offMean = offSum / float64(offDays)
	}
	var out []SeasonalEstimateSuggestion
	for m := 1; m <= 12; m++ {
		if monthDays[m] < 7 || offMean <= 0 {
			continue
		}
		mean := monthSum[m] / float64(monthDays[m])
		mult := seasonalcore.ClampMultiplier(mean / offMean)
		if mult >= 0.95 && mult <= 1.05 {
			continue // skip near-neutral months
		}
		start := time.Date(year, time.Month(m), 1, 0, 0, 0, 0, time.UTC)
		end := start.AddDate(0, 1, -1)
		out = append(out, SeasonalEstimateSuggestion{
			TemplateID: fmt.Sprintf("estimate:month_%02d", m),
			Name:       fmt.Sprintf("Month %02d (estimated)", m),
			StartDate:  start.Format("2006-01-02"),
			EndDate:    end.Format("2006-01-02"),
			Multiplier: mult,
			Basis:      "month_vs_offwindow",
			SampleDays: monthDays[m],
		})
	}
	return out
}

func (s *Service) persistEstimateDrafts(ctx context.Context, supplierID string, suggestions []SeasonalEstimateSuggestion) (int, error) {
	if s == nil || s.Spanner == nil {
		return 0, errors.New("planning unavailable")
	}
	persisted := 0
	for i := range suggestions {
		sug := &suggestions[i]
		start, err := time.Parse("2006-01-02", sug.StartDate)
		if err != nil {
			continue
		}
		end, err := time.Parse("2006-01-02", sug.EndDate)
		if err != nil {
			continue
		}
		overrideID := uuid.NewString()
		// Stable draft key per supplier+template so re-runs upsert one inactive row.
		existingID, _ := s.findDraftOverrideID(ctx, supplierID, sug.TemplateID)
		if existingID != "" {
			overrideID = existingID
		}
		_, err = s.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			return txn.BufferWrite([]*spanner.Mutation{spanner.InsertOrUpdateMap("SeasonalTemplateOverrides", map[string]any{
				"OverrideId": overrideID,
				"SupplierId": supplierID,
				"TemplateId": sug.TemplateID,
				"Name":       sug.Name,
				"StartDate":  start,
				"EndDate":    end,
				"IsActive":   false,
				"Multiplier": sug.Multiplier,
				"CreatedAt":  spanner.CommitTimestamp,
			})})
		})
		if err != nil {
			return persisted, err
		}
		sug.DraftOverrideID = overrideID
		persisted++
	}
	return persisted, nil
}

func (s *Service) findDraftOverrideID(ctx context.Context, supplierID, templateID string) (string, error) {
	if s == nil || s.Spanner == nil {
		return "", nil
	}
	iter := s.Spanner.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT OverrideId FROM SeasonalTemplateOverrides
		      WHERE SupplierId = @sid AND TemplateId = @tid AND IsActive = false
		      ORDER BY CreatedAt DESC LIMIT 1`,
		Params: map[string]any{"sid": supplierID, "tid": templateID},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if errors.Is(err, iterator.Done) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	var id string
	if err := row.Columns(&id); err != nil {
		return "", err
	}
	return id, nil
}
