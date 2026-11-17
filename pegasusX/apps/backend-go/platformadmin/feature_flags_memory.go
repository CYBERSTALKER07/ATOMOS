package platformadmin

import (
	"context"
	"strings"
)

func featureFlagKey(tt, id, fk string) string {
	return strings.ToUpper(strings.TrimSpace(tt)) + "|" + strings.TrimSpace(id) + "|" + strings.ToUpper(strings.TrimSpace(fk))
}

func (r *MemoryRepository) GetFeatureFlag(_ context.Context, tenantType, tenantID, flagKey string) (FeatureFlag, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.flags == nil {
		return FeatureFlag{}, false, nil
	}
	f, ok := r.flags[featureFlagKey(tenantType, tenantID, flagKey)]
	return f, ok, nil
}

func (r *MemoryRepository) ListFeatureFlags(_ context.Context, tenantType, tenantID string) ([]FeatureFlag, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.flags == nil {
		return nil, nil
	}
	prefix := strings.ToUpper(strings.TrimSpace(tenantType)) + "|" + strings.TrimSpace(tenantID) + "|"
	out := make([]FeatureFlag, 0, len(r.flags))
	for k, f := range r.flags {
		if strings.HasPrefix(k, prefix) {
			out = append(out, f)
		}
	}
	return out, nil
}

func (r *MemoryRepository) SetFeatureFlag(_ context.Context, flag FeatureFlag) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.flags == nil {
		r.flags = make(map[string]FeatureFlag)
	}
	r.flags[featureFlagKey(flag.TenantType, flag.TenantID, flag.FlagKey)] = flag
	return nil
}
