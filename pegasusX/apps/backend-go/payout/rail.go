package payout

import (
	"context"
	"fmt"
	"strings"
)

// Rail is a live settlement transport. Implementations actually move money
// (bank API, payment-rail aggregator) and return a rail reference used to
// reconcile the asynchronous settlement confirmation webhook.
//
// The default is BankFileRail — it produces the CSV payment instruction and
// stays in EXPORTED state until a human marks it PAID. A real rail (e.g. a
// bank API or Global Pay payout) implements Submit to dispatch funds and
// returns a reference; the batch then sits SUBMITTED until ConfirmSettlement
// (webhook) flips it to PAID. Money movement is always two-phase so a crash
// between dispatch and confirm never double-pays: the rail reference is the
// idempotency anchor on the rail side.
type Rail interface {
	// Name identifies the rail ("bank-file", "globalpay-payout", ...).
	Name() string
	// Submit dispatches the batch. live=false renders a file/instruction only
	// and must not move money. Returns a rail reference (empty for file rails).
	Submit(ctx context.Context, b Batch, d SupplierBankDetails, live bool) (ref string, err error)
}

// BankFileRail is the default: renders the CSV instruction, no live dispatch.
type BankFileRail struct{}

func (BankFileRail) Name() string { return "bank-file" }

func (BankFileRail) Submit(_ context.Context, b Batch, d SupplierBankDetails, live bool) (string, error) {
	if _, err := RenderBankFile(b, d); err != nil {
		return "", err
	}
	return "", nil
}

// railByName resolves the configured rail. Unknown names fail closed to the
// bank-file rail — a misconfigured payout rail must never silently attempt a
// live dispatch.
func railByName(name string) Rail {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "bank-file", "csv":
		return BankFileRail{}
	default:
		// No live rail is implemented yet; fail closed to the file rail.
		return BankFileRail{}
	}
}

// SubmitForDispatch validates beneficiary details and submits the batch via the
// rail. live=false (default) is a dry-run that only validates + renders.
func (s *Service) SubmitForDispatch(ctx context.Context, batchID string, live bool) (Batch, error) {
	if s == nil || s.repo == nil {
		return Batch{}, fmt.Errorf("payout service unavailable")
	}
	b, found, err := s.repo.Get(ctx, batchID)
	if err != nil {
		return Batch{}, err
	}
	if !found {
		return Batch{}, ErrBatchNotFound
	}
	if b.Status == StatusPaid {
		return Batch{}, fmt.Errorf("batch %s already PAID", batchID)
	}
	if b.Status != StatusDraft && b.Status != StatusExported {
		return Batch{}, fmt.Errorf("batch %s not dispatchable from status %s", batchID, b.Status)
	}
	details, err := s.repo.SupplierBankDetails(ctx, b.SupplierID)
	if err != nil {
		return Batch{}, err
	}
	ref, err := s.rail.Submit(ctx, b, details, live)
	if err != nil {
		return Batch{}, err
	}
	next := StatusExported
	if live {
		next = StatusSubmitted
	}
	if err := s.repo.UpdateStatusRef(ctx, b.BatchID, next, "", ref); err != nil {
		return Batch{}, err
	}
	b.Status = next
	return b, nil
}

// ConfirmSettlement is the webhook entry point: the rail confirms settlement
// of a SUBMITTED batch, flipping it to PAID. Idempotent — confirming an
// already-PAID batch is a no-op. Exported (file-rail) batches use MarkPaid.
func (s *Service) ConfirmSettlement(ctx context.Context, batchID, railRef string) error {
	if s == nil || s.repo == nil {
		return fmt.Errorf("payout service unavailable")
	}
	b, found, err := s.repo.Get(ctx, batchID)
	if err != nil {
		return err
	}
	if !found {
		return ErrBatchNotFound
	}
	if b.Status == StatusPaid {
		return nil // idempotent replay
	}
	if b.Status != StatusSubmitted {
		return fmt.Errorf("batch %s must be SUBMITTED before settlement confirm (current %s)", batchID, b.Status)
	}
	return s.repo.UpdateStatusRef(ctx, batchID, StatusPaid, "", railRef)
}
