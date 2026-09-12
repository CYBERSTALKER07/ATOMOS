package platformadmin

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type SetFeatureFlagRequest struct {
	Enabled bool   `json:"enabled"`
	Reason  string `json:"reason"`
}

type FeatureFlag struct {
	FlagKey    string    `json:"flag_key"`
	TenantType string    `json:"tenant_type"`
	TenantID   string    `json:"tenant_id"`
	Enabled    bool      `json:"enabled"`
	Status     string    `json:"status"`
	Reason     string    `json:"reason"`
	UpdatedBy  string    `json:"updated_by"`
	UpdatedAt  time.Time `json:"updated_at"`
	ApprovedBy string    `json:"approved_by,omitempty"`
}

func (s *Service) ListFeatureFlags(ctx context.Context, tenantType, tenantID string) ([]FeatureFlag, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("platformadmin_unavailable")
	}
	return s.repo.ListFeatureFlags(ctx, tenantType, tenantID)
}

func (s *Service) SetFeatureFlag(ctx context.Context, tenantType, tenantID, flagKey string, req SetFeatureFlagRequest, updatedBy string) error {
	if s == nil || s.repo == nil {
		return fmt.Errorf("platformadmin_unavailable")
	}
	tenantType = strings.ToUpper(strings.TrimSpace(tenantType))
	tenantID = strings.TrimSpace(tenantID)
	flagKey = strings.ToUpper(strings.TrimSpace(flagKey))
	if tenantType == "" || tenantID == "" || flagKey == "" {
		return fmt.Errorf("missing_required_fields")
	}
	if updatedBy == "" {
		return fmt.Errorf("updated_by_required")
	}

	status := "ACTIVE"
	if strings.HasPrefix(flagKey, "PAYMENT_") {
		status = "PENDING"
	}

	flag := FeatureFlag{
		FlagKey:    flagKey,
		TenantType: tenantType,
		TenantID:   tenantID,
		Enabled:    req.Enabled,
		Status:     status,
		Reason:     req.Reason,
		UpdatedBy:  updatedBy,
		UpdatedAt:  s.now(),
	}

	return s.repo.SetFeatureFlag(ctx, flag)
}
