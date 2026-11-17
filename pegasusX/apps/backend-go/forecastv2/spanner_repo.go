package forecastv2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

// SpannerRepository implements Repository against Cloud Spanner / Spanner emulator.
// Every read uses StrongRead() or ReadOnlyTransaction() (zero unguarded stale single reads).
type SpannerRepository struct {
	client *spanner.Client
}

// NewSpannerRepository constructs a Spanner-backed forecastv2 Repository.
func NewSpannerRepository(client *spanner.Client) *SpannerRepository {
	return &SpannerRepository{client: client}
}

func (r *SpannerRepository) SaveRunAndValues(ctx context.Context, run ForecastRun, values []ForecastValue, explanations []ForecastExplanation) error {
	if r == nil || r.client == nil {
		return ErrRepoUnavailable
	}

	const batchLimit = 300
	firstValues := values
	var extraValues []ForecastValue
	if len(values) > batchLimit {
		firstValues = values[:batchLimit]
		extraValues = values[batchLimit:]
	}

	_, err := r.client.ReadWriteTransaction(ctx, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		muts := make([]*spanner.Mutation, 0, 2+len(firstValues)+len(explanations))
		locID := ""
		if len(values) > 0 {
			locID = values[0].LocationID
		}
		runMap := map[string]any{
			"EntityId":           run.EntityID,
			"RunId":              run.RunID,
			"SupplierId":         run.SupplierID,
			"HorizonDays":        run.HorizonDays,
			"Status":             run.Status,
			"ChampionModel":      run.ChampionModel,
			"ChallengerModel":    run.ChallengerModel,
			"WinnerModel":        run.WinnerModel,
			"ChampionWapeBps":    run.ChampionWapeBps,
			"ChallengerWapeBps":  run.ChallengerWapeBps,
			"ChampionBiasBps":    run.ChampionBiasBps,
			"ChallengerBiasBps":  run.ChallengerBiasBps,
			"WapeImprovementBps": run.WapeImprovementBps,
			"SkuCount":           run.SkuCount,
			"LocationCount":      run.LocationCount,
			"RegressorsJson":     nullableString(run.RegressorsJSON),
			"FeatureSource":      nullableString(run.FeatureSource),
			"TriggeredBy":        run.TriggeredBy,
			"FailureReason":      nullableString(run.FailureReason),
			"StartedAt":          spanner.CommitTimestamp,
			"CompletedAt":        spanner.CommitTimestamp,
			"CreatedAt":          spanner.CommitTimestamp,
			"UpdatedAt":          spanner.CommitTimestamp,
		}
		if locID != "" {
			runMap["LocationId"] = locID
		}
		muts = append(muts, spanner.InsertOrUpdateMap("ForecastRuns", runMap))

		for _, v := range firstValues {
			mut, mErr := forecastValueMutation(v)
			if mErr != nil {
				return mErr
			}
			muts = append(muts, mut)
		}
		for _, exp := range explanations {
			muts = append(muts, forecastExplanationMutation(exp, true))
		}

		buf := outbox.NewSpannerTxnBuffer(txn)
		completedAt := run.UpdatedAt.UTC().Format(time.RFC3339Nano)
		if err := outbox.EmitJSON(txCtx, buf, events.AggregateForecastRun, run.RunID, events.TopicIntel, events.ForecastRunCompletedEvent{
			BaseEvent: events.BaseEvent{
				Type:      events.EventForecastRunCompleted,
				Version:   1,
				Timestamp: completedAt,
			},
			EntityID:           run.EntityID,
			RunID:              run.RunID,
			SupplierID:         run.SupplierID,
			LocationID:         locID,
			HorizonDays:        run.HorizonDays,
			Status:             run.Status,
			ChampionModel:      run.ChampionModel,
			ChallengerModel:    run.ChallengerModel,
			WinnerModel:        run.WinnerModel,
			ChampionWapeBps:    run.ChampionWapeBps,
			ChallengerWapeBps:  run.ChallengerWapeBps,
			WapeImprovementBps: run.WapeImprovementBps,
			SkuCount:           run.SkuCount,
			LocationCount:      run.LocationCount,
			TriggeredBy:        run.TriggeredBy,
			CompletedAt:        completedAt,
		}); err != nil {
			return err
		}
		if err := txn.BufferWrite(muts); err != nil {
			return err
		}
		return buf.Flush(txCtx)
	})
	if err != nil {
		return err
	}

	for offset := 0; offset < len(extraValues); offset += batchLimit {
		end := offset + batchLimit
		if end > len(extraValues) {
			end = len(extraValues)
		}
		chunk := extraValues[offset:end]
		if _, err := r.client.ReadWriteTransaction(ctx, func(_ context.Context, txn *spanner.ReadWriteTransaction) error {
			muts := make([]*spanner.Mutation, 0, len(chunk))
			for _, v := range chunk {
				mut, mErr := forecastValueMutation(v)
				if mErr != nil {
					return mErr
				}
				muts = append(muts, mut)
			}
			return txn.BufferWrite(muts)
		}); err != nil {
			return err
		}
	}
	return nil
}

func forecastValueMutation(v ForecastValue) (*spanner.Mutation, error) {
	fDate, err := civil.ParseDate(v.ForecastDate)
	if err != nil {
		return nil, fmt.Errorf("invalid forecast_date %q: %w", v.ForecastDate, err)
	}
	locType := v.LocationType
	if locType == "" {
		locType = "WAREHOUSE"
	}
	regJSON := v.RegressorsJSON
	if regJSON == "" && len(v.Regressors) > 0 {
		b, _ := json.Marshal(v.Regressors)
		regJSON = string(b)
	}
	return spanner.InsertOrUpdateMap("ForecastValues", map[string]any{
		"EntityId":         v.EntityID,
		"RunId":            v.RunID,
		"ForecastDate":     fDate,
		"LocationId":       v.LocationID,
		"LocationType":     locType,
		"Sku":              v.SKU,
		"ProductId":        nullableString(v.ProductID),
		"Model":            v.Model,
		"IsWinner":         v.IsWinner,
		"DemandClass":      nullableString(v.DemandClass),
		"P10Milli":         v.P10Milli,
		"P50Milli":         v.P50Milli,
		"P90Milli":         v.P90Milli,
		"BaselineP50Milli": v.BaselineP50Milli,
		"WapeBps":          v.WapeBps,
		"BiasBps":          v.BiasBps,
		"ConfidenceBps":    v.ConfidenceBps,
		"RegressorsJson":   nullableString(regJSON),
		"CreatedAt":        spanner.CommitTimestamp,
	}), nil
}

func forecastExplanationMutation(exp ForecastExplanation, isCreate bool) *spanner.Mutation {
	evRefs := exp.EvidenceRefsJSON
	if evRefs == "" {
		b, _ := json.Marshal(exp.EvidenceRefs)
		evRefs = string(b)
	}
	if evRefs == "" || evRefs == "null" {
		evRefs = "[]"
	}
	m := map[string]any{
		"EntityId":          exp.EntityID,
		"ExplanationId":     exp.ExplanationID,
		"RunId":             nullableString(exp.RunID),
		"LocationId":        exp.LocationID,
		"Sku":               exp.SKU,
		"Source":            exp.Source,
		"ModelName":         nullableString(exp.ModelName),
		"SummaryText":       exp.SummaryText,
		"EvidenceRefsJson":  evRefs,
		"AdjustmentBps":     exp.AdjustmentBps,
		"AdjustmentReason":  nullableString(exp.AdjustmentReason),
		"Status":            exp.Status,
		"ProposedBy":        exp.ProposedBy,
		"ApprovedBy":        nullableString(exp.ApprovedBy),
		"ApprovalNote":      nullableString(exp.ApprovalNote),
		"ApprovalRequestId": nullableString(exp.ApprovalRequestID),
		"BaselineP50Milli":  exp.BaselineP50Milli,
		"AdjustedP50Milli":  exp.AdjustedP50Milli,
		"UpdatedAt":         spanner.CommitTimestamp,
	}
	if !exp.WindowStart.IsZero() {
		m["WindowStart"] = civil.DateOf(exp.WindowStart.UTC())
	}
	if !exp.WindowEnd.IsZero() {
		m["WindowEnd"] = civil.DateOf(exp.WindowEnd.UTC())
	}
	if exp.ApprovedAt != nil {
		m["ApprovedAt"] = spanner.CommitTimestamp
	}
	if exp.ActualQtyMilli > 0 {
		m["ActualQtyMilli"] = exp.ActualQtyMilli
		m["ForecastValueAddBps"] = exp.ForecastValueAddBps
	} else if exp.ForecastValueAddBps != 0 {
		m["ForecastValueAddBps"] = exp.ForecastValueAddBps
	}
	if isCreate {
		m["CreatedAt"] = spanner.CommitTimestamp
		return spanner.InsertOrUpdateMap("ForecastExplanations", m)
	}
	return spanner.UpdateMap("ForecastExplanations", m)
}

func (r *SpannerRepository) GetRun(ctx context.Context, entityID, runID string) (ForecastRun, bool, error) {
	if r == nil || r.client == nil {
		return ForecastRun{}, false, ErrRepoUnavailable
	}
	stmt := spanner.Statement{
		SQL: `SELECT EntityId, RunId, SupplierId, HorizonDays, Status, ChampionModel, ChallengerModel,
		             WinnerModel, ChampionWapeBps, ChallengerWapeBps, ChampionBiasBps, ChallengerBiasBps,
		             WapeImprovementBps, SkuCount, LocationCount, COALESCE(RegressorsJson, '[]'),
		             COALESCE(FeatureSource, ''), TriggeredBy, COALESCE(FailureReason, ''),
		             StartedAt, CompletedAt, CreatedAt, UpdatedAt
		      FROM ForecastRuns
		      WHERE EntityId = @eid AND RunId = @rid`,
		Params: map[string]any{"eid": entityID, "rid": runID},
	}
	iter := r.client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if errors.Is(err, iterator.Done) {
		return ForecastRun{}, false, nil
	}
	if err != nil {
		return ForecastRun{}, false, err
	}
	run, err := scanForecastRun(row)
	return run, err == nil, err
}

func (r *SpannerRepository) GetLatestRun(ctx context.Context, entityID string) (ForecastRun, bool, error) {
	if r == nil || r.client == nil {
		return ForecastRun{}, false, ErrRepoUnavailable
	}
	stmt := spanner.Statement{
		SQL: `SELECT EntityId, RunId, SupplierId, HorizonDays, Status, ChampionModel, ChallengerModel,
		             WinnerModel, ChampionWapeBps, ChallengerWapeBps, ChampionBiasBps, ChallengerBiasBps,
		             WapeImprovementBps, SkuCount, LocationCount, COALESCE(RegressorsJson, '[]'),
		             COALESCE(FeatureSource, ''), TriggeredBy, COALESCE(FailureReason, ''),
		             StartedAt, CompletedAt, CreatedAt, UpdatedAt
		      FROM ForecastRuns
		      WHERE EntityId = @eid
		      ORDER BY CreatedAt DESC
		      LIMIT 1`,
		Params: map[string]any{"eid": entityID},
	}
	iter := r.client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if errors.Is(err, iterator.Done) {
		return ForecastRun{}, false, nil
	}
	if err != nil {
		return ForecastRun{}, false, err
	}
	run, err := scanForecastRun(row)
	return run, err == nil, err
}

func scanForecastRun(row *spanner.Row) (ForecastRun, error) {
	var out ForecastRun
	var completed spanner.NullTime
	if err := row.Columns(
		&out.EntityID, &out.RunID, &out.SupplierID, &out.HorizonDays, &out.Status,
		&out.ChampionModel, &out.ChallengerModel, &out.WinnerModel,
		&out.ChampionWapeBps, &out.ChallengerWapeBps, &out.ChampionBiasBps, &out.ChallengerBiasBps,
		&out.WapeImprovementBps, &out.SkuCount, &out.LocationCount, &out.RegressorsJSON,
		&out.FeatureSource, &out.TriggeredBy, &out.FailureReason,
		&out.StartedAt, &completed, &out.CreatedAt, &out.UpdatedAt,
	); err != nil {
		return out, err
	}
	if completed.Valid {
		t := completed.Time
		out.CompletedAt = &t
	}
	if strings.TrimSpace(out.RegressorsJSON) != "" {
		_ = json.Unmarshal([]byte(out.RegressorsJSON), &out.Regressors)
	}
	return out, nil
}

func (r *SpannerRepository) ListForecastValues(ctx context.Context, q ForecastQuery) ([]ForecastValue, error) {
	if r == nil || r.client == nil {
		return nil, ErrRepoUnavailable
	}
	sql := `SELECT EntityId, RunId, CAST(ForecastDate AS STRING), LocationId, LocationType,
	               Sku, COALESCE(ProductId, ''), Model, IsWinner, COALESCE(DemandClass, ''),
	               P10Milli, P50Milli, P90Milli, BaselineP50Milli, WapeBps, BiasBps, ConfidenceBps,
	               COALESCE(RegressorsJson, '[]'), CreatedAt
	        FROM ForecastValues
	        WHERE EntityId = @eid`
	params := map[string]any{"eid": q.EntityID}
	if q.RunID != "" {
		sql += " AND RunId = @rid"
		params["rid"] = q.RunID
	}
	if q.LocationID != "" {
		sql += " AND LocationId = @loc"
		params["loc"] = q.LocationID
	}
	if q.SKU != "" {
		sql += " AND Sku = @sku"
		params["sku"] = q.SKU
	}
	limit := q.Limit
	if limit <= 0 || limit > 2000 {
		limit = 500
	}
	sql += " ORDER BY ForecastDate ASC, Sku ASC, LocationId ASC, Model ASC LIMIT @lim"
	params["lim"] = int64(limit)

	iter := r.client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	var out []ForecastValue
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var v ForecastValue
		if err := row.Columns(
			&v.EntityID, &v.RunID, &v.ForecastDate, &v.LocationID, &v.LocationType,
			&v.SKU, &v.ProductID, &v.Model, &v.IsWinner, &v.DemandClass,
			&v.P10Milli, &v.P50Milli, &v.P90Milli, &v.BaselineP50Milli, &v.WapeBps, &v.BiasBps, &v.ConfidenceBps,
			&v.RegressorsJSON, &v.CreatedAt,
		); err != nil {
			return nil, err
		}
		if strings.TrimSpace(v.RegressorsJSON) != "" {
			_ = json.Unmarshal([]byte(v.RegressorsJSON), &v.Regressors)
		}
		out = append(out, v)
	}
	return out, nil
}

func (r *SpannerRepository) ListExplanations(ctx context.Context, entityID, runID, locationID, sku string, limit int) ([]ForecastExplanation, error) {
	if r == nil || r.client == nil {
		return nil, ErrRepoUnavailable
	}
	sql := `SELECT EntityId, ExplanationId, COALESCE(RunId, ''), LocationId, Sku,
	               WindowStart, WindowEnd, Source, COALESCE(ModelName, ''), SummaryText,
	               EvidenceRefsJson, AdjustmentBps, COALESCE(AdjustmentReason, ''), Status,
	               ProposedBy, COALESCE(ApprovedBy, ''), ApprovedAt, COALESCE(ApprovalNote, ''),
	               COALESCE(ApprovalRequestId, ''), BaselineP50Milli, AdjustedP50Milli,
	               COALESCE(ActualQtyMilli, 0), COALESCE(ForecastValueAddBps, 0), CreatedAt, UpdatedAt
	        FROM ForecastExplanations
	        WHERE EntityId = @eid`
	params := map[string]any{"eid": entityID}
	if runID != "" {
		sql += " AND RunId = @rid"
		params["rid"] = runID
	}
	if locationID != "" {
		sql += " AND LocationId = @loc"
		params["loc"] = locationID
	}
	if sku != "" {
		sql += " AND Sku = @sku"
		params["sku"] = sku
	}
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	sql += " ORDER BY CreatedAt DESC LIMIT @lim"
	params["lim"] = int64(limit)

	iter := r.client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	var out []ForecastExplanation
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		exp, err := scanForecastExplanation(row)
		if err != nil {
			return nil, err
		}
		out = append(out, exp)
	}
	return out, nil
}

func (r *SpannerRepository) GetExplanation(ctx context.Context, entityID, explanationID string) (ForecastExplanation, bool, error) {
	if r == nil || r.client == nil {
		return ForecastExplanation{}, false, ErrRepoUnavailable
	}
	stmt := spanner.Statement{
		SQL: `SELECT EntityId, ExplanationId, COALESCE(RunId, ''), LocationId, Sku,
		             WindowStart, WindowEnd, Source, COALESCE(ModelName, ''), SummaryText,
		             EvidenceRefsJson, AdjustmentBps, COALESCE(AdjustmentReason, ''), Status,
		             ProposedBy, COALESCE(ApprovedBy, ''), ApprovedAt, COALESCE(ApprovalNote, ''),
		             COALESCE(ApprovalRequestId, ''), BaselineP50Milli, AdjustedP50Milli,
		             COALESCE(ActualQtyMilli, 0), COALESCE(ForecastValueAddBps, 0), CreatedAt, UpdatedAt
		      FROM ForecastExplanations
		      WHERE EntityId = @eid AND ExplanationId = @xid`,
		Params: map[string]any{"eid": entityID, "xid": explanationID},
	}
	iter := r.client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if errors.Is(err, iterator.Done) {
		return ForecastExplanation{}, false, nil
	}
	if err != nil {
		return ForecastExplanation{}, false, err
	}
	exp, err := scanForecastExplanation(row)
	return exp, err == nil, err
}

func scanForecastExplanation(row *spanner.Row) (ForecastExplanation, error) {
	var exp ForecastExplanation
	var winStart, winEnd spanner.NullDate
	var approvedAt spanner.NullTime
	if err := row.Columns(
		&exp.EntityID, &exp.ExplanationID, &exp.RunID, &exp.LocationID, &exp.SKU,
		&winStart, &winEnd, &exp.Source, &exp.ModelName, &exp.SummaryText,
		&exp.EvidenceRefsJSON, &exp.AdjustmentBps, &exp.AdjustmentReason, &exp.Status,
		&exp.ProposedBy, &exp.ApprovedBy, &approvedAt, &exp.ApprovalNote,
		&exp.ApprovalRequestID, &exp.BaselineP50Milli, &exp.AdjustedP50Milli,
		&exp.ActualQtyMilli, &exp.ForecastValueAddBps, &exp.CreatedAt, &exp.UpdatedAt,
	); err != nil {
		return exp, err
	}
	if winStart.Valid {
		exp.WindowStart = winStart.Date.In(time.UTC)
	}
	if winEnd.Valid {
		exp.WindowEnd = winEnd.Date.In(time.UTC)
	}
	if approvedAt.Valid {
		t := approvedAt.Time
		exp.ApprovedAt = &t
	}
	if strings.TrimSpace(exp.EvidenceRefsJSON) != "" {
		_ = json.Unmarshal([]byte(exp.EvidenceRefsJSON), &exp.EvidenceRefs)
	}
	return exp, nil
}

func (r *SpannerRepository) SaveExplanation(ctx context.Context, exp ForecastExplanation) error {
	if r == nil || r.client == nil {
		return ErrRepoUnavailable
	}
	_, err := r.client.ReadWriteTransaction(ctx, func(_ context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{forecastExplanationMutation(exp, true)})
	})
	return err
}

func (r *SpannerRepository) UpdateExplanation(ctx context.Context, exp ForecastExplanation) error {
	if r == nil || r.client == nil {
		return ErrRepoUnavailable
	}
	_, err := r.client.ReadWriteTransaction(ctx, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		muts := []*spanner.Mutation{forecastExplanationMutation(exp, false)}
		if exp.Status == ExplStatusApproved && exp.AdjustedP50Milli > 0 {
			day := civil.DateOf(time.Now().UTC())
			if !exp.WindowStart.IsZero() {
				day = civil.DateOf(exp.WindowStart.UTC())
			}
			// Also mirror approved adjustment into DemandForecastBaseline when updating live planning.
			qtyUnits := (exp.AdjustedP50Milli + MilliScale/2) / MilliScale
			muts = append(muts, spanner.InsertOrUpdateMap("DemandForecastBaseline", map[string]any{
				"SupplierId":     exp.EntityID,
				"ForecastDate":   day,
				"WarehouseId":    exp.LocationID,
				"ProductId":      exp.SKU,
				"BaselineQty":    qtyUnits,
				"Confidence":     0.85,
				"Source":         "forecast_v2_approved",
				"BaselineSource": "forecast_v2_approved",
				"CreatedAt":      spanner.CommitTimestamp,
			}))
		}
		buf := outbox.NewSpannerTxnBuffer(txn)
		if err := outbox.EmitJSON(txCtx, buf, events.AggregatePlanning, exp.EntityID, events.TopicMain, events.PlanningEvent{
			BaseEvent: events.BaseEvent{
				Type:      events.EventPlanningForecastUpdated,
				Version:   1,
				Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
			},
			SupplierID:  exp.EntityID,
			WarehouseID: exp.LocationID,
			ProductID:   exp.SKU,
			Action:      "explanation_" + strings.ToLower(exp.Status),
		}); err != nil {
			return err
		}
		if err := txn.BufferWrite(muts); err != nil {
			return err
		}
		return buf.Flush(txCtx)
	})
	return err
}

func (r *SpannerRepository) LoadHistoricalSeries(ctx context.Context, entityID, locationID, sku string, asOf time.Time, lookbackDays int) ([]SeriesHistoryInput, []SeasonalWindow, error) {
	if r == nil || r.client == nil {
		return nil, nil, ErrRepoUnavailable
	}
	if lookbackDays <= 0 {
		lookbackDays = 60
	}
	ro := r.client.ReadOnlyTransaction()
	defer ro.Close()

	// 1. Load custom seasonal windows.
	var windows []SeasonalWindow
	ovIter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT OverrideId, TemplateId, COALESCE(Name, ''), StartDate, EndDate, COALESCE(Multiplier, 1.0)
		      FROM SeasonalTemplateOverrides
		      WHERE SupplierId = @sid AND IsActive = true
		      ORDER BY StartDate ASC`,
		Params: map[string]any{"sid": entityID},
	})
	for {
		row, err := ovIter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			break
		}
		var w SeasonalWindow
		var mult float64
		if err := row.Columns(&w.OverrideID, &w.TemplateID, &w.Name, &w.StartDate, &w.EndDate, &mult); err == nil {
			w.MultiplierBps = int64(math.Round(mult * float64(BpsScale)))
			windows = append(windows, w)
		}
	}
	ovIter.Stop()

	endDay := civil.DateOf(asOf.UTC())
	startDay := endDay.AddDays(-(lookbackDays - 1))
	startTS := startDay.In(time.UTC)
	endTS := endDay.AddDays(1).In(time.UTC)

	type seriesKey struct{ loc, sku string }
	dailyMap := map[seriesKey]map[civil.Date]int64{}

	// 2. Load completed/non-cancelled order line-item history.
	ordSQL := `SELECT COALESCE(WarehouseId, 'ALL'), LineItemsJson, CreatedAt
	           FROM Orders
	           WHERE SupplierId = @sid
	             AND Status NOT IN ('CANCELLED', 'REJECTED', 'VOIDED')
	             AND CreatedAt >= @start AND CreatedAt < @end`
	ordParams := map[string]any{"sid": entityID, "start": startTS, "end": endTS}
	if locationID != "" {
		ordSQL += " AND WarehouseId = @loc"
		ordParams["loc"] = locationID
	}
	ordSQL += " LIMIT 2000"

	ordIter := ro.Query(ctx, spanner.Statement{SQL: ordSQL, Params: ordParams})
	for {
		row, err := ordIter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			break
		}
		var whID string
		var rawLines []byte
		var createdAt time.Time
		if err := row.Columns(&whID, &rawLines, &createdAt); err != nil {
			continue
		}
		if strings.TrimSpace(whID) == "" {
			whID = "ALL"
		}
		var items []struct {
			SKU       string `json:"sku"`
			ProductID string `json:"product_id"`
			Quantity  int64  `json:"quantity"`
			Qty       int64  `json:"qty"`
		}
		if err := json.Unmarshal(rawLines, &items); err != nil {
			continue
		}
		day := civil.DateOf(createdAt.UTC())
		for _, it := range items {
			itemSKU := strings.TrimSpace(it.SKU)
			if itemSKU == "" {
				itemSKU = strings.TrimSpace(it.ProductID)
			}
			if itemSKU == "" || (sku != "" && itemSKU != sku) {
				continue
			}
			q := it.Quantity
			if q <= 0 {
				q = it.Qty
			}
			if q <= 0 {
				continue
			}
			k := seriesKey{loc: whID, sku: itemSKU}
			if dailyMap[k] == nil {
				dailyMap[k] = map[civil.Date]int64{}
			}
			dailyMap[k][day] += q * MilliScale
		}
	}
	ordIter.Stop()

	// 3. Fallback/augment from DemandForecastBaseline when Orders are sparse.
	if len(dailyMap) == 0 {
		baseSQL := `SELECT WarehouseId, ProductId, ForecastDate, BaselineQty
		            FROM DemandForecastBaseline
		            WHERE SupplierId = @sid AND ForecastDate BETWEEN @start AND @end`
		baseParams := map[string]any{"sid": entityID, "start": startDay, "end": endDay}
		if locationID != "" {
			baseSQL += " AND WarehouseId = @loc"
			baseParams["loc"] = locationID
		}
		if sku != "" {
			baseSQL += " AND ProductId = @sku"
			baseParams["sku"] = sku
		}
		baseIter := ro.Query(ctx, spanner.Statement{SQL: baseSQL, Params: baseParams})
		for {
			row, err := baseIter.Next()
			if errors.Is(err, iterator.Done) {
				break
			}
			if err != nil {
				break
			}
			var whID, prodID string
			var fDate civil.Date
			var qty int64
			if err := row.Columns(&whID, &prodID, &fDate, &qty); err == nil && qty > 0 {
				if whID == "" {
					whID = "ALL"
				}
				k := seriesKey{loc: whID, sku: prodID}
				if dailyMap[k] == nil {
					dailyMap[k] = map[civil.Date]int64{}
				}
				dailyMap[k][fDate] += qty * MilliScale
			}
		}
		baseIter.Stop()
	}

	keys := make([]seriesKey, 0, len(dailyMap))
	for k := range dailyMap {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].loc != keys[j].loc {
			return keys[i].loc < keys[j].loc
		}
		return keys[i].sku < keys[j].sku
	})

	out := make([]SeriesHistoryInput, 0, len(keys))
	for _, k := range keys {
		dayMap := dailyMap[k]
		dates := make([]civil.Date, 0, lookbackDays)
		vals := make([]int64, 0, lookbackDays)
		for d := startDay; !d.After(endDay); d = d.AddDays(1) {
			dates = append(dates, d)
			vals = append(vals, dayMap[d])
		}
		out = append(out, SeriesHistoryInput{
			LocationID:      k.loc,
			LocationType:    "WAREHOUSE",
			SKU:             k.sku,
			ProductID:       k.sku,
			DailyUnitsMilli: vals,
			Dates:           dates,
		})
	}
	return out, windows, nil
}

func (r *SpannerRepository) LoadActualMilli(ctx context.Context, entityID, locationID, sku string, windowStart, windowEnd time.Time) (int64, bool, error) {
	if r == nil || r.client == nil {
		return 0, false, ErrRepoUnavailable
	}
	startDay := civil.DateOf(windowStart.UTC())
	endDay := civil.DateOf(windowEnd.UTC())
	if windowStart.IsZero() {
		endDay = civil.DateOf(time.Now().UTC())
		startDay = endDay.AddDays(-7)
	}
	sql := `SELECT COALESCE(SUM(ActualQty), 0), COUNT(*)
	        FROM ForecastAccuracyDaily
	        WHERE SupplierId = @sid AND ProductId = @sku
	          AND ForecastDate BETWEEN @start AND @end`
	params := map[string]any{"sid": entityID, "sku": sku, "start": startDay, "end": endDay}
	if locationID != "" && locationID != "ALL" {
		sql += " AND WarehouseId = @loc"
		params["loc"] = locationID
	}
	iter := r.client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return 0, false, nil
	}
	var sumActual, count int64
	if err := row.Columns(&sumActual, &count); err != nil || count == 0 || sumActual <= 0 {
		return 0, false, nil
	}
	avgDailyMilli := (sumActual * MilliScale) / count
	return avgDailyMilli, true, nil
}

func (r *SpannerRepository) IsRetailerLinked(ctx context.Context, supplierID, retailerID string) (bool, error) {
	if r == nil || r.client == nil {
		return false, ErrRepoUnavailable
	}
	ro := r.client.ReadOnlyTransaction()
	defer ro.Close()

	// Check SupplierRetailerLinks, RetailerPaymentTerms, or Orders.
	if _, err := ro.ReadRow(ctx, "SupplierRetailerLinks", spanner.Key{supplierID, retailerID}, []string{"RetailerId"}); err == nil {
		return true, nil
	} else if spanner.ErrCode(err) != codes.NotFound && !strings.Contains(err.Error(), "not found") {
		// continue checking other link tables cleanly
	}
	if _, err := ro.ReadRow(ctx, "RetailerPaymentTerms", spanner.Key{retailerID, supplierID}, []string{"RetailerId"}); err == nil {
		return true, nil
	}
	iter := ro.Query(ctx, spanner.Statement{
		SQL:    `SELECT 1 FROM Orders WHERE SupplierId = @sid AND RetailerId = @rid LIMIT 1`,
		Params: map[string]any{"sid": supplierID, "rid": retailerID},
	})
	defer iter.Stop()
	_, err := iter.Next()
	return err == nil, nil
}

func nullableString(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}
