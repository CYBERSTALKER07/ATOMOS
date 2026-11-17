package forecastv2

import (
	"encoding/json"
	"math"
	"sort"
	"time"

	"cloud.google.com/go/civil"
	"github.com/pegasusx/pegasusx/apps/backend-go/planning/forecast"
	"github.com/pegasusx/pegasusx/apps/backend-go/seasonalcore"
)

// SeriesEvaluation holds the full champion-vs-challenger evaluation and horizon outputs for one SKU/location.
type SeriesEvaluation struct {
	LocationID        string
	LocationType      string
	SKU               string
	ProductID         string
	DemandClass       string
	ChampionModel     string
	ChallengerModel   string
	WinnerModel       string
	ChampionWapeBps   int64
	ChallengerWapeBps int64
	ChampionBiasBps   int64
	ChallengerBiasBps int64
	ConfidenceBps     int64
	ClassicValues     []ForecastValue
	ChallengerValues  []ForecastValue
	ActiveRegressors  []RegressorSignal
	ObservedMeanMilli int64
	LastDayMilli      int64
}

// EvaluateSeriesBatch runs Champion (classic planning/forecast) vs Challenger (global seasonal/calendar)
// across all supplied SKU/location series, pooling cross-series day-of-week patterns for the global model.
func EvaluateSeriesBatch(
	entityID, runID string,
	asOf time.Time,
	horizonDays int,
	series []SeriesHistoryInput,
	overrides []SeasonalWindow,
	globalPromoDiscountBps, globalPriceChangeBps int64,
	globalActivePromo bool,
) ([]SeriesEvaluation, error) {
	if horizonDays <= 0 {
		horizonDays = DefaultHorizonDays
	}
	if horizonDays > MaxHorizonDays {
		horizonDays = MaxHorizonDays
	}
	if len(series) == 0 {
		return nil, nil
	}
	asOfDay := civil.DateOf(asOf.UTC())
	globalDOW := computeGlobalDOWMultipliers(series, asOfDay)

	out := make([]SeriesEvaluation, 0, len(series))
	for _, s := range series {
		if s.SKU == "" {
			continue
		}
		locID := s.LocationID
		if locID == "" {
			locID = "ALL"
		}
		prodID := s.ProductID
		if prodID == "" {
			prodID = s.SKU
		}
		promoDisc := s.PromoDiscountBps
		if promoDisc == 0 {
			promoDisc = globalPromoDiscountBps
		}
		priceChg := s.PriceChangeBps
		if priceChg == 0 {
			priceChg = globalPriceChangeBps
		}
		activePromo := s.ActivePromo || globalActivePromo || promoDisc > 0

		dates := s.Dates
		yMilli := s.DailyUnitsMilli
		if len(dates) != len(yMilli) {
			dates = make([]civil.Date, len(yMilli))
			start := asOfDay.AddDays(-(len(yMilli) - 1))
			for i := range yMilli {
				dates[i] = start.AddDays(i)
			}
		}

		eval := evaluateSingleSeries(
			entityID, runID, asOfDay, horizonDays,
			locID, s.LocationType, s.SKU, prodID,
			dates, yMilli, overrides, globalDOW,
			promoDisc, priceChg, activePromo,
		)
		out = append(out, eval)
	}
	return out, nil
}

func computeGlobalDOWMultipliers(series []SeriesHistoryInput, asOfDay civil.Date) [7]float64 {
	var sums [7]float64
	var counts [7]float64
	for _, s := range series {
		if len(s.DailyUnitsMilli) == 0 {
			continue
		}
		mean := 0.0
		for _, v := range s.DailyUnitsMilli {
			if v > 0 {
				mean += float64(v)
			}
		}
		mean /= float64(len(s.DailyUnitsMilli))
		if mean <= 0 {
			continue
		}
		dates := s.Dates
		start := asOfDay.AddDays(-(len(s.DailyUnitsMilli) - 1))
		for i, v := range s.DailyUnitsMilli {
			if v < 0 {
				v = 0
			}
			d := start.AddDays(i)
			if len(dates) == len(s.DailyUnitsMilli) {
				d = dates[i]
			}
			wd := int(d.In(time.UTC).Weekday())
			sums[wd] += (float64(v) / mean)
			counts[wd] += 1.0
		}
	}
	var out [7]float64
	for wd := 0; wd < 7; wd++ {
		if counts[wd] > 0 {
			r := sums[wd] / counts[wd]
			if r < 0.5 {
				r = 0.5
			}
			if r > 1.8 {
				r = 1.8
			}
			out[wd] = r
		} else {
			out[wd] = 1.0
		}
	}
	return out
}

func evaluateSingleSeries(
	entityID, runID string,
	asOfDay civil.Date,
	horizonDays int,
	locID, locType, sku, prodID string,
	dates []civil.Date,
	yMilli []int64,
	overrides []SeasonalWindow,
	globalDOW [7]float64,
	promoDiscBps, priceChgBps int64,
	activePromo bool,
) SeriesEvaluation {
	y := make([]float64, len(yMilli))
	var sumMilli, lastMilli int64
	for i, v := range yMilli {
		if v < 0 {
			v = 0
		}
		y[i] = float64(v) / float64(MilliScale)
		sumMilli += v
		lastMilli = v
	}
	var obsMeanMilli int64
	if len(yMilli) > 0 {
		obsMeanMilli = sumMilli / int64(len(yMilli))
	}

	// 1. Classic Champion via planning/forecast primitives.
	classicRes := forecast.ForecastSeries(y)
	championModel := mapClassicModelName(classicRes.BaselineSource)
	demandClass := classicRes.Class.String()

	// 2. Backtest Champion vs Challenger over rolling holdout.
	holdout := len(y) / 5
	if holdout > 14 {
		holdout = 14
	}
	if holdout < 7 && len(y) >= 14 {
		holdout = 7
	}
	if holdout >= len(y)/2 {
		holdout = len(y) / 3
	}
	if holdout < 2 && len(y) >= 4 {
		holdout = 2
	}

	var champWapeBps, champBiasBps int64
	var chalWapeBps, chalBiasBps int64
	var chalResiduals []float64

	if holdout >= 2 && len(y) > holdout+2 {
		trainEnd := len(y) - holdout
		actualSlice := y[trainEnd:]
		champPreds := make([]float64, holdout)
		chalPreds := make([]float64, holdout)
		for i := 0; i < holdout; i++ {
			t := trainEnd + i
			prefixY := y[:t]
			prefixDates := dates[:t]
			champPreds[i] = oneStepClassic(prefixY, classicRes.BaselineSource, classicRes.Alpha, classicRes.Beta, classicRes.Gamma)
			chalPreds[i], _ = oneStepChallenger(prefixY, prefixDates, dates[t], overrides, globalDOW, promoDiscBps, priceChgBps, activePromo)
			chalResiduals = append(chalResiduals, y[t]-chalPreds[i])
		}
		cw, cb := forecast.WAPEBias(champPreds, actualSlice)
		chw, chb := forecast.WAPEBias(chalPreds, actualSlice)
		champWapeBps = toBps(cw)
		champBiasBps = toBps(cb)
		chalWapeBps = toBps(chw)
		chalBiasBps = toBps(chb)
	} else if len(y) >= 2 {
		// Short series in-sample residual error check.
		chalResiduals = make([]float64, 0, len(y)-1)
		actualSlice := make([]float64, 0, len(y)-1)
		chalPreds := make([]float64, 0, len(y)-1)
		champPreds := make([]float64, 0, len(y)-1)
		for t := 1; t < len(y); t++ {
			cp := oneStepClassic(y[:t], classicRes.BaselineSource, classicRes.Alpha, classicRes.Beta, classicRes.Gamma)
			chp, _ := oneStepChallenger(y[:t], dates[:t], dates[t], overrides, globalDOW, promoDiscBps, priceChgBps, activePromo)
			champPreds = append(champPreds, cp)
			chalPreds = append(chalPreds, chp)
			actualSlice = append(actualSlice, y[t])
			chalResiduals = append(chalResiduals, y[t]-chp)
		}
		cw, cb := forecast.WAPEBias(champPreds, actualSlice)
		chw, chb := forecast.WAPEBias(chalPreds, actualSlice)
		champWapeBps = toBps(cw)
		champBiasBps = toBps(cb)
		chalWapeBps = toBps(chw)
		chalBiasBps = toBps(chb)
	}

	winnerModel := championModel
	winnerWapeBps := champWapeBps
	if chalWapeBps < champWapeBps && (len(y) >= 4) {
		winnerModel = ModelGlobalSeasonal
		winnerWapeBps = chalWapeBps
	}

	confBps := BpsScale - winnerWapeBps
	if confBps < 1000 {
		confBps = 1000
	}
	if confBps > 9800 {
		confBps = 9800
	}
	if len(y) < 14 && confBps > 6500 {
		confBps = 6500
	}

	// 3. Project horizon h = 1..horizonDays for both models.
	classicVals := make([]ForecastValue, 0, horizonDays)
	challengerVals := make([]ForecastValue, 0, horizonDays)
	seenReg := map[string]RegressorSignal{}
	var allRegs []RegressorSignal

	for h := 1; h <= horizonDays; h++ {
		fDay := asOfDay.AddDays(h)
		fDateStr := fDay.String()

		classicPoint := horizonClassic(y, classicRes.BaselineSource, classicRes.Alpha, classicRes.Beta, classicRes.Gamma, h)
		if classicPoint < 0 {
			classicPoint = 0
		}
		cp10, cp50, cp90 := quantileMilliFromResiduals(classicPoint, classicRes.Residuals)

		chalPoint, regs := oneStepChallenger(y, dates, fDay, overrides, globalDOW, promoDiscBps, priceChgBps, activePromo)
		if chalPoint < 0 {
			chalPoint = 0
		}
		chp10, chp50, chp90 := quantileMilliFromResiduals(chalPoint, chalResiduals)

		for _, rg := range regs {
			if _, ok := seenReg[rg.Name]; !ok {
				seenReg[rg.Name] = rg
				allRegs = append(allRegs, rg)
			}
		}
		regBytes, _ := json.Marshal(regs)

		classicVals = append(classicVals, ForecastValue{
			EntityID:         entityID,
			RunID:            runID,
			ForecastDate:     fDateStr,
			LocationID:       locID,
			LocationType:     locType,
			SKU:              sku,
			ProductID:        prodID,
			Model:            championModel,
			IsWinner:         winnerModel == championModel,
			DemandClass:      demandClass,
			P10Milli:         cp10,
			P50Milli:         cp50,
			P90Milli:         cp90,
			BaselineP50Milli: cp50,
			WapeBps:          champWapeBps,
			BiasBps:          champBiasBps,
			ConfidenceBps:    confBps,
			Regressors:       nil,
			RegressorsJSON:   "[]",
		})

		challengerVals = append(challengerVals, ForecastValue{
			EntityID:         entityID,
			RunID:            runID,
			ForecastDate:     fDateStr,
			LocationID:       locID,
			LocationType:     locType,
			SKU:              sku,
			ProductID:        prodID,
			Model:            ModelGlobalSeasonal,
			IsWinner:         winnerModel == ModelGlobalSeasonal,
			DemandClass:      demandClass,
			P10Milli:         chp10,
			P50Milli:         chp50,
			P90Milli:         chp90,
			BaselineP50Milli: cp50,
			WapeBps:          chalWapeBps,
			BiasBps:          chalBiasBps,
			ConfidenceBps:    confBps,
			Regressors:       regs,
			RegressorsJSON:   string(regBytes),
		})
	}

	sort.Slice(allRegs, func(i, j int) bool {
		return allRegs[i].Name < allRegs[j].Name
	})

	return SeriesEvaluation{
		LocationID:        locID,
		LocationType:      locType,
		SKU:               sku,
		ProductID:         prodID,
		DemandClass:       demandClass,
		ChampionModel:     championModel,
		ChallengerModel:   ModelGlobalSeasonal,
		WinnerModel:       winnerModel,
		ChampionWapeBps:   champWapeBps,
		ChallengerWapeBps: chalWapeBps,
		ChampionBiasBps:   champBiasBps,
		ChallengerBiasBps: chalBiasBps,
		ConfidenceBps:     confBps,
		ClassicValues:     classicVals,
		ChallengerValues:  challengerVals,
		ActiveRegressors:  allRegs,
		ObservedMeanMilli: obsMeanMilli,
		LastDayMilli:      lastMilli,
	}
}

func mapClassicModelName(src string) string {
	switch src {
	case "holt_winters":
		return ModelClassicHoltWinters
	case "croston":
		return ModelClassicCroston
	default:
		return ModelClassicSES
	}
}

func oneStepClassic(y []float64, src string, alpha, beta, gamma float64) float64 {
	if len(y) == 0 {
		return 0
	}
	switch src {
	case "holt_winters":
		f, _ := forecast.FitHoltWinters(y, alpha, beta, gamma, 1)
		return math.Max(0, f)
	case "croston":
		f, _ := forecast.FitCrostonSBA(y, alpha)
		return math.Max(0, f)
	default:
		f, _ := forecast.FitSES(y, alpha)
		return math.Max(0, f)
	}
}

func horizonClassic(y []float64, src string, alpha, beta, gamma float64, horizon int) float64 {
	if len(y) == 0 {
		return 0
	}
	if src == "holt_winters" {
		f, _ := forecast.FitHoltWinters(y, alpha, beta, gamma, horizon)
		return math.Max(0, f)
	}
	return oneStepClassic(y, src, alpha, beta, gamma)
}

func oneStepChallenger(
	y []float64,
	dates []civil.Date,
	targetDay civil.Date,
	overrides []SeasonalWindow,
	globalDOW [7]float64,
	promoDiscBps, priceChgBps int64,
	activePromo bool,
) (float64, []RegressorSignal) {
	if len(y) == 0 {
		return 0, nil
	}
	// Deseasonalize historical series by historical calendar + global DOW factors.
	deseasonal := make([]float64, len(y))
	for i := range y {
		mult := 1.0
		if i < len(dates) {
			histRegs := CalendarRegressorsOn(dates[i], overrides, 0, 0, false)
			histCalMult := composeRegressorMultiplier(histRegs)
			wd := int(dates[i].In(time.UTC).Weekday())
			dow := globalDOW[wd]
			if dow <= 0 {
				dow = 1.0
			}
			mult = histCalMult * dow
		}
		if mult < 0.25 {
			mult = 0.25
		}
		deseasonal[i] = y[i] / mult
	}
	baseLevel, _ := forecast.FitSES(deseasonal, 0.25)
	if baseLevel < 0 {
		baseLevel = 0
	}

	targetRegs := CalendarRegressorsOn(targetDay, overrides, promoDiscBps, priceChgBps, activePromo)
	wd := int(targetDay.In(time.UTC).Weekday())
	dowMult := globalDOW[wd]
	if dowMult <= 0 {
		dowMult = 1.0
	}
	if math.Abs(dowMult-1.0) >= 0.03 {
		targetRegs = append(targetRegs, RegressorSignal{
			Name:          targetDay.In(time.UTC).Weekday().String(),
			Category:      "CALENDAR",
			MultiplierBps: int64(math.Round(dowMult * float64(BpsScale))),
			Description:   "Cross-series day-of-week pattern",
		})
	}
	calMult := composeRegressorMultiplier(targetRegs)
	pred := baseLevel * calMult
	if pred < 0 {
		pred = 0
	}
	return pred, targetRegs
}

// CalendarRegressorsOn returns active Uzbekistan calendar, payday, school-start,
// promo, and price regressors for targetDay.
func CalendarRegressorsOn(
	targetDay civil.Date,
	overrides []SeasonalWindow,
	promoDiscBps, priceChgBps int64,
	activePromo bool,
) []RegressorSignal {
	ts := targetDay.In(time.UTC)
	var regs []RegressorSignal

	// 1. Uzbekistan official holiday calendar (Navruz, Ramadan, Ramazon/Qurbon Hayit, Independence Day).
	for _, h := range seasonalcore.ActiveUzbekistanHolidaysOn(ts) {
		mbps := int64(math.Round(seasonalcore.ClampMultiplier(h.Multiplier) * float64(BpsScale)))
		regs = append(regs, RegressorSignal{
			Name:          h.TemplateID,
			Category:      "HOLIDAY",
			MultiplierBps: mbps,
			Description:   h.Name,
		})
	}

	// 2. Supplier custom seasonal overrides.
	for _, ov := range overrides {
		if !targetDay.Before(ov.StartDate) && !targetDay.After(ov.EndDate) && ov.MultiplierBps > 0 {
			name := ov.Name
			if name == "" {
				name = ov.TemplateID
			}
			regs = append(regs, RegressorSignal{
				Name:          name,
				Category:      "SEASONAL_OVERRIDE",
				MultiplierBps: ov.MultiplierBps,
				Description:   "Supplier seasonal window",
			})
		}
	}

	// 3. SchoolStart window (Aug 20 – Sep 10 inclusive).
	m, d := targetDay.Month, targetDay.Day
	if (m == time.August && d >= 20) || (m == time.September && d <= 10) {
		regs = append(regs, RegressorSignal{
			Name:          "SchoolStart",
			Category:      "CALENDAR",
			MultiplierBps: 12000,
			Description:   "Back-to-school seasonal uplift (Aug 20 - Sep 10)",
		})
	}

	// 4. Payday cycle (1st-5th and 15th-20th of each month).
	if (d >= 1 && d <= 5) || (d >= 15 && d <= 20) {
		regs = append(regs, RegressorSignal{
			Name:          "Payday",
			Category:      "PAYDAY",
			MultiplierBps: 11200,
			Description:   "Semi-monthly salary payday demand lift",
		})
	}

	// 5. Promo regressor.
	if activePromo || promoDiscBps > 0 {
		promoMultBps := int64(11500)
		if promoDiscBps > 0 {
			promoMultBps = 10000 + (promoDiscBps*3)/2
			if promoMultBps > 16000 {
				promoMultBps = 16000
			}
		}
		regs = append(regs, RegressorSignal{
			Name:          "Promo",
			Category:      "PROMO",
			MultiplierBps: promoMultBps,
			Description:   "Active trade promotion uplift",
		})
	}

	// 6. Price change elasticity regressor (elasticity ~ -1.2).
	if priceChgBps != 0 {
		priceMultBps := 10000 - (priceChgBps*12)/10
		if priceMultBps < 5000 {
			priceMultBps = 5000
		}
		if priceMultBps > 15000 {
			priceMultBps = 15000
		}
		regs = append(regs, RegressorSignal{
			Name:          "PriceElasticity",
			Category:      "PRICE",
			MultiplierBps: priceMultBps,
			Description:   "Price change demand elasticity adjustment",
		})
	}

	return regs
}

func composeRegressorMultiplier(regs []RegressorSignal) float64 {
	if len(regs) == 0 {
		return 1.0
	}
	mult := 1.0
	for _, r := range regs {
		if r.MultiplierBps > 0 {
			mult *= float64(r.MultiplierBps) / float64(BpsScale)
		}
	}
	return seasonalcore.ClampMultiplier(mult)
}

func quantileMilliFromResiduals(point float64, residuals []float64) (p10, p50, p90 int64) {
	if point < 0 {
		point = 0
	}
	p50 = int64(math.Round(point * float64(MilliScale)))
	if len(residuals) < 3 {
		spread := int64(math.Round(point * 0.15 * float64(MilliScale)))
		if spread < 100 && p50 > 0 {
			spread = 100
		}
		p10 = p50 - spread
		if p10 < 0 {
			p10 = 0
		}
		p90 = p50 + spread
		return p10, p50, p90
	}
	sorted := append([]float64(nil), residuals...)
	sort.Float64s(sorted)
	q10 := interpPercentile(sorted, 0.10)
	q90 := interpPercentile(sorted, 0.90)
	low := point + q10
	high := point + q90
	if low < 0 {
		low = 0
	}
	p10 = int64(math.Round(low * float64(MilliScale)))
	p90 = int64(math.Round(high * float64(MilliScale)))
	if p10 > p50 {
		p10 = p50
	}
	if p90 < p50 {
		p90 = p50
	}
	return p10, p50, p90
}

func interpPercentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 1 {
		return sorted[len(sorted)-1]
	}
	idx := p * float64(len(sorted)-1)
	lo := int(math.Floor(idx))
	hi := int(math.Ceil(idx))
	if lo == hi {
		return sorted[lo]
	}
	w := idx - float64(lo)
	return sorted[lo]*(1-w) + sorted[hi]*w
}

func toBps(ratio float64) int64 {
	if math.IsNaN(ratio) || math.IsInf(ratio, 0) {
		return 0
	}
	return int64(math.Round(ratio * float64(BpsScale)))
}
