package platformadmin

import "context"

func (r *MemoryRepository) ListFeatureFlags(ctx context.Context, tenantType, tenantID string) ([]FeatureFlag, error) {
	return nil, nil
}

func (r *MemoryRepository) SetFeatureFlag(ctx context.Context, flag FeatureFlag) error {
	return nil
}
