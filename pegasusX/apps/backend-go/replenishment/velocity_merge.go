package replenishment

import (
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"strings"
)

// Demand source labels for reorder suggestions (L3 sell-through flywheel).
const (
	SourceWholesaleHistory = "WHOLESALE_HISTORY"
	SourceStorePOS         = "STORE_POS"
	DefaultSellThroughDays = 7
)

// IsKnownDemandSource reports whether code is a whitelist demand source token.
func IsKnownDemandSource(code string) bool {
	switch strings.ToUpper(strings.TrimSpace(code)) {
	case SourceWholesaleHistory, SourceStorePOS:
		return true
	default:
		return false
	}
}

// ParseDemandSourcesQuery reads source / sources query params.
// Returns normalized uppercase whitelist tokens, or error if any token is unknown.
// Empty input → nil (no filter).
func ParseDemandSourcesQuery(q url.Values) ([]string, error) {
	var raw []string
	if s := strings.TrimSpace(q.Get("source")); s != "" {
		raw = append(raw, s)
	}
	if s := strings.TrimSpace(q.Get("sources")); s != "" {
		for _, part := range strings.Split(s, ",") {
			if t := strings.TrimSpace(part); t != "" {
				raw = append(raw, t)
			}
		}
	}
	if len(raw) == 0 {
		return nil, nil
	}
	seen := map[string]bool{}
	var out []string
	for _, r := range raw {
		code := strings.ToUpper(strings.TrimSpace(r))
		if !IsKnownDemandSource(code) {
			return nil, fmt.Errorf("invalid_source")
		}
		if !seen[code] {
			seen[code] = true
			out = append(out, code)
		}
	}
	return out, nil
}

// SourcesJSONContainsNeedle is the LIKE pattern for a JSON string element in SourcesJson.
func SourcesJSONContainsNeedle(code string) string {
	return `%"%` + strings.ToUpper(strings.TrimSpace(code)) + `"%`
}

// SourcesMatchAny reports whether row sources contain any of filter (OR).
// Empty filter always matches. Empty row sources treated as WHOLESALE_HISTORY.
func SourcesMatchAny(rowSources, filter []string) bool {
	if len(filter) == 0 {
		return true
	}
	src := rowSources
	if len(src) == 0 {
		src = []string{SourceWholesaleHistory}
	}
	set := map[string]bool{}
	for _, s := range src {
		set[strings.ToUpper(strings.TrimSpace(s))] = true
	}
	for _, f := range filter {
		if set[strings.ToUpper(strings.TrimSpace(f))] {
			return true
		}
	}
	return false
}

// MergeDemandVelocities combines wholesale/sensing base demand with POS sell-through velocity.
// sellThroughUnits is net units sold over sellThroughDays (QtySold - QtyVoided).
// Returns demand per day = max(base, ST velocity) and source tags for UI.
func MergeDemandVelocities(baseDemandPerDay, sellThroughUnits float64, sellThroughDays int) (demand float64, sources []string) {
	if sellThroughDays <= 0 {
		sellThroughDays = DefaultSellThroughDays
	}
	stVel := 0.0
	if sellThroughUnits > 0 {
		stVel = sellThroughUnits / float64(sellThroughDays)
	}
	if baseDemandPerDay < 0 {
		baseDemandPerDay = 0
	}
	demand = math.Max(baseDemandPerDay, stVel)
	if baseDemandPerDay > 0 {
		sources = append(sources, SourceWholesaleHistory)
	}
	if stVel > 0 {
		sources = append(sources, SourceStorePOS)
	}
	if len(sources) == 0 {
		// Keep prior empty-demand behavior label for telemetry.
		sources = []string{SourceWholesaleHistory}
	}
	return demand, sources
}

// StripSellThroughFactor removes today's SELL_THROUGH contribution from AdjustedDemand
// so a 7-day rollup is not double-counted with the same-day factor write.
// Returns baseWithoutST and the stripped factor value F.
func StripSellThroughFactor(adjustedDemand, baseVelocity float64, factors map[string]float64) (baseWithoutST, factorF float64) {
	if factors == nil {
		return adjustedDemand, 0
	}
	f := factors["SELL_THROUGH"]
	if f == 0 {
		return adjustedDemand, 0
	}
	base := adjustedDemand - f
	if base < 0 {
		if baseVelocity > 0 {
			return baseVelocity, f
		}
		return 0, f
	}
	return base, f
}

// ParseFactorsJSON decodes DemandAdjustments.FactorsJson (object of float factors).
func ParseFactorsJSON(raw string) map[string]float64 {
	if raw == "" {
		return nil
	}
	var m map[string]float64
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		// Factors may be nested JSON null
		var any map[string]any
		if err2 := json.Unmarshal([]byte(raw), &any); err2 != nil {
			return nil
		}
		m = map[string]float64{}
		for k, v := range any {
			switch n := v.(type) {
			case float64:
				m[k] = n
			case json.Number:
				f, _ := n.Float64()
				m[k] = f
			}
		}
		return m
	}
	return m
}

// EncodeSourcesJSON returns a JSON array string for Spanner SourcesJson.
func EncodeSourcesJSON(sources []string) string {
	if len(sources) == 0 {
		return "[]"
	}
	b, err := json.Marshal(sources)
	if err != nil {
		return "[]"
	}
	return string(b)
}

// DecodeSourcesJSON parses SourcesJson into a string slice.
func DecodeSourcesJSON(raw string) []string {
	if raw == "" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out
}
