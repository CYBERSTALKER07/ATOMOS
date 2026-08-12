// Package platformadmin implements PLATFORM_ADMIN tenant lifecycle and audit.
package platformadmin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
)

const (
	StatusPending    = "PENDING"
	StatusApproved   = "APPROVED"
	StatusSuspended  = "SUSPENDED"
	StatusOffboarded = "OFFBOARDED"

	TenantSupplier = "SUPPLIER"
	TenantRetailer = "RETAILER"
)

// Tenant is one platform-governed org.
type Tenant struct {
	TenantType  string
	TenantID    string
	Status      string
	DisplayName string
	KybNotes    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ApprovedAt  *time.Time
	SuspendedAt *time.Time
	OffboardedAt *time.Time
}

// AuditRow is an immutable admin action record.
type AuditRow struct {
	AuditID      string
	ActorSubject string
	Action       string
	TenantType   string
	TenantID     string
	DetailJSON   string
	CreatedAt    time.Time
}

// Repository persists tenants + audit.
type Repository interface {
	UpsertTenant(ctx context.Context, t Tenant) error
	GetTenant(ctx context.Context, tenantType, tenantID string) (Tenant, bool, error)
	ListTenants(ctx context.Context, status string, limit int) ([]Tenant, error)
	InsertAudit(ctx context.Context, row AuditRow) error
	ListAudit(ctx context.Context, limit int) ([]AuditRow, error)
}

// Service orchestrates lifecycle transitions with audit.
type Service struct {
	repo Repository
	hub  *ws.Hub
	now  func() time.Time
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo, now: func() time.Time { return time.Now().UTC() }}
}

// SetHub attaches the platform-admin WebSocket hub for live console refresh.
func (s *Service) SetHub(h *ws.Hub) {
	if s != nil {
		s.hub = h
	}
}

func (s *Service) publish(ctx context.Context, eventType, action, tenantType, tenantID, actor string, detail any) {
	if s == nil || s.hub == nil {
		return
	}
	payload, err := json.Marshal(map[string]any{
		"type":        eventType,
		"action":      action,
		"tenant_type": tenantType,
		"tenant_id":   tenantID,
		"actor":       actor,
		"detail":      detail,
		"at":          s.now().Format(time.RFC3339Nano),
	})
	if err != nil {
		return
	}
	s.hub.Broadcast(ctx, ws.PlatformAdminRoom(), payload)
}

func (s *Service) EnsurePending(ctx context.Context, tenantType, tenantID, displayName string) error {
	if s == nil || s.repo == nil {
		return fmt.Errorf("platformadmin_unavailable")
	}
	tenantType = strings.ToUpper(strings.TrimSpace(tenantType))
	tenantID = strings.TrimSpace(tenantID)
	if tenantType == "" || tenantID == "" {
		return fmt.Errorf("tenant_required")
	}
	_, ok, err := s.repo.GetTenant(ctx, tenantType, tenantID)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	now := s.now()
	if err := s.repo.UpsertTenant(ctx, Tenant{
		TenantType:  tenantType,
		TenantID:    tenantID,
		Status:      StatusPending,
		DisplayName: strings.TrimSpace(displayName),
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		return err
	}
	s.publish(ctx, "PLATFORM_ADMIN_AUDIT", "TENANT_PENDING", tenantType, tenantID, "system", map[string]string{
		"display_name": strings.TrimSpace(displayName),
	})
	return nil
}

func (s *Service) List(ctx context.Context, status string, limit int) ([]Tenant, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("platformadmin_unavailable")
	}
	return s.repo.ListTenants(ctx, status, limit)
}

func (s *Service) Get(ctx context.Context, tenantType, tenantID string) (Tenant, bool, error) {
	if s == nil || s.repo == nil {
		return Tenant{}, false, fmt.Errorf("platformadmin_unavailable")
	}
	return s.repo.GetTenant(ctx, tenantType, tenantID)
}

func (s *Service) IsActive(ctx context.Context, tenantType, tenantID string) (bool, error) {
	t, ok, err := s.Get(ctx, tenantType, tenantID)
	if err != nil {
		return false, err
	}
	if !ok {
		// Legacy tenants without a row remain active (single-tenant seed).
		return true, nil
	}
	return t.Status == StatusApproved, nil
}

func (s *Service) Transition(ctx context.Context, actor, tenantType, tenantID, toStatus, kybNotes string) (Tenant, error) {
	if s == nil || s.repo == nil {
		return Tenant{}, fmt.Errorf("platformadmin_unavailable")
	}
	toStatus = strings.ToUpper(strings.TrimSpace(toStatus))
	switch toStatus {
	case StatusApproved, StatusSuspended, StatusOffboarded, StatusPending:
	default:
		return Tenant{}, fmt.Errorf("invalid_status")
	}
	t, ok, err := s.repo.GetTenant(ctx, tenantType, tenantID)
	if err != nil {
		return Tenant{}, err
	}
	now := s.now()
	if !ok {
		t = Tenant{
			TenantType:  strings.ToUpper(strings.TrimSpace(tenantType)),
			TenantID:    strings.TrimSpace(tenantID),
			Status:      StatusPending,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
	}
	prev := t.Status
	if err := validateTransition(prev, toStatus); err != nil {
		return Tenant{}, err
	}
	t.Status = toStatus
	t.UpdatedAt = now
	if notes := strings.TrimSpace(kybNotes); notes != "" {
		t.KybNotes = notes
	}
	switch toStatus {
	case StatusApproved:
		t.ApprovedAt = &now
		t.SuspendedAt = nil
	case StatusSuspended:
		t.SuspendedAt = &now
	case StatusOffboarded:
		t.OffboardedAt = &now
	}
	if err := s.repo.UpsertTenant(ctx, t); err != nil {
		return Tenant{}, err
	}
	detailMap := map[string]string{"to": toStatus, "from": prev}
	detail, _ := json.Marshal(detailMap)
	action := "TENANT_" + toStatus
	_ = s.repo.InsertAudit(ctx, AuditRow{
		AuditID:      uuid.NewString(),
		ActorSubject: strings.TrimSpace(actor),
		Action:       action,
		TenantType:   t.TenantType,
		TenantID:     t.TenantID,
		DetailJSON:   string(detail),
		CreatedAt:    now,
	})
	s.publish(ctx, "PLATFORM_ADMIN_AUDIT", action, t.TenantType, t.TenantID, actor, detailMap)
	return t, nil
}

func validateTransition(from, to string) error {
	from = strings.ToUpper(strings.TrimSpace(from))
	to = strings.ToUpper(strings.TrimSpace(to))
	if from == "" {
		from = StatusPending
	}
	if from == to {
		return nil
	}
	allowed := map[string]map[string]bool{
		StatusPending:    {StatusApproved: true, StatusOffboarded: true},
		StatusApproved:   {StatusSuspended: true, StatusOffboarded: true},
		StatusSuspended:  {StatusApproved: true, StatusOffboarded: true},
		StatusOffboarded: {},
	}
	if m, ok := allowed[from]; ok && m[to] {
		return nil
	}
	return fmt.Errorf("illegal_transition:%s->%s", from, to)
}

func (s *Service) ListAudit(ctx context.Context, limit int) ([]AuditRow, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("platformadmin_unavailable")
	}
	return s.repo.ListAudit(ctx, limit)
}

// RecordFlagAudit implements featureflags.FlagAuditor for dual-control place-flip trail.
func (s *Service) RecordFlagAudit(ctx context.Context, actor, action, tenantType, tenantID, detailJSON string) error {
	if s == nil || s.repo == nil {
		return fmt.Errorf("platformadmin_unavailable")
	}
	if err := s.repo.InsertAudit(ctx, AuditRow{
		AuditID:      uuid.NewString(),
		ActorSubject: actor,
		Action:       action,
		TenantType:   tenantType,
		TenantID:     tenantID,
		DetailJSON:   detailJSON,
		CreatedAt:    time.Now().UTC(),
	}); err != nil {
		return err
	}
	var detail any = detailJSON
	var parsed map[string]any
	if json.Unmarshal([]byte(detailJSON), &parsed) == nil {
		detail = parsed
	}
	s.publish(ctx, "PLATFORM_ADMIN_AUDIT", action, tenantType, tenantID, actor, detail)
	return nil
}
