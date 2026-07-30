package creditnote

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

// CreateFromBuyerReject generates a full credit note when a buyer rejects an entire order upon delivery.
func (s *Service) CreateFromBuyerReject(ctx context.Context, orderId string, actor string) (*CreditNote, error) {
	// In a real implementation, we would query the order lines here to build the credit note lines.
	// For now, we scaffold the structure.

	cn := CreditNote{
		CreditNoteId: uuid.New().String(),
		OrderId:      orderId,
		Type:         CreditNoteTypeBuyerReject,
		Status:       CreditNoteStatusDraft,
		ReasonCode:   "BUYER_REJECTED_DELIVERY",
		CreatedBy:    actor,
		CreatedAt:    time.Now(),
	}

	if err := s.repo.SaveCreditNote(ctx, cn); err != nil {
		return nil, fmt.Errorf("failed to save credit note: %w", err)
	}

	return &cn, nil
}

// CreateFromClaim generates a credit note from an approved logistics claim.
func (s *Service) CreateFromClaim(ctx context.Context, claimId string, actor string) (*CreditNote, error) {
	cn := CreditNote{
		CreditNoteId: uuid.New().String(),
		OrderId:      "ORDER_FROM_CLAIM", // mock
		Type:         CreditNoteTypeClaim,
		Status:       CreditNoteStatusDraft,
		ReasonCode:   "LOGISTICS_CLAIM_APPROVED",
		CreatedBy:    actor,
		CreatedAt:    time.Now(),
	}

	if err := s.repo.SaveCreditNote(ctx, cn); err != nil {
		return nil, fmt.Errorf("failed to save credit note: %w", err)
	}

	return &cn, nil
}

// CreateManual allows a manager or finance user to issue a manual credit note for a specific amount/lines.
func (s *Service) CreateManual(ctx context.Context, req CreateManualCreditNoteRequest, actor string) (*CreditNote, error) {
	reasonText := req.ReasonText
	cn := CreditNote{
		CreditNoteId: uuid.New().String(),
		OrderId:      req.OrderId,
		Type:         CreditNoteTypeManual,
		Status:       CreditNoteStatusDraft,
		ReasonCode:   req.ReasonCode,
		ReasonText:   &reasonText,
		CreatedBy:    actor,
		CreatedAt:    time.Now(),
	}

	// Scaffold lines mapping here...

	if err := s.repo.SaveCreditNote(ctx, cn); err != nil {
		return nil, fmt.Errorf("failed to save credit note: %w", err)
	}

	return &cn, nil
}

// Issue advances the credit note to ISSUED or FISCAL_PENDING state and triggers reverse logistics if applicable.
func (s *Service) Issue(ctx context.Context, creditNoteId string, actor string) error {
	cn, err := s.repo.GetCreditNote(ctx, creditNoteId)
	if err != nil {
		return err
	}
	if cn == nil {
		return fmt.Errorf("credit note not found")
	}

	cn.Status = CreditNoteStatusFiscalPending
	now := time.Now()
	cn.IssuedAt = &now

	if err := s.repo.SaveCreditNote(ctx, *cn); err != nil {
		return err
	}

	// Create reverse logistics task
	task := ReverseLogisticsTask{
		TaskId:       uuid.New().String(),
		CreditNoteId: cn.CreditNoteId,
		OrderId:      cn.OrderId,
		Status:       ReverseTaskStatusOpen,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.repo.SaveReverseLogisticsTask(ctx, task); err != nil {
		return fmt.Errorf("failed to create reverse logistics task: %w", err)
	}

	return nil
}
