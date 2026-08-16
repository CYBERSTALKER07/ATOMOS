package payout

import (
	"context"
	"errors"
	"fmt"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
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
	// IsLive reports whether Submit(live=true) actually moves money. The file
	// rail is not live; a real bank/payment rail returns true. SubmitForDispatch
	// refuses live=true on a non-live rail so a batch can never be stranded in
	// SUBMITTED with no rail to confirm it.
	IsLive() bool
	// Submit dispatches the batch. live=false renders a file/instruction only
	// and must not move money. Returns a rail reference (empty for file rails).
	Submit(ctx context.Context, b Batch, d SupplierBankDetails, live bool) (ref string, err error)
}

// ErrNoLiveRail rejects a live dispatch when the configured rail cannot move
// money. Prevents batches stranding in SUBMITTED with no settlement webhook.
var ErrNoLiveRail = fmt.Errorf("no live payout rail configured: cannot dispatch live")

// BankFileRail is the default: renders the CSV instruction, no live dispatch.
type BankFileRail struct{}

func (BankFileRail) Name() string { return "bank-file" }

func (BankFileRail) IsLive() bool { return false }

func (BankFileRail) Submit(_ context.Context, b Batch, d SupplierBankDetails, live bool) (string, error) {
	if live {
		return "", ErrNoLiveRail
	}
	if _, err := RenderBankFile(b, d); err != nil {
		return "", err
	}
	return "", nil
}

// namedFileRail is a catalog file rail (SEPA/ACH file) that does not move money.
type namedFileRail struct{ name string }

func (r namedFileRail) Name() string { return r.name }
func (namedFileRail) IsLive() bool   { return false }

func (r namedFileRail) Submit(ctx context.Context, b Batch, d SupplierBankDetails, live bool) (string, error) {
	return BankFileRail{}.Submit(ctx, b, d, live)
}

// railByName resolves a catalog file rail. Unknown names do not fall through
// to bank-file (GS-M6) — live dispatch must not invent a PSP.
func railByName(name string) (Rail, error) {
	switch auth.CanonicalPayoutRail(name) {
	case auth.PayoutRailBankFile:
		return BankFileRail{}, nil
	case auth.PayoutRailSEPAFile:
		return namedFileRail{name: auth.PayoutRailSEPAFile}, nil
	case auth.PayoutRailACHFile:
		return namedFileRail{name: auth.PayoutRailACHFile}, nil
	default:
		return nil, auth.ErrPayoutRailUnknown
	}
}

func (s *Service) resolveRail(ctx context.Context, supplierID string) (Rail, error) {
	if s != nil && s.rail != nil && s.rail.IsLive() {
		return s.rail, nil
	}
	name, err := auth.PayoutRailFromContext(ctx, supplierID)
	if err != nil {
		return nil, err
	}
	return railByName(name)
}

// RailInfo is the honesty surface for clients (G1.D): bank-file is prod-valid only as
// export CSV → bank → MarkPaid, never as silent live money movement.
type RailInfo struct {
	Name     string   `json:"name"`
	IsLive   bool     `json:"is_live"`
	Workflow string   `json:"workflow"`
	Steps    []string `json:"steps,omitempty"`
	Message  string   `json:"message,omitempty"`
}

func fileRailInfo(name string) RailInfo {
	return RailInfo{
		Name:     name,
		IsLive:   false,
		Workflow: "bank_file_export_then_mark_paid",
		Steps:    []string{"generate", "export_csv", "bank_processes_file", "mark_paid"},
		Message:  "Bank-file rail only: export CSV, process at bank, then mark-paid. live dispatch is rejected (no_live_rail).",
	}
}

func liveRailInfo(name string) RailInfo {
	return RailInfo{
		Name:     name,
		IsLive:   true,
		Workflow: "live_dispatch_then_webhook",
		Steps:    []string{"generate", "dispatch_live", "settlement_webhook", "paid"},
		Message:  "Live rail: dispatch moves money; ConfirmSettlement webhook marks PAID",
	}
}

// RailInfoContext returns pack payout_rail honesty (GS-M6). Planned pack fails closed.
func (s *Service) RailInfoContext(ctx context.Context, supplierID string) (RailInfo, error) {
	if s != nil && s.rail != nil && s.rail.IsLive() {
		return liveRailInfo(s.rail.Name()), nil
	}
	name, err := auth.PayoutRailFromContext(ctx, supplierID)
	if err != nil {
		return RailInfo{}, err
	}
	return fileRailInfo(name), nil
}

// RailInfo returns pack rail metadata for the env shipped default.
func (s *Service) RailInfo() RailInfo {
	info, err := s.RailInfoContext(context.Background(), "")
	if err != nil {
		return RailInfo{IsLive: false, Message: err.Error()}
	}
	return info
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
	if _, packErr := auth.PayoutRailFromContext(ctx, b.SupplierID); packErr != nil {
		if live && errors.Is(packErr, auth.ErrPayoutRailUnknown) {
			return Batch{}, ErrNoLiveRail
		}
		return Batch{}, packErr
	}
	rail, err := s.resolveRail(ctx, b.SupplierID)
	if err != nil {
		if live && errors.Is(err, auth.ErrPayoutRailUnknown) {
			return Batch{}, ErrNoLiveRail
		}
		return Batch{}, err
	}
	if live && !rail.IsLive() {
		// Fail closed: never set SUBMITTED on a rail that cannot settle, else the
		// batch strands with an empty RailReference and no webhook to flip it PAID.
		return Batch{}, ErrNoLiveRail
	}
	ref, err := rail.Submit(ctx, b, details, live)
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
