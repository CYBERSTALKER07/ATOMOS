package forecastv2

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/civil"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestEvaluateSeriesBatch_ChallengerSeasonalBeatsClassicOnHolidaySeries(t *testing.T) {
	// Generate 45 days ending 2026-03-23 (during Navruz/Ramazon Hayit window in Uzbekistan calendar!)
	// where base demand is 10 units/day (10_000 milli), surging during Ramadan/Navruz + Payday.
	asOf := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)
	asOfDay := civil.DateOf(asOf)
	n := 45
	yMilli := make([]int64, n)
	dates := make([]civil.Date, n)
	start := asOfDay.AddDays(-(n - 1))
	for i := 0; i < n; i++ {
		d := start.AddDays(i)
		dates[i] = d
		regs := CalendarRegressorsOn(d, nil, 0, 0, false)
		mult := composeRegressorMultiplier(regs)
		yMilli[i] = int64(float64(10_000) * mult)
	}

	repo := NewMemoryRepository()
	repo.SeedSeries("sup-1", []SeriesHistoryInput{
		{
			LocationID:      "wh-1",
			LocationType:    "WAREHOUSE",
			SKU:             "sku-tea",
			DailyUnitsMilli: yMilli,
			Dates:           dates,
		},
	}, nil)

	svc := NewServiceWithRepository(repo)
	svc.SetNow(func() time.Time { return asOf })

	out, err := svc.RunForecast(context.Background(), RunForecastInput{
		EntityID:    "sup-1",
		HorizonDays: 7,
		AsOf:        asOf,
		TriggeredBy: "planner-1",
	})
	if err != nil {
		t.Fatalf("RunForecast failed: %v", err)
	}
	if out.Run.Status != StatusCompleted || out.Run.SkuCount != 1 || out.Run.LocationCount != 1 {
		t.Fatalf("unexpected run summary: %+v", out.Run)
	}
	if len(out.Forecasts) != 14 { // 7 days * 2 models (champion + challenger)
		t.Fatalf("expected 14 forecast rows (7d x 2 models), got %d", len(out.Forecasts))
	}
	for _, fv := range out.Forecasts {
		if fv.P10Milli > fv.P50Milli || fv.P50Milli > fv.P90Milli {
			t.Fatalf("invalid quantile ordering P10=%d P50=%d P90=%d", fv.P10Milli, fv.P50Milli, fv.P90Milli)
		}
		if fv.P50Milli <= 0 {
			t.Fatalf("expected positive P50Milli, got %d", fv.P50Milli)
		}
	}
	if out.Run.ChallengerWapeBps >= out.Run.ChampionWapeBps {
		t.Fatalf("expected challenger (%d bps) to beat classic (%d bps) on Uzbekistan holiday series", out.Run.ChallengerWapeBps, out.Run.ChampionWapeBps)
	}
	if out.Run.WinnerModel != ModelGlobalSeasonal {
		t.Fatalf("expected winner %s, got %s", ModelGlobalSeasonal, out.Run.WinnerModel)
	}
	if len(out.Explanations) != 1 {
		t.Fatalf("expected 1 structural explanation, got %d", len(out.Explanations))
	}
	exp := out.Explanations[0]
	if exp.Source != ExplSourceClassicStruct || len(exp.EvidenceRefs) < 2 {
		t.Fatalf("expected structural classic explanation with evidence citations, got %+v", exp)
	}
	if !strings.Contains(exp.SummaryText, "sku-tea") {
		t.Fatalf("expected explanation to reference SKU, got %q", exp.SummaryText)
	}
}

func TestCalendarRegressorsOn_SchoolStartAndPaydayAndPromoPrice(t *testing.T) {
	// Aug 25 is inside SchoolStart (Aug 20 - Sep 10), and Sep 2 is IndependenceDay + SchoolStart + Payday.
	day := civil.Date{Year: 2026, Month: time.September, Day: 2}
	regs := CalendarRegressorsOn(day, nil, 2000, -1000, true)
	seen := map[string]int64{}
	for _, r := range regs {
		seen[r.Name] = r.MultiplierBps
	}
	if seen["uz_independence_day"] != 12000 {
		t.Fatalf("expected uz_independence_day=12000, got %+v", seen)
	}
	if seen["SchoolStart"] != 12000 {
		t.Fatalf("expected SchoolStart=12000, got %+v", seen)
	}
	if seen["Payday"] != 11200 {
		t.Fatalf("expected Payday=11200, got %+v", seen)
	}
	if seen["Promo"] != 13000 {
		t.Fatalf("expected Promo=13000 (2000 bps * 1.5), got %+v", seen)
	}
	if seen["PriceElasticity"] != 11200 {
		t.Fatalf("expected PriceElasticity=11200 (-1000 bps * -1.2), got %+v", seen)
	}
}

func TestGeminiExplainer_FailClosedWhenRequiredAndSuccessWhenConfigured(t *testing.T) {
	eval := SeriesEvaluation{
		LocationID:        "wh-1",
		SKU:               "sku-cola",
		DemandClass:       "smooth",
		ChampionModel:     ModelClassicHoltWinters,
		ChallengerModel:   ModelGlobalSeasonal,
		WinnerModel:       ModelGlobalSeasonal,
		ChampionWapeBps:   1400,
		ChallengerWapeBps: 900,
		ObservedMeanMilli: 25_000,
		LastDayMilli:      30_000,
		ActiveRegressors: []RegressorSignal{
			{Name: "uz_navruz", Category: "HOLIDAY", MultiplierBps: 13500, Description: "Navruz 2026"},
		},
	}

	// Required + unconfigured -> fail closed.
	reqExplainer := NewGeminiExplainer(GeminiConfig{Required: true})
	if _, err := reqExplainer.Explain(context.Background(), ExplainInput{EntityID: "sup-1", RunID: "run-1", Evaluation: eval}); !errors.Is(err, ErrGeminiExplainerNotConfigured) {
		t.Fatalf("expected ErrGeminiExplainerNotConfigured, got %v", err)
	}

	// Configured Gemini server -> parses structured JSON explanation.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-goog-api-key") != "gem-key" {
			http.Error(w, "missing key", http.StatusUnauthorized)
			return
		}
		resp := map[string]any{
			"candidates": []map[string]any{
				{
					"content": map[string]any{
						"parts": []map[string]string{
							{"text": `{"summary_text":"Navruz uplift (+35%) supports +1500 bps adjustment for sku-cola (challenger WAPE 900 bps).","adjustment_bps":1500,"adjustment_reason":"navruz_holiday_surge"}`},
						},
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	liveExplainer := NewGeminiExplainer(GeminiConfig{
		Endpoint: srv.URL,
		APIKey:   "gem-key",
		Model:    "gemini-2.5-flash",
		Required: true,
	})
	out, err := liveExplainer.Explain(context.Background(), ExplainInput{EntityID: "sup-1", RunID: "run-1", Evaluation: eval})
	if err != nil {
		t.Fatalf("expected Gemini explainer success, got %v", err)
	}
	if out.Source != ExplSourceGemini || out.AdjustmentBps != 1500 || out.AdjustmentReason != "navruz_holiday_surge" {
		t.Fatalf("unexpected Gemini output: %+v", out)
	}
}

func TestApproveDemandAdjustment_SoD_AgentBan_And_FVA(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewServiceWithRepository(repo)

	// Propose a +2000 bps (+20%) adjustment over 10_000 milli baseline -> 12_000 milli adjusted.
	delta := int64(2000)
	_ = repo.SaveRunAndValues(context.Background(), ForecastRun{
		EntityID: "sup-1", RunID: "run-1", Status: StatusCompleted, CreatedAt: time.Now().UTC(),
	}, []ForecastValue{
		{
			EntityID: "sup-1", RunID: "run-1", ForecastDate: "2026-04-01",
			LocationID: "wh-1", SKU: "sku-1", Model: ModelGlobalSeasonal, IsWinner: true,
			DemandClass: "smooth", P10Milli: 8000, P50Milli: 10000, P90Milli: 12000, BaselineP50Milli: 10000,
		},
	}, nil)

	exp, err := svc.ProposeDemandAdjustment(context.Background(), ProposeAdjustmentInput{
		EntityID:         "sup-1",
		RunID:            "run-1",
		LocationID:       "wh-1",
		SKU:              "sku-1",
		AdjustmentBps:    &delta,
		AdjustmentReason: "promo_confirmed",
		ProposedBy:       "analyst-1",
	})
	if err != nil {
		t.Fatalf("ProposeDemandAdjustment: %v", err)
	}
	if exp.AdjustedP50Milli != 12000 {
		t.Fatalf("expected adjusted P50 12000, got %d", exp.AdjustedP50Milli)
	}

	// AGENT actor forbidden.
	if _, err := svc.ApproveDemandAdjustment(context.Background(), ApproveAdjustmentInput{
		EntityID: "sup-1", ExplanationID: exp.ExplanationID, ActorID: "agent-1", ActorType: auth.ActorTypeAgent,
	}); !errors.Is(err, ErrAgentForbidden) {
		t.Fatalf("expected ErrAgentForbidden, got %v", err)
	}

	// AUDITOR forbidden.
	if _, err := svc.ApproveDemandAdjustment(context.Background(), ApproveAdjustmentInput{
		EntityID: "sup-1", ExplanationID: exp.ExplanationID, ActorID: "aud-1", ActorRole: auth.SupplierRoleAuditor,
	}); !errors.Is(err, ErrAuditorReadOnly) {
		t.Fatalf("expected ErrAuditorReadOnly, got %v", err)
	}

	// Proposer cannot self-approve (SoD).
	if _, err := svc.ApproveDemandAdjustment(context.Background(), ApproveAdjustmentInput{
		EntityID: "sup-1", ExplanationID: exp.ExplanationID, ActorID: "analyst-1", ActorRole: auth.SupplierRoleOwner,
	}); !errors.Is(err, ErrSoDViolation) {
		t.Fatalf("expected ErrSoDViolation, got %v", err)
	}

	// Reviewer approves with actual = 12_000 milli:
	// Baseline err = |10000 - 12000| / 12000 = 1666 bps; Adjusted err = 0 bps -> FVA = +1666 bps!
	actual := int64(12000)
	approved, err := svc.ApproveDemandAdjustment(context.Background(), ApproveAdjustmentInput{
		EntityID:       "sup-1",
		ExplanationID:  exp.ExplanationID,
		ActorID:        "manager-1",
		ActorRole:      auth.SupplierRoleOwner,
		ActualQtyMilli: &actual,
		Comment:        "verified promo contract",
	})
	if err != nil {
		t.Fatalf("ApproveDemandAdjustment: %v", err)
	}
	if approved.Status != ExplStatusApproved || approved.ForecastValueAddBps != 1666 {
		t.Fatalf("expected APPROVED with FVA=1666 bps, got status=%s fva=%d", approved.Status, approved.ForecastValueAddBps)
	}

	// Re-approving returns ErrExplanationAlreadyDecided.
	if _, err := svc.ApproveDemandAdjustment(context.Background(), ApproveAdjustmentInput{
		EntityID: "sup-1", ExplanationID: exp.ExplanationID, ActorID: "manager-2",
	}); !errors.Is(err, ErrExplanationAlreadyDecided) {
		t.Fatalf("expected ErrExplanationAlreadyDecided, got %v", err)
	}
}
