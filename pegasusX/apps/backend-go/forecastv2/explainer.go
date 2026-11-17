package forecastv2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	defaultGeminiModel   = "gemini-2.5-flash"
	defaultGeminiTimeout = 10 * time.Second
)

// GeminiConfig configures Gemini explainability (with fail-closed toggle GEMINI_EXPLAINER_REQUIRED).
type GeminiConfig struct {
	Endpoint    string
	APIKey      string
	BearerToken string
	Model       string
	Required    bool
	HTTPClient  *http.Client
	Timeout     time.Duration
}

// GeminiConfigFromEnv loads Gemini explainer settings from environment variables.
func GeminiConfigFromEnv() GeminiConfig {
	model := strings.TrimSpace(os.Getenv("GEMINI_MODEL"))
	if model == "" {
		model = defaultGeminiModel
	}
	apiKey := strings.TrimSpace(os.Getenv("GEMINI_API_KEY"))
	bearer := strings.TrimSpace(os.Getenv("VERTEX_ACCESS_TOKEN"))
	endpoint := strings.TrimSpace(os.Getenv("GEMINI_ENDPOINT"))
	if endpoint == "" && (apiKey != "" || bearer != "") {
		endpoint = fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent", model)
	}
	reqRaw := strings.ToLower(strings.TrimSpace(os.Getenv("GEMINI_EXPLAINER_REQUIRED")))
	required := reqRaw == "1" || reqRaw == "true" || reqRaw == "yes" || reqRaw == "on"
	return GeminiConfig{
		Endpoint:    endpoint,
		APIKey:      apiKey,
		BearerToken: bearer,
		Model:       model,
		Required:    required,
		Timeout:     defaultGeminiTimeout,
	}
}

// GeminiExplainer generates evidence-cited demand explanations via Gemini HTTP API when configured,
// or falls back to a deterministic structural classic explainer locally unless GEMINI_EXPLAINER_REQUIRED=true.
type GeminiExplainer struct {
	cfg    GeminiConfig
	client *http.Client
}

// NewGeminiExplainer constructs a GeminiExplainer from cfg.
func NewGeminiExplainer(cfg GeminiConfig) *GeminiExplainer {
	if cfg.Model == "" {
		cfg.Model = defaultGeminiModel
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultGeminiTimeout
	}
	cli := cfg.HTTPClient
	if cli == nil {
		cli = &http.Client{Timeout: cfg.Timeout}
	}
	return &GeminiExplainer{cfg: cfg, client: cli}
}

// IsConfigured reports whether an external Gemini / Vertex endpoint and credential are present.
func (g *GeminiExplainer) IsConfigured() bool {
	if g == nil {
		return false
	}
	return strings.TrimSpace(g.cfg.Endpoint) != "" &&
		(strings.TrimSpace(g.cfg.APIKey) != "" || strings.TrimSpace(g.cfg.BearerToken) != "")
}

// ExplainInput carries observed real forecast statistics (never fabricated) for explanation generation.
type ExplainInput struct {
	EntityID       string
	RunID          string
	Evaluation     SeriesEvaluation
	Notes          string
	OverrideDelta  *int64
	OverrideReason string
}

// ExplainOutput is the structured result of explaining one series forecast.
type ExplainOutput struct {
	Source           string
	ModelName        string
	SummaryText      string
	EvidenceRefs     []EvidenceCitation
	AdjustmentBps    int64
	AdjustmentReason string
}

// Explain generates a grounded demand explanation + proposed adjustment bps.
func (g *GeminiExplainer) Explain(ctx context.Context, in ExplainInput) (ExplainOutput, error) {
	if g != nil && g.cfg.Required && !g.IsConfigured() {
		return ExplainOutput{}, ErrGeminiExplainerNotConfigured
	}
	if g != nil && g.IsConfigured() {
		out, err := g.callGemini(ctx, in)
		if err == nil {
			if in.OverrideDelta != nil {
				out.AdjustmentBps = clampAdjustmentBps(*in.OverrideDelta)
			}
			if strings.TrimSpace(in.OverrideReason) != "" {
				out.AdjustmentReason = strings.TrimSpace(in.OverrideReason)
			}
			return out, nil
		}
		if g.cfg.Required {
			return ExplainOutput{}, fmt.Errorf("%w: %v", ErrGeminiExplainerFailed, err)
		}
	}
	return buildClassicStructuralExplanation(in), nil
}

func (g *GeminiExplainer) callGemini(ctx context.Context, in ExplainInput) (ExplainOutput, error) {
	ev := in.Evaluation
	citations := buildObservedCitations(ev)
	evidencePayload := map[string]any{
		"entity_id":           in.EntityID,
		"run_id":              in.RunID,
		"location_id":         ev.LocationID,
		"sku":                 ev.SKU,
		"demand_class":        ev.DemandClass,
		"champion_model":      ev.ChampionModel,
		"challenger_model":    ev.ChallengerModel,
		"winner_model":        ev.WinnerModel,
		"champion_wape_bps":   ev.ChampionWapeBps,
		"challenger_wape_bps": ev.ChallengerWapeBps,
		"champion_bias_bps":   ev.ChampionBiasBps,
		"challenger_bias_bps": ev.ChallengerBiasBps,
		"observed_mean_milli": ev.ObservedMeanMilli,
		"active_regressors":   ev.ActiveRegressors,
		"citations":           citations,
		"notes":               strings.TrimSpace(in.Notes),
	}
	rawEvidence, _ := json.Marshal(evidencePayload)
	prompt := "You are an operations demand-planning analyst. Cite ONLY the exact metrics and ref_ids in the JSON evidence below (never invent numbers). Return strict JSON: {\"summary_text\":string,\"adjustment_bps\":integer,\"adjustment_reason\":string}.\nEVIDENCE:\n" + string(rawEvidence)

	reqBody := map[string]any{
		"contents": []map[string]any{
			{
				"role": "user",
				"parts": []map[string]string{
					{"text": prompt},
				},
			},
		},
		"generationConfig": map[string]any{
			"temperature":      0,
			"responseMimeType": "application/json",
		},
	}
	b, err := json.Marshal(reqBody)
	if err != nil {
		return ExplainOutput{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, g.cfg.Endpoint, bytes.NewReader(b))
	if err != nil {
		return ExplainOutput{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if g.cfg.APIKey != "" {
		httpReq.Header.Set("X-goog-api-key", g.cfg.APIKey)
	}
	if g.cfg.BearerToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+g.cfg.BearerToken)
	}

	resp, err := g.client.Do(httpReq)
	if err != nil {
		return ExplainOutput{}, err
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return ExplainOutput{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ExplainOutput{}, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}

	var parsed struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(bodyBytes, &parsed); err != nil {
		return ExplainOutput{}, err
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return ExplainOutput{}, fmt.Errorf("empty Gemini candidates")
	}
	text := strings.TrimSpace(parsed.Candidates[0].Content.Parts[0].Text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)

	var outJSON struct {
		SummaryText      string `json:"summary_text"`
		AdjustmentBps    int64  `json:"adjustment_bps"`
		AdjustmentReason string `json:"adjustment_reason"`
	}
	if err := json.Unmarshal([]byte(text), &outJSON); err != nil {
		return ExplainOutput{}, err
	}
	if strings.TrimSpace(outJSON.SummaryText) == "" {
		return ExplainOutput{}, fmt.Errorf("empty summary_text from Gemini")
	}
	return ExplainOutput{
		Source:           ExplSourceGemini,
		ModelName:        g.cfg.Model,
		SummaryText:      strings.TrimSpace(outJSON.SummaryText),
		EvidenceRefs:     citations,
		AdjustmentBps:    clampAdjustmentBps(outJSON.AdjustmentBps),
		AdjustmentReason: strings.TrimSpace(outJSON.AdjustmentReason),
	}, nil
}

func buildObservedCitations(ev SeriesEvaluation) []EvidenceCitation {
	citations := []EvidenceCitation{
		{
			RefType:   "WAPE_BACKTEST",
			RefID:     ev.SKU + ":" + ev.LocationID,
			Label:     "Champion vs Challenger WAPE",
			ValueText: fmt.Sprintf("winner=%s classic_wape_bps=%d challenger_wape_bps=%d", ev.WinnerModel, ev.ChampionWapeBps, ev.ChallengerWapeBps),
		},
		{
			RefType:   "SERIES",
			RefID:     ev.SKU + ":" + ev.LocationID,
			Label:     "Observed demand baseline",
			ValueText: fmt.Sprintf("class=%s mean_milli=%d last_day_milli=%d", ev.DemandClass, ev.ObservedMeanMilli, ev.LastDayMilli),
		},
	}
	for _, rg := range ev.ActiveRegressors {
		citations = append(citations, EvidenceCitation{
			RefType:   rg.Category,
			RefID:     rg.Name,
			Label:     rg.Description,
			ValueText: fmt.Sprintf("multiplier_bps=%d", rg.MultiplierBps),
		})
	}
	return citations
}

func buildClassicStructuralExplanation(in ExplainInput) ExplainOutput {
	ev := in.Evaluation
	citations := buildObservedCitations(ev)

	adjBps := int64(0)
	reason := "no_adjustment_needed"
	if in.OverrideDelta != nil {
		adjBps = clampAdjustmentBps(*in.OverrideDelta)
		reason = strings.TrimSpace(in.OverrideReason)
		if reason == "" {
			reason = "manual_adjustment_proposal"
		}
	} else {
		// Derive evidence-backed adjustment from bias or active holiday/promo regressors.
		winnerBias := ev.ChampionBiasBps
		if ev.WinnerModel == ModelGlobalSeasonal {
			winnerBias = ev.ChallengerBiasBps
		}
		if winnerBias > 500 || winnerBias < -500 {
			adjBps = clampAdjustmentBps(-winnerBias / 2)
			reason = "bias_correction"
		} else if len(ev.ActiveRegressors) > 0 {
			maxLift := int64(0)
			topName := ""
			for _, rg := range ev.ActiveRegressors {
				delta := rg.MultiplierBps - BpsScale
				if absInt64(delta) > absInt64(maxLift) {
					maxLift = delta
					topName = rg.Name
				}
			}
			if maxLift != 0 {
				adjBps = clampAdjustmentBps(maxLift / 4)
				reason = "regressor_" + strings.ToLower(topName)
			}
		}
	}

	regNames := make([]string, 0, len(ev.ActiveRegressors))
	for _, rg := range ev.ActiveRegressors {
		regNames = append(regNames, fmt.Sprintf("%s (%dbps)", rg.Name, rg.MultiplierBps))
	}
	regSummary := "none"
	if len(regNames) > 0 {
		regSummary = strings.Join(regNames, ", ")
	}
	summary := fmt.Sprintf(
		"SKU %s at %s (%s class): winner %s (classic WAPE %d bps vs challenger WAPE %d bps, observed mean %d milli-units/day). Active regressors: %s. Proposed adjustment: %d bps (%s).",
		ev.SKU, ev.LocationID, ev.DemandClass, ev.WinnerModel,
		ev.ChampionWapeBps, ev.ChallengerWapeBps, ev.ObservedMeanMilli,
		regSummary, adjBps, reason,
	)

	return ExplainOutput{
		Source:           ExplSourceClassicStruct,
		ModelName:        "structural_evidence_v1",
		SummaryText:      summary,
		EvidenceRefs:     citations,
		AdjustmentBps:    adjBps,
		AdjustmentReason: reason,
	}
}

func clampAdjustmentBps(bps int64) int64 {
	if bps < -5000 {
		return -5000
	}
	if bps > 10000 {
		return 10000
	}
	return bps
}

func absInt64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

// ComputeForecastValueAddBps measures Forecast Value Added in bps:
// FVA = BaselineWapeBps - AdjustedWapeBps (>0 means adjustment reduced forecast error).
func ComputeForecastValueAddBps(baselineP50Milli, adjustedP50Milli, actualQtyMilli int64) int64 {
	if actualQtyMilli <= 0 {
		return 0
	}
	baseErr := absInt64(baselineP50Milli - actualQtyMilli)
	adjErr := absInt64(adjustedP50Milli - actualQtyMilli)
	baseWapeBps := (baseErr * BpsScale) / actualQtyMilli
	adjWapeBps := (adjErr * BpsScale) / actualQtyMilli
	return baseWapeBps - adjWapeBps
}
