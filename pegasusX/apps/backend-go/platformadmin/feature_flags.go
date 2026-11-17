package platformadmin

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrFeatureFlagNotFound   = errors.New("feature_flag_not_found")
	ErrFeatureFlagNotPending = errors.New("feature_flag_not_pending")
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

func isDualControlFlag(flagKey string) bool {
	upper := strings.ToUpper(strings.TrimSpace(flagKey))
	return strings.HasPrefix(upper, "PAYMENT_") ||
		upper == "AR_INVOICES_ENABLED" ||
		upper == "AUTO_ORDER_PLACE_ENABLED" ||
		upper == "FISCAL_PROVIDER"
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
	if isDualControlFlag(flagKey) {
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

func (s *Service) ApproveFeatureFlag(ctx context.Context, tenantType, tenantID, flagKey, approvedBy string) error {
	if s == nil || s.repo == nil {
		return fmt.Errorf("platformadmin_unavailable")
	}
	tenantType = strings.ToUpper(strings.TrimSpace(tenantType))
	tenantID = strings.TrimSpace(tenantID)
	flagKey = strings.ToUpper(strings.TrimSpace(flagKey))
	approvedBy = strings.TrimSpace(approvedBy)
	if tenantType == "" || tenantID == "" || flagKey == "" {
		return fmt.Errorf("missing_required_fields")
	}
	if approvedBy == "" {
		return ErrActorRequired
	}

	flag, ok, err := s.repo.GetFeatureFlag(ctx, tenantType, tenantID, flagKey)
	if err != nil {
		return err
	}
	if !ok {
		return ErrFeatureFlagNotFound
	}
	if flag.Status != "PENDING" {
		return ErrFeatureFlagNotPending
	}
	if strings.EqualFold(flag.UpdatedBy, approvedBy) {
		return ErrApproverMustDiffer
	}

	flag.Status = "ACTIVE"
	flag.ApprovedBy = approvedBy
	flag.UpdatedAt = s.now()

	return s.repo.SetFeatureFlag(ctx, flag)
}
