package controltower

import (
	"encoding/json"
	"strings"
	"time"
)

// typeAliases maps playbook type tokens to exception types in the datastore.
var typeAliases = map[string][]string{
	"BUYER_REJECTED":       {"BUYER_REJECTED", "BUYER_EHF_REJECTION"},
	"BUYER_EHF_REJECTION":  {"BUYER_REJECTED", "BUYER_EHF_REJECTION"},
	"CASH_SHORT":           {"CASH_SHORT", "CASH_SHORTFALL"},
	"CASH_SHORTFALL":       {"CASH_SHORT", "CASH_SHORTFALL"},
	"SHOP_CLOSED_RETURNED": {"SHOP_CLOSED_RETURNED", "SHOP_CLOSED_RESOLVED"},
	"SHOP_CLOSED_RESOLVED": {"SHOP_CLOSED_RETURNED", "SHOP_CLOSED_RESOLVED"},
}

func expandTypes(types []string) map[string]bool {
	out := make(map[string]bool)
	for _, t := range types {
		key := strings.ToUpper(strings.TrimSpace(t))
		if key == "" {
			continue
		}
		out[key] = true
		if alts, ok := typeAliases[key]; ok {
			for _, a := range alts {
				out[a] = true
			}
		}
	}
	return out
}

// MatchesException returns true when all non-empty rule fields match the exception.
func (rules MatchRules) MatchesException(ex Exception, now time.Time) bool {
	if len(rules.Types) > 0 {
		allowed := expandTypes(rules.Types)
		if !allowed[strings.ToUpper(strings.TrimSpace(ex.Type))] {
			return false
		}
	}
	if len(rules.Severities) > 0 {
		sev := strings.ToUpper(strings.TrimSpace(ex.Severity))
		matched := false
		for _, s := range rules.Severities {
			if strings.EqualFold(strings.TrimSpace(s), sev) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	if rules.MinAmountMinor > 0 && ex.AmountMinor < rules.MinAmountMinor {
		return false
	}
	if len(rules.RetailerSegments) > 0 {
		seg := strings.ToUpper(strings.TrimSpace(ex.RetailerSegment))
		matched := false
		for _, s := range rules.RetailerSegments {
			if strings.EqualFold(strings.TrimSpace(s), seg) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	ageMin := now.Sub(ex.CreatedAt).Minutes()
	if rules.MinAgeMinutes > 0 && ageMin < float64(rules.MinAgeMinutes) {
		return false
	}
	if rules.MaxAgeMinutes > 0 && ageMin > float64(rules.MaxAgeMinutes) {
		return false
	}
	return true
}

func decodeMatchRules(raw json.RawMessage) (MatchRules, error) {
	if len(raw) == 0 {
		return MatchRules{}, nil
	}
	var rules MatchRules
	if err := json.Unmarshal(raw, &rules); err != nil {
		return MatchRules{}, err
	}
	return rules, nil
}

func decodeActions(raw json.RawMessage) ([]ActionSpec, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var actions []ActionSpec
	if err := json.Unmarshal(raw, &actions); err != nil {
		return nil, err
	}
	return actions, nil
}
