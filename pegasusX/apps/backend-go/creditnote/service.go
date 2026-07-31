package creditnote

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
		now:  func() time.Time { return time.Now().UTC() },
	}
}

// CreateFromBuyerReject generates a full credit note when a buyer rejects an entire order upon delivery.
func (s *Service) CreateFromBuyerReject(ctx context.Context, orderId string, actor string) (*CreditNote, error) {
	existingCNs, err := s.repo.ListCreditNotesByOrder(ctx, orderId)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing credit notes: %w", err)
	}

	for _, cn := range existingCNs {
		if cn.Type == CreditNoteTypeBuyerReject && cn.Status != CreditNoteStatusCancelled {
			return nil, fmt.Errorf("a buyer reject credit note already exists for order %s", orderId)
		}
	}

	lines, err := s.repo.GetDeliveredOrderLines(ctx, orderId)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch order lines: %w", err)
	}

	if len(lines) == 0 {
		return nil, fmt.Errorf("no delivered lines found for order %s to credit", orderId)
	}

	cn := CreditNote{
		CreditNoteId: uuid.New().String(),
		OrderId:      orderId,
		Type:         CreditNoteTypeBuyerReject,
		Status:       CreditNoteStatusDraft,
		ReasonCode:   "BUYER_REJECTED_DELIVERY",
		CreatedBy:    actor,
		CreatedAt:    s.now(),
	}

	var totalNet, totalVat, totalGross int64
	for i := range lines {
		lines[i].CreditNoteId = cn.CreditNoteId
		lines[i].LineId = uuid.New().String()

		totalNet += lines[i].LineNetMinor
		totalVat += lines[i].LineVatMinor
		totalGross += lines[i].LineGrossMinor
	}

	cn.Lines = lines
	cn.TotalNetMinor = totalNet
	cn.TotalVatMinor = totalVat
	cn.TotalGrossMinor = totalGross

	if err := s.repo.SaveCreditNote(ctx, cn, EventCreditNoteCreated); err != nil {
		return nil, fmt.Errorf("failed to save credit note: %w", err)
	}

	return &cn, nil
}

// CreateFromClaim generates a credit note from an approved logistics claim.
func (s *Service) CreateFromClaim(ctx context.Context, claimId string, actor string) (*CreditNote, error) {
	orderID, amountMinor, ok, err := s.repo.GetClaimOrder(ctx, claimId)
	if err != nil {
		return nil, err
	}
	if !ok || orderID == "" {
		return nil, fmt.Errorf("claim %s not found", claimId)
	}
	cn := CreditNote{
		CreditNoteId:    uuid.New().String(),
		OrderId:         orderID,
		Type:            CreditNoteTypeClaim,
		Status:          CreditNoteStatusDraft,
		ReasonCode:      "LOGISTICS_CLAIM_APPROVED",
		TotalGrossMinor: amountMinor,
		TotalNetMinor:   amountMinor,
		CreatedBy:       actor,
		CreatedAt:       s.now(),
	}
	reason := fmt.Sprintf("claim_id=%s", claimId)
	cn.ReasonText = &reason

	if err := s.repo.SaveCreditNote(ctx, cn, EventCreditNoteCreated); err != nil {
		return nil, fmt.Errorf("failed to save credit note: %w", err)
	}

	return &cn, nil
}

// CreateManual allows a manager or finance user to issue a manual credit note.
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
		CreatedAt:    s.now(),
	}

	lines, err := s.repo.GetDeliveredOrderLines(ctx, req.OrderId)
	if err != nil {
		return nil, err
	}
	lineByID := make(map[string]CreditNoteLine, len(lines))
	for _, l := range lines {
		lineByID[l.OrderLineId] = l
	}
	for _, in := range req.Lines {
		base, ok := lineByID[in.OrderLineId]
		if !ok || in.Qty <= 0 {
			continue
		}
		qty := in.Qty
		if qty > base.Qty {
			qty = base.Qty
		}
		line := base
		line.CreditNoteId = cn.CreditNoteId
		line.LineId = uuid.New().String()
		line.Qty = qty
		line.LineNetMinor = (base.LineNetMinor / base.Qty) * qty
		line.LineVatMinor = (base.LineVatMinor / base.Qty) * qty
		line.LineGrossMinor = (base.LineGrossMinor / base.Qty) * qty
		cn.Lines = append(cn.Lines, line)
		cn.TotalNetMinor += line.LineNetMinor
		cn.TotalVatMinor += line.LineVatMinor
		cn.TotalGrossMinor += line.LineGrossMinor
	}

	if err := s.repo.SaveCreditNote(ctx, cn, EventCreditNoteCreated); err != nil {
		return nil, fmt.Errorf("failed to save credit note: %w", err)
	}

	return &cn, nil
}

// Issue advances the credit note to FISCAL_PENDING and creates reverse logistics.
func (s *Service) Issue(ctx context.Context, creditNoteId string, actor string) error {
	cn, err := s.repo.GetCreditNote(ctx, creditNoteId)
	if err != nil {
		return err
	}
	if cn == nil {
		return fmt.Errorf("credit note not found")
	}

	cn.Status = CreditNoteStatusFiscalPending
	now := s.now()
	cn.IssuedAt = &now

	if err := s.repo.SaveCreditNote(ctx, *cn, EventCreditNoteIssued); err != nil {
		return err
	}

	expected := make(map[string]int64)
	for _, line := range cn.Lines {
		expected[line.Sku] += line.Qty
	}
	expectedJSON, _ := json.Marshal(expected)

	task := ReverseLogisticsTask{
		TaskId:          uuid.New().String(),
		CreditNoteId:    cn.CreditNoteId,
		OrderId:         cn.OrderId,
		Status:          ReverseTaskStatusOpen,
		ExpectedQtyJson: expectedJSON,
		CreatedAt:       s.now(),
		UpdatedAt:       s.now(),
	}

	if err := s.repo.SaveReverseLogisticsTask(ctx, task, EventReverseLogisticsCreated); err != nil {
		return fmt.Errorf("failed to create reverse logistics task: %w", err)
	}

	return nil
}

func (s *Service) ListBySupplier(ctx context.Context, supplierID string, status CreditNoteStatus, limit int) ([]CreditNote, error) {
	return s.repo.ListBySupplier(ctx, supplierID, status, limit)
}

func (s *Service) ReceiveReverseTask(ctx context.Context, taskID, warehouseID string, received map[string]int64, actor string) error {
	raw, err := json.Marshal(received)
	if err != nil {
		return err
	}
	return s.repo.ReceiveReverseLogisticsTask(ctx, taskID, warehouseID, raw, actor)
}

// ClaimsBridge adapts Service for claims.CreditNoteCreator.
type ClaimsBridge struct {
	Svc *Service
}

func (b ClaimsBridge) CreateFromClaim(ctx context.Context, claimID, actor string) error {
	if b.Svc == nil {
		return fmt.Errorf("credit note service unavailable")
	}
	_, err := b.Svc.CreateFromClaim(ctx, claimID, actor)
	return err
}
