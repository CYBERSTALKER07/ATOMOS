// Package featureflags evaluates env defaults with optional per-tenant overrides.
package featureflags

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// MoneyAffectingFlags require two-person style audit when overridden.
var MoneyAffectingFlags = map[string]bool{
	"AR_INVOICES_ENABLED":              true,
	"AR_DUNNING_ENABLED":               true,
	"AUTO_ORDER_PLACE_ENABLED":         true,
	"AUTO_ORDER_SHADOW_ENABLED":        true,
	"AUTO_ORDER_SOAK_GATE_DISABLED":    true,
	"FISCAL_PROVIDER":                  true,
}

// Override is one tenant-scoped flag value.
type Override struct {
	FlagKey    string
	TenantType string
	TenantID   string
	Enabled    bool
	UpdatedBy  string
	UpdatedAt  time.Time
	Reason     string
	// Dual-control (money flags): PENDING until a second PLATFORM_ADMIN approves.
	Status     string // StatusActive | StatusPending
	ApprovedBy string
	ApprovedAt time.Time
}

const (
	StatusActive  = "ACTIVE"
	StatusPending = "PENDING"
)

// Repository stores overrides.
type Repository interface {
	Get(ctx context.Context, flagKey, tenantType, tenantID string) (Override, bool, error)
	Upsert(ctx context.Context, o Override) error
	ListForTenant(ctx context.Context, tenantType, tenantID string) ([]Override, error)
}

// MemoryRepository for tests.
type MemoryRepository struct {
	mu   sync.Mutex
	rows map[string]Override
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{rows: map[string]Override{}}
}

func ovKey(flag, tt, id string) string {
	return strings.ToUpper(flag) + "|" + strings.ToUpper(tt) + "|" + id
}

func (r *MemoryRepository) Get(_ context.Context, flagKey, tenantType, tenantID string) (Override, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	o, ok := r.rows[ovKey(flagKey, tenantType, tenantID)]
	return o, ok, nil
}

func (r *MemoryRepository) Upsert(_ context.Context, o Override) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	o.FlagKey = strings.ToUpper(strings.TrimSpace(o.FlagKey))
	o.TenantType = strings.ToUpper(strings.TrimSpace(o.TenantType))
	if o.Status == "" {
		o.Status = StatusActive
	}
	r.rows[ovKey(o.FlagKey, o.TenantType, o.TenantID)] = o
	return nil
}

func (r *MemoryRepository) ListForTenant(_ context.Context, tenantType, tenantID string) ([]Override, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Override, 0)
	prefix := "|" + strings.ToUpper(tenantType) + "|" + tenantID
	for k, o := range r.rows {
		if strings.HasSuffix(k, prefix) {
			out = append(out, o)
		}
	}
	return out, nil
}

// Service evaluates flags: tenant override → env default.
type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func envEnabled(flagKey string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(flagKey)))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// Evaluate returns whether flag is on for the tenant (empty tenant = env only).
func (s *Service) Evaluate(ctx context.Context, flagKey, tenantType, tenantID string) (bool, string, error) {
	flagKey = strings.ToUpper(strings.TrimSpace(flagKey))
	if flagKey == "" {
		return false, "missing_flag", fmt.Errorf("flag_required")
	}
	envVal := envEnabled(flagKey)
	source := "env"
	if s != nil && s.repo != nil && tenantType != "" && tenantID != "" {
		ov, ok, err := s.repo.Get(ctx, flagKey, tenantType, tenantID)
		if err != nil {
			return false, source, err
		}
		// Only ACTIVE overrides take effect; PENDING money-flag overrides await
		// a second approver and must not change runtime behavior yet.
		if ok && ov.Status == StatusActive {
			return ov.Enabled, "tenant_override", nil
		}
	}
	return envVal, source, nil
}

// SetOverride writes a tenant override. Money-affecting flags require a reason
// AND dual control: the override is stored PENDING and only takes effect after
// a different PLATFORM_ADMIN approves it via ApproveOverride. Non-money flags
// apply immediately (ACTIVE).
func (s *Service) SetOverride(ctx context.Context, o Override) error {
	if s == nil || s.repo == nil {
		return fmt.Errorf("featureflags_unavailable")
	}
	o.FlagKey = strings.ToUpper(strings.TrimSpace(o.FlagKey))
	o.TenantType = strings.ToUpper(strings.TrimSpace(o.TenantType))
	o.TenantID = strings.TrimSpace(o.TenantID)
	if o.FlagKey == "" || o.TenantType == "" || o.TenantID == "" {
		return fmt.Errorf("flag_tenant_required")
	}
	if MoneyAffectingFlags[o.FlagKey] {
		if strings.TrimSpace(o.Reason) == "" {
			return fmt.Errorf("reason_required_for_money_flag")
		}
		o.Status = StatusPending
		o.ApprovedBy = ""
		o.ApprovedAt = time.Time{}
	} else {
		o.Status = StatusActive
	}
	if o.UpdatedAt.IsZero() {
		o.UpdatedAt = time.Now().UTC()
	}
	return s.repo.Upsert(ctx, o)
}

// ApproveOverride activates a PENDING money-flag override. The approver must
// differ from the actor who set it (dual control) — the same person cannot
// both request and approve a money-affecting change.
func (s *Service) ApproveOverride(ctx context.Context, flagKey, tenantType, tenantID, approver string) error {
	if s == nil || s.repo == nil {
		return fmt.Errorf("featureflags_unavailable")
	}
	flagKey = strings.ToUpper(strings.TrimSpace(flagKey))
	tenantType = strings.ToUpper(strings.TrimSpace(tenantType))
	tenantID = strings.TrimSpace(tenantID)
	approver = strings.TrimSpace(approver)
	ov, ok, err := s.repo.Get(ctx, flagKey, tenantType, tenantID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("override_not_found")
	}
	if !MoneyAffectingFlags[ov.FlagKey] {
		return fmt.Errorf("not_a_money_flag")
	}
	if ov.Status != StatusPending {
		return fmt.Errorf("override_not_pending")
	}
	if strings.EqualFold(ov.UpdatedBy, approver) {
		return fmt.Errorf("approver_must_differ_from_setter")
	}
	ov.Status = StatusActive
	ov.ApprovedBy = approver
	ov.ApprovedAt = time.Now().UTC()
	return s.repo.Upsert(ctx, ov)
}

// RevertApproveToPending undoes a money-flag activation when audit cannot be
// recorded (B5 M-P0-11 fail-closed). Best-effort; callers treat audit failure
// as the primary error even if revert fails.
func (s *Service) RevertApproveToPending(ctx context.Context, flagKey, tenantType, tenantID string) error {
	if s == nil || s.repo == nil {
		return fmt.Errorf("featureflags_unavailable")
	}
	flagKey = strings.ToUpper(strings.TrimSpace(flagKey))
	tenantType = strings.ToUpper(strings.TrimSpace(tenantType))
	tenantID = strings.TrimSpace(tenantID)
	ov, ok, err := s.repo.Get(ctx, flagKey, tenantType, tenantID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("override_not_found")
	}
	if ov.Status != StatusActive {
		return nil
	}
	ov.Status = StatusPending
	ov.ApprovedBy = ""
	ov.ApprovedAt = time.Time{}
	return s.repo.Upsert(ctx, ov)
}
