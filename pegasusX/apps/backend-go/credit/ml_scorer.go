package credit

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// MLFeatures is the canonical integer-only credit feature vector hashed into FeaturesHash.
type MLFeatures struct {
	RetailerID            string `json:"retailer_id"`
	SupplierID            string `json:"supplier_id,omitempty"`
	Scope                 string `json:"scope"`
	CreditLimitMinor      int64  `json:"credit_limit_minor"`
	CurrentBalanceMinor   int64  `json:"current_balance_minor"`
	ReservedMinor         int64  `json:"reserved_minor"`
	UtilizationBps        int64  `json:"utilization_bps"`
	DelinquencyCount      int64  `json:"delinquency_count"`
	MaxDaysPastDue        int64  `json:"max_days_past_due"`
	PaymentsLast90d       int64  `json:"payments_last_90d"`
	ExpectedPayments90d   int64  `json:"expected_payments_90d"`
	PaidVolume90dMinor    int64  `json:"paid_volume_90d_minor"`
	AvgOrderMinor         int64  `json:"avg_order_minor"`
	OrderCount90d         int64  `json:"order_count_90d"`
	DisputedInvoicesCount int64  `json:"disputed_invoices_count"`
	TenureDays            int64  `json:"tenure_days"`
	ConsentingSuppliers   int64  `json:"consenting_suppliers"`
	ProfileStatus         string `json:"profile_status"`
}

// ComputeFeaturesHash computes deterministic SHA-256 hex digest of f.
func ComputeFeaturesHash(f MLFeatures) string {
	b, _ := json.Marshal(f)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// MLScorerRequest is sent to CREDIT_ML_SCORER_URL.
type MLScorerRequest struct {
	RetailerID   string     `json:"retailer_id"`
	SupplierID   string     `json:"supplier_id,omitempty"`
	Scope        string     `json:"scope"`
	Features     MLFeatures `json:"features"`
	FeaturesHash string     `json:"features_hash"`
}

// MLScorerResponse is returned by MLScorer implementations (zero floats).
type MLScorerResponse struct {
	ModelVersion string       `json:"model_version"`
	ScorerSource string       `json:"scorer_source"`
	Score        int64        `json:"score"`
	PdBps        int64        `json:"pd_bps"`
	ReasonCodes  []ReasonCode `json:"reason_codes"`
	FeaturesHash string       `json:"features_hash"`
}

// MLScorer scores a retailer credit feature vector.
type MLScorer interface {
	Score(ctx context.Context, req MLScorerRequest) (MLScorerResponse, error)
}

// MLScorerConfig configures the external credit ML scorer HTTP client.
type MLScorerConfig struct {
	URL        string
	Required   bool
	APIKey     string
	Timeout    time.Duration
	HTTPClient *http.Client
}

// MLScorerConfigFromEnv builds MLScorerConfig from environment variables.
func MLScorerConfigFromEnv() MLScorerConfig {
	reqRaw := strings.ToLower(strings.TrimSpace(os.Getenv("CREDIT_ML_SCORER_REQUIRED")))
	required := reqRaw == "1" || reqRaw == "true" || reqRaw == "yes" || reqRaw == "on"
	timeout := 3 * time.Second
	if raw := strings.TrimSpace(os.Getenv("CREDIT_ML_SCORER_TIMEOUT_MS")); raw != "" {
		if ms, err := strconv.Atoi(raw); err == nil && ms > 0 {
			timeout = time.Duration(ms) * time.Millisecond
		}
	}
	return MLScorerConfig{
		URL:      strings.TrimSpace(os.Getenv("CREDIT_ML_SCORER_URL")),
		Required: required,
		APIKey:   strings.TrimSpace(os.Getenv("INTERNAL_API_KEY")),
		Timeout:  timeout,
	}
}

// MLScorerClient calls the configured ML scorer service over HTTP JSON when CREDIT_ML_SCORER_URL is set,
// fails closed when CREDIT_ML_SCORER_REQUIRED=true and unconfigured/failing, and otherwise returns
// calibrated scorecard PD with FeaturesHash.
type MLScorerClient struct {
	cfg    MLScorerConfig
	client *http.Client
}

// NewHTTPMLScorerClient builds an MLScorerClient from cfg.
func NewHTTPMLScorerClient(cfg MLScorerConfig) *MLScorerClient {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 3 * time.Second
	}
	cli := cfg.HTTPClient
	if cli == nil {
		cli = &http.Client{Timeout: cfg.Timeout}
	}
	return &MLScorerClient{cfg: cfg, client: cli}
}

// NewHTTPMLScorerClientFromEnv builds an MLScorerClient from environment variables.
func NewHTTPMLScorerClientFromEnv() *MLScorerClient {
	return NewHTTPMLScorerClient(MLScorerConfigFromEnv())
}

// IsConfigured reports whether CREDIT_ML_SCORER_URL is set.
func (c *MLScorerClient) IsConfigured() bool {
	return c != nil && strings.TrimSpace(c.cfg.URL) != ""
}

// Score scores req via CREDIT_ML_SCORER_URL or fails closed if required and unconfigured.
func (c *MLScorerClient) Score(ctx context.Context, req MLScorerRequest) (MLScorerResponse, error) {
	if req.FeaturesHash == "" {
		req.FeaturesHash = ComputeFeaturesHash(req.Features)
	}
	if c == nil || !c.IsConfigured() {
		if c != nil && c.cfg.Required {
			return MLScorerResponse{}, ErrMLScorerNotConfigured
		}
		return MLScorerResponse{
			ScorerSource: ScorerSourceScorecard,
			ModelVersion: "scorecard_v2",
			FeaturesHash: req.FeaturesHash,
		}, nil
	}

	endpoint := resolveMLScorerEndpoint(c.cfg.URL)
	payload, err := json.Marshal(req)
	if err != nil {
		return MLScorerResponse{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return MLScorerResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.cfg.APIKey != "" {
		httpReq.Header.Set("X-Internal-Api-Key", c.cfg.APIKey)
	}

	resp, err := c.client.Do(httpReq)
	if err != nil {
		if c.cfg.Required {
			return MLScorerResponse{}, fmt.Errorf("%w: %v", ErrMLScorerFailed, err)
		}
		return MLScorerResponse{
			ScorerSource: ScorerSourceScorecard,
			ModelVersion: "scorecard_v2",
			FeaturesHash: req.FeaturesHash,
			ReasonCodes: []ReasonCode{{
				Code: "ML_SIDECAR_UNAVAILABLE_FALLBACK", Label: "ML scorer unreachable; used deterministic scorecard",
				Detail: err.Error(), Source: "SCORECARD",
			}},
		}, nil
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		if c.cfg.Required {
			return MLScorerResponse{}, fmt.Errorf("%w: %v", ErrMLScorerFailed, err)
		}
		return MLScorerResponse{ScorerSource: ScorerSourceScorecard, ModelVersion: "scorecard_v2", FeaturesHash: req.FeaturesHash}, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if c.cfg.Required {
			return MLScorerResponse{}, fmt.Errorf("%w: HTTP %d", ErrMLScorerFailed, resp.StatusCode)
		}
		return MLScorerResponse{
			ScorerSource: ScorerSourceScorecard,
			ModelVersion: "scorecard_v2",
			FeaturesHash: req.FeaturesHash,
			ReasonCodes: []ReasonCode{{
				Code: "ML_SIDECAR_ERROR_FALLBACK", Label: "ML scorer returned non-2xx; used scorecard",
				Detail: fmt.Sprintf("http_status=%d", resp.StatusCode), Source: "SCORECARD",
			}},
		}, nil
	}

	parsed, err := decodeMLScorerJSON(rawBody, req.FeaturesHash)
	if err != nil {
		if c.cfg.Required {
			return MLScorerResponse{}, fmt.Errorf("%w: %v", ErrMLScorerFailed, err)
		}
		return MLScorerResponse{ScorerSource: ScorerSourceScorecard, ModelVersion: "scorecard_v2", FeaturesHash: req.FeaturesHash}, nil
	}
	return parsed, nil
}

func resolveMLScorerEndpoint(raw string) string {
	trimmed := strings.TrimSpace(raw)
	u, err := url.Parse(trimmed)
	if err != nil || u.Path == "" || u.Path == "/" {
		return strings.TrimRight(trimmed, "/") + "/score"
	}
	return trimmed
}

func decodeMLScorerJSON(raw []byte, defaultHash string) (MLScorerResponse, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var payload struct {
		ModelVersion string          `json:"model_version"`
		Score        json.Number     `json:"score"`
		PdBps        json.Number     `json:"pd_bps"`
		PD           json.Number     `json:"pd"`
		ReasonCodes  json.RawMessage `json:"reason_codes"`
		FeaturesHash string          `json:"features_hash"`
	}
	if err := dec.Decode(&payload); err != nil {
		return MLScorerResponse{}, err
	}
	pdBps := int64(-1)
	if payload.PdBps != "" {
		if v, err := parseDecimalScaled(payload.PdBps.String(), 1); err == nil {
			pdBps = v
		}
	}
	if pdBps < 0 && payload.PD != "" {
		if v, err := parseDecimalScaled(payload.PD.String(), 10000); err == nil {
			pdBps = v
		}
	}
	if pdBps < 0 {
		return MLScorerResponse{}, fmt.Errorf("missing pd_bps or pd in ML scorer response")
	}
	if pdBps > 10000 {
		pdBps = 10000
	}

	score := int64(-1)
	if payload.Score != "" {
		if v, err := parseDecimalScaled(payload.Score.String(), 1); err == nil {
			score = v
		}
	}
	if score < 0 {
		score = 100 - pdBps/100
	}
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	var reasons []ReasonCode
	if len(payload.ReasonCodes) > 0 {
		if err := json.Unmarshal(payload.ReasonCodes, &reasons); err != nil {
			var strs []string
			if err2 := json.Unmarshal(payload.ReasonCodes, &strs); err2 == nil {
				for _, code := range strs {
					reasons = append(reasons, ReasonCode{
						Code: code, Label: code, Source: ScorerSourceMLSidecar,
					})
				}
			}
		}
	}
	hash := strings.TrimSpace(payload.FeaturesHash)
	if hash == "" {
		hash = defaultHash
	}
	ver := strings.TrimSpace(payload.ModelVersion)
	if ver == "" {
		ver = "ml_v1"
	}
	return MLScorerResponse{
		ModelVersion: ver,
		ScorerSource: ScorerSourceMLSidecar,
		Score:        score,
		PdBps:        pdBps,
		ReasonCodes:  reasons,
		FeaturesHash: hash,
	}, nil
}

// parseDecimalScaled parses a numeric literal string into int64 * scale without float64.
func parseDecimalScaled(raw string, scale int64) (int64, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, fmt.Errorf("empty number")
	}
	neg := false
	if strings.HasPrefix(s, "-") {
		neg = true
		s = s[1:]
	}
	parts := strings.SplitN(s, ".", 2)
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil && parts[0] != "" {
		return 0, err
	}
	res := whole * scale
	if len(parts) == 2 && scale > 1 {
		frac := parts[1]
		remScale := scale
		for i := 0; i < len(frac) && remScale >= 10; i++ {
			ch := frac[i]
			if ch < '0' || ch > '9' {
				break
			}
			remScale /= 10
			res += int64(ch-'0') * remScale
		}
	}
	if neg {
		res = -res
	}
	return res, nil
}
