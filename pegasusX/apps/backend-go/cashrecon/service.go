package cashrecon

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

type Service struct {
	repo Repository
	cash ExpectedCashComputer
	now  func() time.Time
}

func NewService(repo Repository, cash ExpectedCashComputer) *Service {
	return &Service{
		repo: repo,
		cash: cash,
		now:  func() time.Time { return time.Now().UTC() },
	}
}

// SubmitReconciliation is called by the driver app when ending a shift.
func (s *Service) SubmitReconciliation(ctx context.Context, req SubmitReconciliationRequest) (*CashReconciliation, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("cash reconciliation service unavailable")
	}
	driverID := strings.TrimSpace(req.DriverId)
	if driverID == "" {
		return nil, fmt.Errorf("driver_id required")
	}
	supplierID := strings.TrimSpace(req.SupplierId)
	if supplierID == "" && s.repo != nil {
		if sid, err := s.repo.ResolveDriverSupplierID(ctx, driverID); err == nil && strings.TrimSpace(sid) != "" {
			supplierID = strings.TrimSpace(sid)
		}
	}
	if supplierID == "" {
		supplierID = auth.PreferTenantSupplierID(ctx, "")
	}
	if supplierID == "" {
		if sid, ok := auth.ResolveSupplierID(ctx); ok {
			supplierID = strings.TrimSpace(sid)
		}
	}
	if supplierID == "" {
		supplierID = "sup_61d822c6ab9714ca11f20db9"
	}
	shiftDate := req.ShiftDate
	if shiftDate.IsZero() {
		shiftDate = s.now()
	}
	var expected int64
	if s.cash != nil {
		computed, err := s.cash.ComputeExpectedCashMinor(ctx, driverID, shiftDate, req.RouteId)
		if err != nil {
			return nil, fmt.Errorf("compute expected cash: %w", err)
		}
		expected = computed
	} else {
		expected = req.ExpectedCashMinor
	}
	if req.ExpectedCashMinor != 0 && req.ExpectedCashMinor != expected {
		return nil, fmt.Errorf("expected_cash_mismatch: server computed %d", expected)
	}

	diff := req.DeclaredCashMinor - expected
	status := ReconciliationStatusPending
	if diff == 0 {
		status = ReconciliationStatusAccepted
	}

	cr := CashReconciliation{
		ReconciliationId:  uuid.New().String(),
		SupplierId:        supplierID,
		DriverId:          driverID,
		RouteId:           req.RouteId,
		ShiftDate:         shiftDate,
		ExpectedCashMinor: expected,
		DeclaredCashMinor: req.DeclaredCashMinor,
		DifferenceMinor:   diff,
		Status:            status,
		DriverNote:        req.DriverNote,
		CreatedAt:         s.now(),
	}

	if status == ReconciliationStatusAccepted {
		now := s.now()
		cr.ResolvedAt = &now
		sysActor := "SYSTEM"
		cr.ResolvedBy = &sysActor
	}

	eventType := EventCashReconciliationCreated
	if status == ReconciliationStatusAccepted {
		eventType = EventCashReconciliationAccepted
	}
	if err := s.repo.SaveReconciliation(ctx, cr, eventType); err != nil {
		return nil, fmt.Errorf("failed to save cash reconciliation: %w", err)
	}

	return &cr, nil
}

// Accept is called by Finance when reviewing a discrepancy.
func (s *Service) Accept(ctx context.Context, id string, actor string, note string) error {
	cr, err := s.repo.GetReconciliation(ctx, id)
	if err != nil {
		return err
	}
	if cr == nil {
		return fmt.Errorf("cash reconciliation not found")
	}
	if cr.Status != ReconciliationStatusPending && cr.Status != ReconciliationStatusDisputed {
		return fmt.Errorf("cannot accept from status %s", cr.Status)
	}

	cr.Status = ReconciliationStatusAccepted
	if note != "" {
		cr.FinanceNote = &note
	}
	now := s.now()
	cr.ResolvedAt = &now
	cr.ResolvedBy = &actor

	return s.repo.SaveReconciliation(ctx, *cr, EventCashReconciliationAccepted)
}

// WriteOff is called by Finance when money is lost and they agree to write it off.
func (s *Service) WriteOff(ctx context.Context, id string, actor string, note string) error {
	cr, err := s.repo.GetReconciliation(ctx, id)
	if err != nil {
		return err
	}
	if cr == nil {
		return fmt.Errorf("cash reconciliation not found")
	}
	if cr.Status != ReconciliationStatusPending && cr.Status != ReconciliationStatusDisputed {
		return fmt.Errorf("cannot write off from status %s", cr.Status)
	}

	cr.Status = ReconciliationStatusWriteOff
	if note != "" {
		cr.FinanceNote = &note
	}
	now := s.now()
	cr.ResolvedAt = &now
	cr.ResolvedBy = &actor

	return s.repo.SaveReconciliation(ctx, *cr, EventCashReconciliationWrittenOff)
}

func (s *Service) ListByDriver(ctx context.Context, driverID string, shiftDate time.Time) ([]CashReconciliation, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("cash reconciliation service unavailable")
	}
	return s.repo.ListByDriver(ctx, driverID, shiftDate)
}

func (s *Service) ListBySupplier(ctx context.Context, supplierID string, status ReconciliationStatus, limit int) ([]CashReconciliation, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("cash reconciliation service unavailable")
	}
	return s.repo.ListBySupplier(ctx, supplierID, status, limit)
}

func (s *Service) HasAcceptedReconciliation(ctx context.Context, driverID string, shiftDate time.Time) (bool, error) {
	if s == nil || s.cash == nil {
		return true, nil
	}
	gate, ok := s.cash.(ReconciliationGate)
	if !ok {
		return true, nil
	}
	return gate.HasAcceptedReconciliation(ctx, driverID, shiftDate)
}
