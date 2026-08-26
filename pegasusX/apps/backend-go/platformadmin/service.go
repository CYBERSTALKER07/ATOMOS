// Package platformadmin implements PLATFORM_ADMIN tenant lifecycle and audit.
package platformadmin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
)

var (
	ErrTenantNotFound     = errors.New("tenant_not_found")
	ErrActorRequired      = errors.New("actor_required")
	ErrMarketCodeRequired = errors.New("market_code_required")
	ErrUnknownMarket      = errors.New("unknown_market")
	ErrMarketNotShipped   = errors.New("market_not_shipped")
	ErrHomeCellMismatch   = errors.New("home_cell_mismatch")
	ErrApproverMustDiffer = errors.New("approver_must_differ")
	ErrPackCellMismatch   = errors.New("pack_cell_mismatch")
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
	TenantType   string
	TenantID     string
	Status       string
	DisplayName  string
	KybNotes     string
	MarketCode   string `json:"market_code,omitempty"`
	HomeCell     string `json:"home_cell,omitempty"`
	RequestedBy  string `json:"requested_by,omitempty"`
	ApprovedBy   string `json:"approved_by,omitempty"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	ApprovedAt   *time.Time
	SuspendedAt  *time.Time
	OffboardedAt *time.Time
}

// TransitionInput is a PA lifecycle change. APPROVE is dual-control (GS-T3).
type TransitionInput struct {
	Actor      string
	TenantType string
	TenantID   string
	Status     string
	KybNotes   string
	MarketCode string
	HomeCell   string
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
	ListFeatureFlags(ctx context.Context, tenantType, tenantID string) ([]FeatureFlag, error)
	SetFeatureFlag(ctx context.Context, flag FeatureFlag) error
}

// Service orchestrates lifecycle transitions with audit.
type Service struct {
	repo Repository
	hub  *ws.Hub
	now  func() time.Time
	// OnApproved copies pack+cell onto the supplier row after KYB APPROVED.
	OnApproved func(ctx context.Context, tenantType, tenantID, marketCode, homeCell string) error
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
		// GS-T3: missing KYB row is not active. Seed must be EnsurePending + approved.
		return false, nil
	}
	return t.Status == StatusApproved, nil
}

func (s *Service) Transition(ctx context.Context, in TransitionInput) (Tenant, error) {
	if s == nil || s.repo == nil {
		return Tenant{}, fmt.Errorf("platformadmin_unavailable")
	}
	toStatus := strings.ToUpper(strings.TrimSpace(in.Status))
	switch toStatus {
	case StatusApproved, StatusSuspended, StatusOffboarded, StatusPending:
	default:
		return Tenant{}, fmt.Errorf("invalid_status")
	}
	t, ok, err := s.repo.GetTenant(ctx, in.TenantType, in.TenantID)
	if err != nil {
		return Tenant{}, err
	}
	if !ok {
		return Tenant{}, ErrTenantNotFound
	}
	if toStatus == StatusApproved {
		return s.approve(ctx, in, t)
	}
	prev := t.Status
	if err := validateTransition(prev, toStatus); err != nil {
		return Tenant{}, err
	}
	now := s.now()
	t.Status = toStatus
	t.UpdatedAt = now
	if notes := strings.TrimSpace(in.KybNotes); notes != "" {
		t.KybNotes = notes
	}
	switch toStatus {
	case StatusSuspended:
		t.SuspendedAt = &now
		t.RequestedBy = ""
	case StatusOffboarded:
		t.OffboardedAt = &now
		t.RequestedBy = ""
	}
	return s.persistTransition(ctx, t, strings.TrimSpace(in.Actor), "TENANT_"+toStatus, map[string]string{
		"to": toStatus, "from": prev,
	})
}

func (s *Service) approve(ctx context.Context, in TransitionInput, t Tenant) (Tenant, error) {
	if t.Status == StatusApproved {
		return t, nil
	}
	if err := validateTransition(t.Status, StatusApproved); err != nil {
		return Tenant{}, err
	}
	pack, cell, err := resolveApprovePack(in.MarketCode, in.HomeCell)
	if err != nil {
		return Tenant{}, err
	}
	actor := strings.TrimSpace(in.Actor)
	if isSystemActor(actor) {
		return s.finishApprove(ctx, t, actor, pack, cell, in.KybNotes, t.Status)
	}
	if actor == "" {
		return Tenant{}, ErrActorRequired
	}
	if strings.TrimSpace(t.RequestedBy) == "" {
		t.RequestedBy = actor
		t.MarketCode = pack
		t.HomeCell = cell
		if notes := strings.TrimSpace(in.KybNotes); notes != "" {
			t.KybNotes = notes
		}
		t.UpdatedAt = s.now()
		return s.persistTransition(ctx, t, actor, "TENANT_APPROVE_REQUESTED", map[string]string{
			"to":          StatusPending,
			"from":        t.Status,
			"market_code": pack,
			"home_cell":   cell,
		})
	}
	if strings.EqualFold(t.RequestedBy, actor) {
		return Tenant{}, ErrApproverMustDiffer
	}
	if auth.NormalizeMarketCode(t.MarketCode) != pack || strings.ToLower(strings.TrimSpace(t.HomeCell)) != cell {
		return Tenant{}, ErrPackCellMismatch
	}
	return s.finishApprove(ctx, t, actor, pack, cell, in.KybNotes, t.Status)
}

func (s *Service) finishApprove(ctx context.Context, t Tenant, actor, pack, cell, notes, prev string) (Tenant, error) {
	now := s.now()
	t.Status = StatusApproved
	t.MarketCode = pack
	t.HomeCell = cell
	t.ApprovedBy = actor
	t.ApprovedAt = &now
	t.SuspendedAt = nil
	t.UpdatedAt = now
	if n := strings.TrimSpace(notes); n != "" {
		t.KybNotes = n
	}
	out, err := s.persistTransition(ctx, t, actor, "TENANT_APPROVED", map[string]string{
		"to":          StatusApproved,
		"from":        prev,
		"market_code": pack,
		"home_cell":   cell,
	})
	if err != nil {
		return Tenant{}, err
	}
	if s.OnApproved != nil {
		if hookErr := s.OnApproved(ctx, t.TenantType, t.TenantID, pack, cell); hookErr != nil {
			// KYB row is the activity source; pack copy is best-effort.
			_ = hookErr
		}
	}
	return out, nil
}

func (s *Service) persistTransition(ctx context.Context, t Tenant, actor, action string, detailMap map[string]string) (Tenant, error) {
	if err := s.repo.UpsertTenant(ctx, t); err != nil {
		return Tenant{}, err
	}
	detail, _ := json.Marshal(detailMap)
	_ = s.repo.InsertAudit(ctx, AuditRow{
		AuditID:      uuid.NewString(),
		ActorSubject: actor,
		Action:       action,
		TenantType:   t.TenantType,
		TenantID:     t.TenantID,
		DetailJSON:   string(detail),
		CreatedAt:    s.now(),
	})
	s.publish(ctx, "PLATFORM_ADMIN_AUDIT", action, t.TenantType, t.TenantID, actor, detailMap)
	return t, nil
}

func resolveApprovePack(marketCode, homeCell string) (string, string, error) {
	code := auth.NormalizeMarketCode(marketCode)
	if code == "" {
		return "", "", ErrMarketCodeRequired
	}
	pack, ok := auth.ResolveMarketPack(code)
	if !ok {
		return "", "", fmt.Errorf("%w: %s", ErrUnknownMarket, code)
	}
	if pack.Status != auth.MarketPackShipped {
		return "", "", fmt.Errorf("%w: %s", ErrMarketNotShipped, pack.Code)
	}
	cell := strings.ToLower(strings.TrimSpace(homeCell))
	if cell == "" {
		cell = pack.HomeCell
	}
	if cell != pack.HomeCell {
		return "", "", ErrHomeCellMismatch
	}
	return pack.Code, pack.HomeCell, nil
}

func isSystemActor(actor string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(actor)), "system:")
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
