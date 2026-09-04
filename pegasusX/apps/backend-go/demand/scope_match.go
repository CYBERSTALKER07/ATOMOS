package demand

import (
	"encoding/json"
	"strings"
)

// RetailerGeo is the fail-closed geo snapshot used for REGION/CITY signal matching.
// Empty City/RegionID means unknown → regional scopes do not match.
type RetailerGeo struct {
	City     string
	RegionID string
	Address  string // DeliveryAddress / free-text fallback for CITY contains
}

// signalScopeMatches reports whether a DemandSignal scope applies to a retailer.
// GLOBAL / country:UZ always match. REGION/CITY require geo (fail-closed when unknown).
// RETAILER scopes match explicit retailer ids only.
func signalScopeMatches(scope string, meta json.RawMessage, retailerID string, geo RetailerGeo) bool {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return false
	}
	upper := strings.ToUpper(scope)
	lower := strings.ToLower(scope)

	switch {
	case upper == "GLOBAL" || lower == "country:uz":
		return true
	case strings.HasPrefix(lower, "retailer:"):
		want := strings.TrimSpace(scope[len("retailer:"):])
		return want != "" && strings.EqualFold(want, retailerID)
	case upper == "RETAILER" || upper == "RETAILER_SKU":
		return metaRetailerID(meta) != "" && strings.EqualFold(metaRetailerID(meta), retailerID)
	case upper == "REGION" || strings.HasPrefix(upper, "REGION:"):
		want := regionCodeFromScope(scope, meta)
		if want == "" {
			return false // bare REGION with no Meta → fail-closed
		}
		have := strings.TrimSpace(geo.RegionID)
		if have == "" {
			// Meta may pin retailer list for region campaigns without geo column.
			return metaRetailerInList(meta, retailerID)
		}
		return strings.EqualFold(have, want) || strings.EqualFold(normalizeGeoToken(have), normalizeGeoToken(want))
	case upper == "CITY" || strings.HasPrefix(upper, "CITY:") || strings.HasPrefix(lower, "city:"):
		want := cityNameFromScope(scope, meta)
		if want == "" {
			return false
		}
		return cityMatchesGeo(want, geo) || metaRetailerInList(meta, retailerID)
	default:
		// Legacy free-form: only exact retailer:uuid already handled; unknown scopes fail-closed.
		return false
	}
}

func regionCodeFromScope(scope string, meta json.RawMessage) string {
	upper := strings.ToUpper(strings.TrimSpace(scope))
	if strings.HasPrefix(upper, "REGION:") {
		return strings.TrimSpace(scope[len("REGION:"):])
	}
	if m := parseSignalMeta(meta); m != nil {
		if v := firstNonEmpty(m["region_id"], m["region"], m["regionId"]); v != "" {
			return v
		}
	}
	return ""
}

func cityNameFromScope(scope string, meta json.RawMessage) string {
	s := strings.TrimSpace(scope)
	lower := strings.ToLower(s)
	if strings.HasPrefix(lower, "city:") {
		return strings.TrimSpace(s[len("city:"):])
	}
	if strings.HasPrefix(strings.ToUpper(s), "CITY:") {
		return strings.TrimSpace(s[len("CITY:"):])
	}
	if m := parseSignalMeta(meta); m != nil {
		if v := firstNonEmpty(m["city"], m["city_name"], m["cityName"]); v != "" {
			return v
		}
	}
	return ""
}

func cityMatchesGeo(want string, geo RetailerGeo) bool {
	wantN := normalizeGeoToken(want)
	if wantN == "" {
		return false
	}
	if c := normalizeGeoToken(geo.City); c != "" {
		if c == wantN || strings.Contains(c, wantN) || strings.Contains(wantN, c) {
			return true
		}
	}
	addr := normalizeGeoToken(geo.Address)
	if addr == "" {
		return false
	}
	// Fail-closed for empty city: only match when address clearly contains the city token.
	return strings.Contains(addr, wantN)
}

func metaRetailerID(meta json.RawMessage) string {
	m := parseSignalMeta(meta)
	if m == nil {
		return ""
	}
	return firstNonEmpty(m["retailer_id"], m["retailerId"], m["RetailerId"])
}

func metaRetailerInList(meta json.RawMessage, retailerID string) bool {
	if strings.TrimSpace(retailerID) == "" || len(meta) == 0 {
		return false
	}
	var raw map[string]any
	if err := json.Unmarshal(meta, &raw); err != nil {
		return false
	}
	for _, key := range []string{"retailer_ids", "retailerIds", "retailers"} {
		v, ok := raw[key]
		if !ok {
			continue
		}
		switch t := v.(type) {
		case []any:
			for _, item := range t {
				if s, ok := item.(string); ok && strings.EqualFold(strings.TrimSpace(s), retailerID) {
					return true
				}
			}
		case []string:
			for _, s := range t {
				if strings.EqualFold(strings.TrimSpace(s), retailerID) {
					return true
				}
			}
		}
	}
	return strings.EqualFold(metaRetailerID(meta), retailerID)
}

func parseSignalMeta(meta json.RawMessage) map[string]string {
	if len(meta) == 0 {
		return nil
	}
	var raw map[string]any
	if err := json.Unmarshal(meta, &raw); err != nil {
		return nil
	}
	out := make(map[string]string, len(raw))
	for k, v := range raw {
		switch t := v.(type) {
		case string:
			out[k] = strings.TrimSpace(t)
		}
	}
	return out
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func normalizeGeoToken(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, "_", "")
	return s
}
