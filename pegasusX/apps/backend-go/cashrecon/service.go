package cashrecon

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// SubmitReconciliation is called by the driver app when ending a shift.
func (s *Service) SubmitReconciliation(ctx context.Context, req SubmitReconciliationRequest) (*CashReconciliation, error) {
	diff := req.DeclaredCashMinor - req.ExpectedCashMinor
	status := ReconciliationStatusPending
	if diff == 0 {
		status = ReconciliationStatusAccepted // auto-accept if perfect match
	}

	cr := CashReconciliation{
		ReconciliationId:  uuid.New().String(),
		DriverId:          req.DriverId,
		RouteId:           req.RouteId,
		ShiftDate:         req.ShiftDate,
		ExpectedCashMinor: req.ExpectedCashMinor,
		DeclaredCashMinor: req.DeclaredCashMinor,
		DifferenceMinor:   diff,
		Status:            status,
		DriverNote:        req.DriverNote,
		CreatedAt:         time.Now(),
	}

	if status == ReconciliationStatusAccepted {
		now := time.Now()
		cr.ResolvedAt = &now
		sysActor := "SYSTEM"
		cr.ResolvedBy = &sysActor
	}

	if err := s.repo.SaveReconciliation(ctx, cr); err != nil {
		return nil, fmt.Errorf("failed to save cash reconciliation: %w", err)
	}

	return &cr, nil
}

// Accept is called by Finance when reviewing a discrepancy (e.g. they verified physical cash and accept the difference).
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
	now := time.Now()
	cr.ResolvedAt = &now
	cr.ResolvedBy = &actor

	return s.repo.SaveReconciliation(ctx, *cr)
}

// WriteOff is called by Finance when money is lost and they agree to write it off (accept the loss).
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
	now := time.Now()
	cr.ResolvedAt = &now
	cr.ResolvedBy = &actor

	// TODO: Emit OutboxEvent for ledger adjustments (Ledger integration)

	return s.repo.SaveReconciliation(ctx, *cr)
}
