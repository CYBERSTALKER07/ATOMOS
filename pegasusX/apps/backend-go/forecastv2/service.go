package forecastv2

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// Service orchestrates M1.5 hybrid forecasting, Gemini/classic explainability,
// demand adjustment proposals, human approval, and FVA tracking.
type Service struct {
	repo      Repository
	explainer *GeminiExplainer
	now       func() time.Time
}

// NewService creates a Spanner-backed forecastv2 Service (or empty-repo safe service when client == nil).
func NewService(client *spanner.Client) *Service {
	if client == nil {
		return NewServiceWithRepository(nil)
	}
	return NewServiceWithRepository(NewSpannerRepository(client))
}

// NewServiceWithRepository creates a forecastv2 Service with a custom Repository.
func NewServiceWithRepository(repo Repository) *Service {
	return &Service{
		repo:      repo,
		explainer: NewGeminiExplainer(GeminiConfigFromEnv()),
		now:       func() time.Time { return time.Now().UTC() },
	}
}

// SetGeminiConfig configures the Gemini explainer client on s.
func (s *Service) SetGeminiConfig(cfg GeminiConfig) {
	if s != nil {
		s.explainer = NewGeminiExplainer(cfg)
	}
}

// SetNow overrides the clock for deterministic tests.
func (s *Service) SetNow(now func() time.Time) {
	if s != nil && now != nil {
		s.now = now
	}
}

// Repo returns the underlying Repository.
func (s *Service) Repo() Repository {
	if s == nil {
		return nil
	}
	return s.repo
}

func (s *Service) currentNow() time.Time {
	if s != nil && s.now != nil {
		return s.now().UTC()
	}
	return time.Now().UTC()
}

// RunForecast runs Champion (classic) vs Challenger (global seasonal/calendar) over historical
// demand series, computes backtest WAPE/bias, persists ForecastRuns + ForecastValues + ForecastExplanations,
// and emits forecast.run.completed atomically in Spanner mode.
func (s *Service) RunForecast(ctx context.Context, in RunForecastInput) (RunForecastOutput, error) {
	if s == nil || s.repo == nil {
		return RunForecastOutput{}, ErrRepoUnavailable
	}
	entityID := strings.TrimSpace(in.EntityID)
	if entityID == "" {
		entityID = strings.TrimSpace(in.SupplierID)
	}
	if entityID == "" {
		return RunForecastOutput{}, fmt.Errorf("%w: entity_id or supplier_id is required", ErrInvalidInput)
	}
	supplierID := strings.TrimSpace(in.SupplierID)
	if supplierID == "" {
		supplierID = entityID
	}

	horizonDays := in.HorizonDays
	if horizonDays <= 0 {
		horizonDays = DefaultHorizonDays
	}
	if horizonDays > MaxHorizonDays {
		horizonDays = MaxHorizonDays
	}

	asOf := in.AsOf.UTC()
	if asOf.IsZero() {
		asOf = s.currentNow()
	}

	series := in.Series
	var windows []SeasonalWindow
	featureSource := "caller_series"
	if len(series) == 0 {
		loaded, loadedWindows, err := s.repo.LoadHistoricalSeries(ctx, entityID, strings.TrimSpace(in.LocationID), strings.TrimSpace(in.SKU), asOf, 60)
		if err != nil {
			return RunForecastOutput{}, err
		}
		series = loaded
		windows = loadedWindows
		featureSource = "spanner_orders_sell_through"
	} else {
		_, loadedWindows, err := s.repo.LoadHistoricalSeries(ctx, entityID, strings.TrimSpace(in.LocationID), strings.TrimSpace(in.SKU), asOf, 60)
		if err == nil {
			windows = loadedWindows
		}
	}

	runID := "fcr_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:24]
	startedAt := s.currentNow()

	evals, err := EvaluateSeriesBatch(
		entityID, runID, asOf, horizonDays,
		series, windows,
		in.PromoDiscountBps, in.PriceChangeBps, in.ActivePromo,
	)
	if err != nil {
		return RunForecastOutput{}, err
	}

	var values []ForecastValue
	var explanations []ForecastExplanation
	skuSet := map[string]struct{}{}
	locSet := map[string]struct{}{}
	regMap := map[string]RegressorSignal{}
	champCount := 0
	chalCount := 0
	var champWapeSum, chalWapeSum, champBiasSum, chalBiasSum int64
	champModels := map[string]int{}

	explainEnabled := true
	if in.Explain != nil {
		explainEnabled = *in.Explain
	}

	triggeredBy := strings.TrimSpace(in.TriggeredBy)
	if triggeredBy == "" {
		triggeredBy = auth.ActorFromContext(ctx)
	}

	for _, ev := range evals {
		skuSet[ev.SKU] = struct{}{}
		locSet[ev.LocationID] = struct{}{}
		champModels[ev.ChampionModel]++
		champWapeSum += ev.ChampionWapeBps
		chalWapeSum += ev.ChallengerWapeBps
		champBiasSum += ev.ChampionBiasBps
		chalBiasSum += ev.ChallengerBiasBps
		if ev.WinnerModel == ModelGlobalSeasonal {
			chalCount++
		} else {
			champCount++
		}
		for _, rg := range ev.ActiveRegressors {
			regMap[rg.Name] = rg
		}
		for i := range ev.ClassicValues {
			ev.ClassicValues[i].CreatedAt = startedAt
			values = append(values, ev.ClassicValues[i])
		}
		for i := range ev.ChallengerValues {
			ev.ChallengerValues[i].CreatedAt = startedAt
			values = append(values, ev.ChallengerValues[i])
		}

		if explainEnabled && len(ev.ChallengerValues) > 0 {
			expOut, expErr := s.explainer.Explain(ctx, ExplainInput{
				EntityID:   entityID,
				RunID:      runID,
				Evaluation: ev,
			})
			if expErr != nil {
				return RunForecastOutput{}, expErr
			}
			baselineP50 := ev.ClassicValues[0].P50Milli
			if ev.WinnerModel == ModelGlobalSeasonal {
				baselineP50 = ev.ChallengerValues[0].P50Milli
			}
			adjustedP50 := (baselineP50 * (BpsScale + expOut.AdjustmentBps)) / BpsScale
			if adjustedP50 < 0 {
				adjustedP50 = 0
			}
			evRefsJSON, _ := json.Marshal(expOut.EvidenceRefs)
			winStart := civil.DateOf(asOf).AddDays(1).In(time.UTC)
			winEnd := civil.DateOf(asOf).AddDays(horizonDays).In(time.UTC)

			proposedBy := "system:forecastv2"
			if expOut.Source == ExplSourceGemini {
				proposedBy = "gemini:" + expOut.ModelName
			}

			explanations = append(explanations, ForecastExplanation{
				EntityID:         entityID,
				ExplanationID:    "fex_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:24],
				RunID:            runID,
				LocationID:       ev.LocationID,
				SKU:              ev.SKU,
				WindowStart:      winStart,
				WindowEnd:        winEnd,
				Source:           expOut.Source,
				ModelName:        expOut.ModelName,
				SummaryText:      expOut.SummaryText,
				EvidenceRefs:     expOut.EvidenceRefs,
				EvidenceRefsJSON: string(evRefsJSON),
				AdjustmentBps:    expOut.AdjustmentBps,
				AdjustmentReason: expOut.AdjustmentReason,
				Status:           ExplStatusProposed,
				ProposedBy:       proposedBy,
				BaselineP50Milli: baselineP50,
				AdjustedP50Milli: adjustedP50,
				CreatedAt:        startedAt,
				UpdatedAt:        startedAt,
			})
		}
	}

	n := int64(len(evals))
	var avgChampWape, avgChalWape, avgChampBias, avgChalBias int64
	if n > 0 {
		avgChampWape = champWapeSum / n
		avgChalWape = chalWapeSum / n
		avgChampBias = champBiasSum / n
		avgChalBias = chalBiasSum / n
	}
	dominantChamp := ModelClassicSES
	bestCnt := -1
	for m, cnt := range champModels {
		if cnt > bestCnt {
			bestCnt = cnt
			dominantChamp = m
		}
	}
	winnerModel := dominantChamp
	if chalCount > champCount || (avgChalWape > 0 && avgChalWape < avgChampWape) {
		winnerModel = ModelGlobalSeasonal
	}

	regs := make([]RegressorSignal, 0, len(regMap))
	for _, rg := range regMap {
		regs = append(regs, rg)
	}
	regsJSON, _ := json.Marshal(regs)
	completedAt := s.currentNow()

	run := ForecastRun{
		EntityID:           entityID,
		RunID:              runID,
		SupplierID:         supplierID,
		HorizonDays:        int64(horizonDays),
		Status:             StatusCompleted,
		ChampionModel:      dominantChamp,
		ChallengerModel:    ModelGlobalSeasonal,
		WinnerModel:        winnerModel,
		ChampionWapeBps:    avgChampWape,
		ChallengerWapeBps:  avgChalWape,
		ChampionBiasBps:    avgChampBias,
		ChallengerBiasBps:  avgChalBias,
		WapeImprovementBps: avgChampWape - avgChalWape,
		SkuCount:           int64(len(skuSet)),
		LocationCount:      int64(len(locSet)),
		Regressors:         regs,
		RegressorsJSON:     string(regsJSON),
		FeatureSource:      featureSource,
		TriggeredBy:        triggeredBy,
		StartedAt:          startedAt,
		CompletedAt:        &completedAt,
		CreatedAt:          startedAt,
		UpdatedAt:          completedAt,
	}

	if err := s.repo.SaveRunAndValues(ctx, run, values, explanations); err != nil {
		return RunForecastOutput{}, err
	}
	return RunForecastOutput{
		Run:          run,
		Forecasts:    values,
		Explanations: explanations,
	}, nil
}

// GetForecast returns the active/requested run, forecast quantiles, and explanations.
func (s *Service) GetForecast(ctx context.Context, q ForecastQuery) (ForecastView, error) {
	if s == nil || s.repo == nil {
		return ForecastView{}, ErrRepoUnavailable
	}
	entityID := strings.TrimSpace(q.EntityID)
	if entityID == "" {
		return ForecastView{}, fmt.Errorf("%w: entity_id is required", ErrInvalidInput)
	}
	q.EntityID = entityID
	var runPtr *ForecastRun
	if strings.TrimSpace(q.RunID) != "" {
		r, ok, err := s.repo.GetRun(ctx, entityID, strings.TrimSpace(q.RunID))
		if err != nil {
			return ForecastView{}, err
		}
		if ok {
			runPtr = &r
			q.RunID = r.RunID
		}
	} else {
		r, ok, err := s.repo.GetLatestRun(ctx, entityID)
		if err != nil {
			return ForecastView{}, err
		}
		if ok {
			runPtr = &r
			q.RunID = r.RunID
		}
	}

	vals, err := s.repo.ListForecastValues(ctx, q)
	if err != nil {
		return ForecastView{}, err
	}
	if vals == nil {
		vals = []ForecastValue{}
	}
	exps, err := s.repo.ListExplanations(ctx, entityID, q.RunID, strings.TrimSpace(q.LocationID), strings.TrimSpace(q.SKU), q.Limit)
	if err != nil {
		return ForecastView{}, err
	}
	if exps == nil {
		exps = []ForecastExplanation{}
	}
	return ForecastView{
		Run:          runPtr,
		Forecasts:    vals,
		Explanations: exps,
	}, nil
}

// ProposeDemandAdjustment creates a grounded ForecastExplanation adjustment proposal
// using Gemini (when configured) or the classic structural explainer locally.
func (s *Service) ProposeDemandAdjustment(ctx context.Context, in ProposeAdjustmentInput) (ForecastExplanation, error) {
	if s == nil || s.repo == nil {
		return ForecastExplanation{}, ErrRepoUnavailable
	}
	entityID := strings.TrimSpace(in.EntityID)
	sku := strings.TrimSpace(in.SKU)
	locID := strings.TrimSpace(in.LocationID)
	if locID == "" {
		locID = "ALL"
	}
	if entityID == "" || sku == "" {
		return ForecastExplanation{}, fmt.Errorf("%w: entity_id and sku are required", ErrInvalidInput)
	}
	runID := strings.TrimSpace(in.RunID)
	if runID == "" {
		if latest, ok, err := s.repo.GetLatestRun(ctx, entityID); err == nil && ok {
			runID = latest.RunID
		}
	}

	var baselineP50Milli int64
	var eval SeriesEvaluation
	eval.LocationID = locID
	eval.SKU = sku
	eval.ChampionModel = ModelClassicSES
	eval.ChallengerModel = ModelGlobalSeasonal
	eval.WinnerModel = ModelClassicSES
	eval.DemandClass = "smooth"

	if runID != "" {
		vals, err := s.repo.ListForecastValues(ctx, ForecastQuery{
			EntityID:   entityID,
			RunID:      runID,
			LocationID: locID,
			SKU:        sku,
		})
		if err == nil && len(vals) > 0 {
			for _, v := range vals {
				if v.IsWinner {
					baselineP50Milli = v.P50Milli
					eval.WinnerModel = v.Model
					eval.DemandClass = v.DemandClass
					eval.ObservedMeanMilli = v.BaselineP50Milli
					eval.ActiveRegressors = v.Regressors
					if v.Model == ModelGlobalSeasonal {
						eval.ChallengerWapeBps = v.WapeBps
						eval.ChallengerBiasBps = v.BiasBps
					} else {
						eval.ChampionWapeBps = v.WapeBps
						eval.ChampionBiasBps = v.BiasBps
					}
					break
				}
			}
			if baselineP50Milli == 0 {
				baselineP50Milli = vals[0].P50Milli
			}
		}
	}

	expOut, err := s.explainer.Explain(ctx, ExplainInput{
		EntityID:       entityID,
		RunID:          runID,
		Evaluation:     eval,
		Notes:          in.Notes,
		OverrideDelta:  in.AdjustmentBps,
		OverrideReason: in.AdjustmentReason,
	})
	if err != nil {
		return ForecastExplanation{}, err
	}

	adjustedP50 := (baselineP50Milli * (BpsScale + expOut.AdjustmentBps)) / BpsScale
	if adjustedP50 < 0 {
		adjustedP50 = 0
	}
	now := s.currentNow()
	proposedBy := strings.TrimSpace(in.ProposedBy)
	if proposedBy == "" {
		proposedBy = auth.ActorFromContext(ctx)
	}
	if proposedBy == "" {
		proposedBy = "system:forecastv2"
	}
	evRefsJSON, _ := json.Marshal(expOut.EvidenceRefs)

	exp := ForecastExplanation{
		EntityID:         entityID,
		ExplanationID:    "fex_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:24],
		RunID:            runID,
		LocationID:       locID,
		SKU:              sku,
		WindowStart:      now.Truncate(24*time.Hour).AddDate(0, 0, 1),
		WindowEnd:        now.Truncate(24*time.Hour).AddDate(0, 0, DefaultHorizonDays),
		Source:           expOut.Source,
		ModelName:        expOut.ModelName,
		SummaryText:      expOut.SummaryText,
		EvidenceRefs:     expOut.EvidenceRefs,
		EvidenceRefsJSON: string(evRefsJSON),
		AdjustmentBps:    expOut.AdjustmentBps,
		AdjustmentReason: expOut.AdjustmentReason,
		Status:           ExplStatusProposed,
		ProposedBy:       proposedBy,
		BaselineP50Milli: baselineP50Milli,
		AdjustedP50Milli: adjustedP50,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := s.repo.SaveExplanation(ctx, exp); err != nil {
		return ForecastExplanation{}, err
	}
	return exp, nil
}

// ApproveDemandAdjustment approves or rejects a proposed ForecastExplanation,
// enforcing AGENT/AUDITOR bans and maker!=checker SoD, and computes ForecastValueAddBps.
func (s *Service) ApproveDemandAdjustment(ctx context.Context, in ApproveAdjustmentInput) (ForecastExplanation, error) {
	if s == nil || s.repo == nil {
		return ForecastExplanation{}, ErrRepoUnavailable
	}
	entityID := strings.TrimSpace(in.EntityID)
	expID := strings.TrimSpace(in.ExplanationID)
	if entityID == "" || expID == "" {
		return ForecastExplanation{}, fmt.Errorf("%w: entity_id and explanation_id are required", ErrInvalidInput)
	}

	actorType := strings.ToLower(strings.TrimSpace(in.ActorType))
	actorRole := strings.ToUpper(strings.TrimSpace(in.ActorRole))
	if actorType == auth.ActorTypeAgent || actorRole == auth.SupplierRoleAgent {
		return ForecastExplanation{}, ErrAgentForbidden
	}
	if actorRole == auth.SupplierRoleAuditor {
		return ForecastExplanation{}, ErrAuditorReadOnly
	}

	actorID := strings.TrimSpace(in.ActorID)
	if actorID == "" {
		actorID = auth.ActorFromContext(ctx)
	}
	if actorID == "" {
		return ForecastExplanation{}, fmt.Errorf("%w: actor_id is required", ErrInvalidInput)
	}

	decision := strings.ToUpper(strings.TrimSpace(in.Decision))
	if decision == "" {
		decision = "APPROVE"
	}
	if decision != "APPROVE" && decision != "REJECT" {
		return ForecastExplanation{}, fmt.Errorf("%w: decision must be APPROVE or REJECT", ErrInvalidInput)
	}

	exp, found, err := s.repo.GetExplanation(ctx, entityID, expID)
	if err != nil {
		return ForecastExplanation{}, err
	}
	if !found {
		return ForecastExplanation{}, ErrExplanationNotFound
	}
	if exp.Status != ExplStatusProposed {
		return ForecastExplanation{}, ErrExplanationAlreadyDecided
	}
	// Segregation of duties: human proposer cannot approve own proposal.
	if strings.EqualFold(actorID, strings.TrimSpace(exp.ProposedBy)) {
		return ForecastExplanation{}, ErrSoDViolation
	}

	now := s.currentNow()
	if in.AdjustmentBps != nil {
		exp.AdjustmentBps = clampAdjustmentBps(*in.AdjustmentBps)
		exp.AdjustedP50Milli = (exp.BaselineP50Milli * (BpsScale + exp.AdjustmentBps)) / BpsScale
		if exp.AdjustedP50Milli < 0 {
			exp.AdjustedP50Milli = 0
		}
	}

	if decision == "REJECT" {
		exp.Status = ExplStatusRejected
	} else {
		exp.Status = ExplStatusApproved
	}
	exp.ApprovedBy = actorID
	exp.ApprovedAt = &now
	exp.ApprovalNote = strings.TrimSpace(in.Comment)
	exp.UpdatedAt = now

	var actualMilli int64
	if in.ActualQtyMilli != nil && *in.ActualQtyMilli > 0 {
		actualMilli = *in.ActualQtyMilli
	} else if loadedActual, ok, aErr := s.repo.LoadActualMilli(ctx, entityID, exp.LocationID, exp.SKU, exp.WindowStart, exp.WindowEnd); aErr == nil && ok && loadedActual > 0 {
		actualMilli = loadedActual
	}
	if actualMilli > 0 {
		exp.ActualQtyMilli = actualMilli
		exp.ForecastValueAddBps = ComputeForecastValueAddBps(exp.BaselineP50Milli, exp.AdjustedP50Milli, actualMilli)
	}

	if err := s.repo.UpdateExplanation(ctx, exp); err != nil {
		return ForecastExplanation{}, err
	}
	return exp, nil
}
