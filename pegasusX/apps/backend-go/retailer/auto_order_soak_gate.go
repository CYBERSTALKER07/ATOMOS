package retailer

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// Soak-gate thresholds for the AUTO_ORDER place flip. The place flip is a
// money-affecting autonomy step: it must be backed by shadow evidence, not just
// an env var. These thresholds define what "enough evidence" means and can be
// tuned per environment.
//
//	AUTO_ORDER_SOAK_MIN_PROPOSALS   minimum shadow proposals in the window (default 20)
//	AUTO_ORDER_SOAK_MAX_WAPE        max weighted-abs-pct-error vs actual orders (default 0.30)
//	AUTO_ORDER_SOAK_MIN_UNMODIFIED  min unmodified accept rate (default 0.80 — matches place-flip policy)
//	AUTO_ORDER_SOAK_GATE_DISABLED   "true" bypasses the gate (break-glass; default off)
type SoakGateConfig struct {
	MinProposals  int64
	MaxWAPE       float64
	MinUnmodified float64
	Disabled      bool
}

// SoakGateConfigFromEnv reads the gate thresholds from the environment.
func SoakGateConfigFromEnv() SoakGateConfig {
	return SoakGateConfig{
		MinProposals:  envInt64("AUTO_ORDER_SOAK_MIN_PROPOSALS", 20),
		MaxWAPE:       envFloat("AUTO_ORDER_SOAK_MAX_WAPE", 0.30),
		MinUnmodified: envFloat("AUTO_ORDER_SOAK_MIN_UNMODIFIED", 0.80),
		Disabled:      strings.EqualFold(strings.TrimSpace(os.Getenv("AUTO_ORDER_SOAK_GATE_DISABLED")), "true"),
	}
}

// SoakGateDecision explains whether place mode is permitted and why.
type SoakGateDecision struct {
	Allowed bool     `json:"allowed"`
	Reasons []string `json:"reasons,omitempty"`
	Stats   *AutoOrderShadowStats `json:"stats,omitempty"`
}

// EvaluateSoakGate computes whether the retailer may run place mode based on
// its 30-day shadow soak stats. Fail-closed: any stat error or missing data
// denies the flip unless the gate is explicitly disabled.
func (s *Service) EvaluateSoakGate(ctx context.Context, orgID string, cfg SoakGateConfig) SoakGateDecision {
	if cfg.Disabled {
		return SoakGateDecision{Allowed: true, Reasons: []string{"gate_disabled"}}
	}
	stats, err := s.loadShadowStats(ctx, orgID, 30)
	if err != nil {
		return SoakGateDecision{Allowed: false, Reasons: []string{"stats_unavailable: " + err.Error()}}
	}
	d := SoakGateDecision{Allowed: true, Stats: &stats}
	if stats.ProposalCount < cfg.MinProposals {
		d.Allowed = false
		d.Reasons = append(d.Reasons, "insufficient_shadow_proposals")
	}
	// Only judge accuracy when there is something to judge.
	if stats.MatchedOrders > 0 {
		if stats.WAPE > cfg.MaxWAPE {
			d.Allowed = false
			d.Reasons = append(d.Reasons, "wape_above_threshold")
		}
		if stats.UnmodifiedRate < cfg.MinUnmodified {
			d.Allowed = false
			d.Reasons = append(d.Reasons, "unmodified_rate_below_threshold")
		}
	} else if stats.ProposalCount >= cfg.MinProposals {
		// Enough proposals but none matched a real order → shadow is not
		// predicting reality; do not allow place.
		d.Allowed = false
		d.Reasons = append(d.Reasons, "no_matched_orders")
	}
	if d.Allowed {
		d.Reasons = append(d.Reasons, "soak_passed")
	}
	return d
}

// placeAllowedForRetailer combines dual-control place flag evaluation with the
// per-retailer soak gate. Used by the worker before honoring ExecutionMode=place.
func (s *Service) placeAllowedForRetailer(ctx context.Context, orgID string) bool {
	enabled := s.autoOrderPlaceEnabled
	if s.placeFlags != nil && strings.TrimSpace(orgID) != "" {
		if on, _, err := s.placeFlags.Evaluate(ctx, "AUTO_ORDER_PLACE_ENABLED", "RETAILER", orgID); err == nil {
			enabled = on
		}
	}
	if !enabled {
		return false
	}
	if s.soakGateDisabled {
		return true
	}
	return s.EvaluateSoakGate(ctx, orgID, SoakGateConfigFromEnv()).Allowed
}

func envInt64(k string, def int64) int64 {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	return def
}

func envFloat(k string, def float64) float64 {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

// HandleAutoOrderSoakGate GET /v1/retailer/settings/auto-order/soak-gate
// returns the live gate decision (allowed + reasons + stats) for the caller.
func (s *Service) HandleAutoOrderSoakGate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	cfg := SoakGateConfigFromEnv()
	d := s.EvaluateSoakGate(r.Context(), orgID, cfg)
	writeJSON(w, http.StatusOK, map[string]any{
		"decision":  d,
		"thresholds": map[string]any{
			"min_proposals":  cfg.MinProposals,
			"max_wape":       cfg.MaxWAPE,
			"min_unmodified": cfg.MinUnmodified,
			"gate_disabled":  cfg.Disabled,
		},
		"place_flag_enabled": s.autoOrderPlaceEnabled,
	})
}

// HandleAutoOrderSoakArtifact GET /v1/retailer/settings/auto-order/soak-artifact
// emits the 30-day shadow soak as a durable, downloadable JSON artifact — the
// evidence pack an operator attaches to the place-flip approval. Content-Disposition
// is set so the browser saves it as a file.
func (s *Service) HandleAutoOrderSoakArtifact(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	cfg := SoakGateConfigFromEnv()
	d := s.EvaluateSoakGate(r.Context(), orgID, cfg)
	props, perr := s.listShadowProposals(r.Context(), orgID, 1000)
	if perr != nil {
		props = []AutoOrderShadowProposal{}
	}
	rate := 0.0
	if d.Stats != nil {
		rate = d.Stats.UnmodifiedRate
	}
	artifact := map[string]any{
		"artifact":     "auto-order-soak",
		"version":      1,
		"retailer_id":  orgID,
		"generated_at": s.now().UTC().Format(time.RFC3339Nano),
		"window_days":  30,
		"soak_days":    30,
		"thresholds": map[string]any{
			"min_proposals":  cfg.MinProposals,
			"max_wape":       cfg.MaxWAPE,
			"min_unmodified": cfg.MinUnmodified,
		},
		"decision": d,
		// Dual field names for flip-check + runtime schema compatibility (P1-2).
		"unmodified_accept_rate":      rate,
		"unmodified_acceptance_rate":  rate,
		"proposals":                   props,
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", `attachment; filename="auto-order-soak-`+orgID+`.json"`)
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(artifact)
}
