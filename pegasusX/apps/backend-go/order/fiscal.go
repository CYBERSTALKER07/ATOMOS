package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/notifications"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/soliq"
	"github.com/pegasusx/pegasusx/apps/backend-go/tax"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// Fiscal attempt / order denorm statuses (app-enforced).
const (
	FiscalAttemptPending      = "PENDING"
	FiscalAttemptSuccess      = "SUCCESS"
	FiscalAttemptFailed       = "FAILED"
	FiscalAttemptForceSkipped = "FORCE_SKIPPED"

	FiscalStatusNone              = "NONE"
	FiscalStatusPending           = "PENDING"
	FiscalStatusSuccess           = "SUCCESS"
	FiscalStatusFailed            = "FAILED"
	FiscalStatusForceSkipped      = "FORCE_SKIPPED"
	FiscalStatusHistoricalSkipped = "HISTORICAL_SKIPPED"
	FiscalStatusDeadLetter        = "DEAD_LETTER"

	FiscalProviderFake      = "FAKE"
	FiscalProviderPegasus   = "PEGASUS"    // platform commercial receipt (default product path)
	FiscalProviderGlobalPay = "GLOBAL_PAY" // payment-provider receipt (optional secondary)
	FiscalProviderMySoliq   = "MY_SOLIQ"   // tax OFD — deferred until Soliq sandbox creds

	// BuyerAcceptancePending / Accepted / Rejected / Expired are the Soliq EHF
	// buyer-clearance states (parallel to ADR-009 COMPLETED).
	BuyerAcceptancePending  = "PENDING"
	BuyerAcceptanceAccepted = "ACCEPTED"
	BuyerAcceptanceRejected = "REJECTED"
	BuyerAcceptanceExpired  = "EXPIRED"

	// BuyerAcceptanceWindowDefault is the UZ EHF buyer clearance window (10 days
	// per soliq-ehf-integration.md). Overridable via BUYER_ACCEPTANCE_DAYS.
	BuyerAcceptanceWindowDefault = 10 * 24 * time.Hour

	// FiscalOFDTimeout aligns with Soliq client outbound timeout per external receipt call (P0 T8/T9).
	FiscalOFDTimeout = 30 * time.Second
	// FiscalMaxFailedAttempts before order stays in FISCAL_FAILED requiring retry/force.
	FiscalMaxFailedAttempts = 3
)

// Force-complete reason codes (audited enum — ADR-009 Phase 4).
const (
	ForceReasonOFDDown       = "OFD_DOWN"
	ForceReasonOFDTimeout    = "OFD_TIMEOUT"
	ForceReasonOpsEscalation = "OPS_ESCALATION"
	ForceReasonTaxExempt     = "TAX_EXEMPT_POLICY"
	ForceReasonOther         = "OTHER"
)

// ValidForceReasonCodes is the closed set accepted by force-complete.
var ValidForceReasonCodes = map[string]struct{}{
	ForceReasonOFDDown:       {},
	ForceReasonOFDTimeout:    {},
	ForceReasonOpsEscalation: {},
	ForceReasonTaxExempt:     {},
	ForceReasonOther:         {},
}

func NormalizeForceReasonCode(code string) (string, error) {
	c := strings.ToUpper(strings.TrimSpace(code))
	if c == "" {
		return "", ErrForceReasonRequired
	}
	if _, ok := ValidForceReasonCodes[c]; !ok {
		return "", fmt.Errorf("%w: %s", ErrForceReasonInvalid, code)
	}
	return c, nil
}

// isTerminalMoneyStatus blocks late webhooks inventing new money (P0 T7).
func isTerminalMoneyStatus(st Status) bool {
	switch st {
	case StatusCompleted, StatusCancelled, StatusReconciliationRequired:
		return true
	default:
		return false
	}
}

// FiscalReceiptRow is one immutable OFD attempt (supplier-scoped leg).
type FiscalReceiptRow struct {
	OrderID         string
	AttemptID       string
	SupplierID      string
	RetailerID      string
	Provider        string
	Status          string
	FiscalReceiptID string
	FiscalQR        string
	AmountMinor     int64
	Currency        string
	PaymentMethod   string
	ProviderPayload []byte
	ErrorCode       string
	ErrorMessage    string
	ReasonCode      string
	ActorID         string
	TraceID         string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// FiscalProvider calls the OFD / tax receipt API asynchronously from a worker.
type FiscalProvider interface {
	CreateReceipt(ctx context.Context, req FiscalCreateRequest) (FiscalCreateResult, error)
	GetSoliqClient() soliq.SoliqClient
}

// FiscalCreateRequest is the provider input (integer Tiyin only).
type FiscalCreateRequest struct {
	AttemptID     string
	OrderID       string
	SupplierID    string
	RetailerID    string
	AmountMinor   int64
	Currency      string
	PaymentMethod string
	LineItems     []LineItem
}

// FiscalCreateResult is a successful OFD response.
type FiscalCreateResult struct {
	FiscalReceiptID string
	FiscalQR        string
	RawPayload      []byte
}

// FiscalCorrectiveRequest issues a corrective (credit-note) receipt that
// references the original fiscal receipt — UZ practice for refunds/returns.
type FiscalCorrectiveRequest struct {
	AttemptID         string
	OrderID           string
	SupplierID        string
	RetailerID        string
	OriginalReceiptID string
	AmountMinor       int64
	Currency          string
	ReasonCode        string
}

// CorrectiveFiscalProvider is an optional extension of FiscalProvider for the
// refund corrective chain. Providers that cannot issue corrective receipts do
// not implement it; callers type-assert and record the gap instead of faking.
type CorrectiveFiscalProvider interface {
	CreateCorrectiveReceipt(ctx context.Context, req FiscalCorrectiveRequest) (FiscalCreateResult, error)
}

func defaultFiscalProvider() FiscalProvider {
	return ProviderFromEnv()
}

// requireFiscalPack fails closed when the shipped pack cannot fiscalize (GS-M2).
// Pack fiscal_adapter is the law (PEPPOL/planned → unimplemented). Cell runtime
// PEGASUS/FAKE is still allowed outside production — that disagreement keeps
// checkout_reads_this false.
func (s *Service) requireFiscalPack(ctx context.Context, supplierID string) error {
	pack, err := auth.FiscalPackFromContext(ctx, supplierID)
	if err != nil {
		return err
	}
	if _, err := auth.PackFiscalAdapter(pack); err != nil {
		return err
	}
	runtime := FiscalProviderPegasus
	if s != nil {
		runtime = s.ProviderName()
	}
	return auth.AssertFiscalRuntime(pack, runtime)
}

func (s *Service) fiscalProvider() FiscalProvider {
	if s != nil && s.ofd != nil {
		return s.ofd
	}
	return defaultFiscalProvider()
}

// SetFiscalProvider injects an OFD adapter (tests / bootstrap).
func (s *Service) SetFiscalProvider(p FiscalProvider) {
	if s == nil {
		return
	}
	s.ofd = p
}

// FiscalSuccessNotifier asynchronously notifies the retailer (SMS / Telegram)
// when a fiscal receipt completes and carries a Soliq/OFD QR URL.
type FiscalSuccessNotifier interface {
	NotifyFiscalReceiptQR(ctx context.Context, orderRecord Order, receipt FiscalReceiptRow) error
}

// SetFiscalNotifier overrides the fiscal receipt SMS/Telegram dispatcher.
func (s *Service) SetFiscalNotifier(n FiscalSuccessNotifier) {
	if s == nil {
		return
	}
	s.fiscalNotifier = n
}

// FlushFiscalNotifications waits for any in-flight async fiscal receipt deliveries.
func (s *Service) FlushFiscalNotifications() {
	if s == nil {
		return
	}
	s.fiscalNotifyWG.Wait()
}

// ProviderName returns the active provider label for attempt rows.
func (s *Service) ProviderName() string {
	p := s.fiscalProvider()
	switch v := p.(type) {
	case *MySoliqProvider:
		return FiscalProviderMySoliq
	case *GlobalPayReceiptProvider:
		return FiscalProviderGlobalPay
	case PegasusReceiptProvider:
		return FiscalProviderPegasus
	case FakeFiscalProvider:
		return FiscalProviderFake
	case multiReceiptProvider:
		if v.name != "" {
			return v.name
		}
		return FiscalProviderPegasus
	default:
		// hardFailProvider and unknown → report resolved intent (G1.B defaults).
		return ResolveFiscalProviderName()
	}
}

// newFiscalPendingRow builds a PENDING supplier-leg attempt for capture txn.
// amountMinor is the fiscalized cash/card amount (received cash or order total).
func (s *Service) newFiscalPendingRow(ctx context.Context, orderRecord Order, paymentMethod, attemptID string, amountMinor int64) FiscalReceiptRow {
	now := time.Now().UTC()
	if s != nil && s.now != nil {
		now = s.now().UTC()
	}
	if attemptID == "" {
		if s != nil && s.newID != nil {
			attemptID = s.newID()
		} else {
			attemptID = defaultOrderID()
		}
	}
	currency := strings.TrimSpace(orderRecord.Currency)
	if currency == "" {
		if cur, err := auth.CurrencyFromContext(ctx, orderRecord.SupplierID); err == nil {
			currency = cur
		}
	}
	if amountMinor < 0 {
		amountMinor = 0
	}
	if amountMinor == 0 {
		amountMinor = orderRecord.TotalMinor
	}
	provider := FiscalProviderPegasus
	if s != nil {
		provider = s.ProviderName()
	}
	return FiscalReceiptRow{
		OrderID:       orderRecord.OrderID,
		AttemptID:     attemptID,
		SupplierID:    orderRecord.SupplierID,
		RetailerID:    orderRecord.RetailerID,
		Provider:      provider,
		Status:        FiscalAttemptPending,
		AmountMinor:   amountMinor,
		Currency:      currency,
		PaymentMethod: paymentMethod,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func emitCashVariance(ctx context.Context, txn outbox.TxnBuffer, orderRecord Order, driverID string, expected, received int64, note string) error {
	currency := strings.TrimSpace(orderRecord.Currency)
	if currency == "" {
		if cur, err := auth.CurrencyFromContext(ctx, orderRecord.SupplierID); err == nil {
			currency = cur
		}
	}
	shortfall := expected - received
	if shortfall < 0 {
		shortfall = 0
	}
	overage := received - expected
	if overage < 0 {
		overage = 0
	}
	if shortfall == 0 && overage == 0 {
		return nil
	}
	eventType := events.EventCashShortfall
	if overage > 0 {
		eventType = events.EventCashOverage
	}
	ts := orderRecord.UpdatedAt
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	return outbox.EmitJSON(ctx, txn, events.AggregateOrder, orderRecord.OrderID, events.TopicMain, events.CashVarianceEvent{
		BaseEvent:      events.BaseEvent{Type: eventType, Timestamp: ts.Format(time.RFC3339Nano)},
		OrderID:        orderRecord.OrderID,
		SupplierID:     orderRecord.SupplierID,
		RetailerID:     orderRecord.RetailerID,
		DriverID:       driverID,
		ExpectedMinor:  expected,
		ReceivedMinor:  received,
		ShortfallMinor: shortfall,
		OverageMinor:   overage,
		Currency:       currency,
		Note:           strings.TrimSpace(note),
	})
}

func emitFiscalReceiptRequested(ctx context.Context, txn outbox.TxnBuffer, row FiscalReceiptRow) error {
	return outbox.EmitJSON(ctx, txn, events.AggregateOrder, row.OrderID, events.TopicMain, events.FiscalReceiptEvent{
		BaseEvent:     events.BaseEvent{Type: events.EventFiscalReceiptRequested, Timestamp: row.CreatedAt.Format(time.RFC3339Nano)},
		OrderID:       row.OrderID,
		AttemptID:     row.AttemptID,
		SupplierID:    row.SupplierID,
		RetailerID:    row.RetailerID,
		AmountMinor:   row.AmountMinor,
		Currency:      row.Currency,
		PaymentMethod: row.PaymentMethod,
		Provider:      row.Provider,
		Status:        row.Status,
		TraceID:       row.TraceID,
	})
}

func emitFiscalReceiptSucceeded(ctx context.Context, txn outbox.TxnBuffer, row FiscalReceiptRow) error {
	return outbox.EmitJSON(ctx, txn, events.AggregateOrder, row.OrderID, events.TopicMain, events.FiscalReceiptEvent{
		BaseEvent:       events.BaseEvent{Type: events.EventFiscalReceiptSucceeded, Timestamp: row.UpdatedAt.Format(time.RFC3339Nano)},
		OrderID:         row.OrderID,
		AttemptID:       row.AttemptID,
		SupplierID:      row.SupplierID,
		RetailerID:      row.RetailerID,
		AmountMinor:     row.AmountMinor,
		Currency:        row.Currency,
		PaymentMethod:   row.PaymentMethod,
		Provider:        row.Provider,
		Status:          row.Status,
		FiscalReceiptID: row.FiscalReceiptID,
		FiscalQR:        row.FiscalQR,
		TraceID:         row.TraceID,
	})
}

func emitFiscalReceiptFailed(ctx context.Context, txn outbox.TxnBuffer, row FiscalReceiptRow) error {
	return outbox.EmitJSON(ctx, txn, events.AggregateOrder, row.OrderID, events.TopicMain, events.FiscalReceiptEvent{
		BaseEvent:     events.BaseEvent{Type: events.EventFiscalReceiptFailed, Timestamp: row.UpdatedAt.Format(time.RFC3339Nano)},
		OrderID:       row.OrderID,
		AttemptID:     row.AttemptID,
		SupplierID:    row.SupplierID,
		RetailerID:    row.RetailerID,
		AmountMinor:   row.AmountMinor,
		Currency:      row.Currency,
		PaymentMethod: row.PaymentMethod,
		Provider:      row.Provider,
		Status:        row.Status,
		ErrorCode:     row.ErrorCode,
		ErrorMessage:  row.ErrorMessage,
		TraceID:       row.TraceID,
	})
}

func emitOrderForceCompleted(ctx context.Context, txn outbox.TxnBuffer, orderRecord Order, reasonCode, actorID string) error {
	return outbox.EmitJSON(ctx, txn, events.AggregateOrder, orderRecord.OrderID, events.TopicMain, events.OrderForceCompletedEvent{
		BaseEvent:  events.BaseEvent{Type: events.EventOrderForceCompleted, Timestamp: orderRecord.UpdatedAt.Format(time.RFC3339Nano)},
		OrderID:    orderRecord.OrderID,
		SupplierID: orderRecord.SupplierID,
		RetailerID: orderRecord.RetailerID,
		ReasonCode: reasonCode,
		ActorID:    actorID,
	})
}

func emitBuyerAcceptance(ctx context.Context, txn outbox.TxnBuffer, orderRecord Order, eventType, status string) error {
	deadline := ""
	if orderRecord.BuyerAcceptanceDeadline != nil {
		deadline = orderRecord.BuyerAcceptanceDeadline.UTC().Format(time.RFC3339Nano)
	}
	ts := orderRecord.UpdatedAt
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	return outbox.EmitJSON(ctx, txn, events.AggregateOrder, orderRecord.OrderID, events.TopicMain, events.BuyerAcceptanceEvent{
		BaseEvent:  events.BaseEvent{Type: eventType, Timestamp: ts.Format(time.RFC3339Nano)},
		OrderID:    orderRecord.OrderID,
		SupplierID: orderRecord.SupplierID,
		RetailerID: orderRecord.RetailerID,
		EhfID:      orderRecord.LatestFiscalReceiptID,
		Status:     status,
		Deadline:   deadline,
	})
}

// stampBuyerAcceptancePending marks the order for the Soliq EHF buyer-clearance
// poller. Only applies to MY_SOLIQ (real EHF); PEGASUS/FAKE commercial receipts
// have no buyer-acceptance window. ADR-009 still completes the order; this is
// the parallel track that gates reverse-settlement on REJECT.
func stampBuyerAcceptancePending(orderRecord *Order, provider string, now time.Time) bool {
	if orderRecord == nil || provider != FiscalProviderMySoliq {
		return false
	}
	if strings.TrimSpace(orderRecord.BuyerAcceptanceStatus) != "" &&
		orderRecord.BuyerAcceptanceStatus != BuyerAcceptancePending {
		// Already resolved (ACCEPTED/REJECTED/EXPIRED) — do not reopen.
		return false
	}
	orderRecord.BuyerAcceptanceStatus = BuyerAcceptancePending
	deadline := now.Add(buyerAcceptanceWindow())
	orderRecord.BuyerAcceptanceDeadline = &deadline
	return true
}

func buyerAcceptanceWindow() time.Duration {
	raw := strings.TrimSpace(os.Getenv("BUYER_ACCEPTANCE_DAYS"))
	if raw == "" {
		return BuyerAcceptanceWindowDefault
	}
	days, err := strconv.Atoi(raw)
	if err != nil || days <= 0 {
		return BuyerAcceptanceWindowDefault
	}
	return time.Duration(days) * 24 * time.Hour
}

// emitPaymentCaptureFiscal wires PAYMENT_CLEARED + FISCAL_RECEIPT_REQUESTED for a capture txn.
// Does NOT emit ORDER_FINALIZED (ADR-009).
func emitPaymentCaptureFiscal(ctx context.Context, txn outbox.TxnBuffer, orderRecord Order, row FiscalReceiptRow, paymentMethod string) error {
	if err := emitPaymentCleared(ctx, txn, orderRecord, paymentMethod); err != nil {
		return err
	}
	return emitFiscalReceiptRequested(ctx, txn, row)
}

// ApplyFiscalWorkerResult is invoked by the order event consumer on FISCAL_RECEIPT_REQUESTED.
// P0 guarantees:
//   - Idempotent: SUCCESS attempt never re-calls OFD
//   - Hard timeout per OFD call
//   - After max failed attempts, order is FISCAL_FAILED
func (s *Service) ApplyFiscalWorkerResult(ctx context.Context, orderID, attemptID string) error {
	orderID = strings.TrimSpace(orderID)
	attemptID = strings.TrimSpace(attemptID)
	if orderID == "" || attemptID == "" {
		return nil
	}

	orderRecord, ok, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrOrderNotFound
	}
	if orderRecord.Status == StatusCompleted {
		return nil // terminal — never re-fiscalize
	}
	if orderRecord.Status != StatusFiscalizing && orderRecord.Status != StatusFiscalFailed {
		s.log.InfoContext(ctx, "skip fiscal worker; order not fiscalizing",
			"order_id", orderID, "status", orderRecord.Status)
		return nil
	}

	// Idempotency: if attempt already SUCCESS, complete order if needed without OFD.
	if existing, found, gErr := s.repo.GetFiscalAttempt(ctx, orderID, attemptID); gErr == nil && found {
		if existing.Status == FiscalAttemptSuccess {
			if orderRecord.Status != StatusCompleted {
				return s.completeOrderFromExistingFiscalSuccess(ctx, orderRecord, existing)
			}
			return nil
		}
		if existing.Status == FiscalAttemptForceSkipped {
			return nil
		}
	}

	amountMinor := orderRecord.TotalMinor
	payMethod := "CASH"
	if existing, found, gErr := s.repo.GetFiscalAttempt(ctx, orderID, attemptID); gErr == nil && found {
		if existing.AmountMinor > 0 {
			amountMinor = existing.AmountMinor
		}
		if existing.PaymentMethod != "" {
			payMethod = existing.PaymentMethod
		}
	}

	req := FiscalCreateRequest{
		AttemptID:     attemptID,
		OrderID:       orderRecord.OrderID,
		SupplierID:    orderRecord.SupplierID,
		RetailerID:    orderRecord.RetailerID,
		AmountMinor:   amountMinor,
		Currency:      orderRecord.Currency,
		PaymentMethod: payMethod,
		LineItems:     orderRecord.LineItems,
	}
	if req.Currency == "" {
		if cur, err := auth.CurrencyFromContext(ctx, orderRecord.SupplierID); err == nil {
			req.Currency = cur
		}
	}

	var result FiscalCreateResult
	var provErr error
	if packErr := s.requireFiscalPack(ctx, orderRecord.SupplierID); packErr != nil {
		provErr = packErr
	} else {
		ofdCtx, cancel := context.WithTimeout(ctx, FiscalOFDTimeout)
		result, provErr = s.fiscalProvider().CreateReceipt(ofdCtx, req)
		if provErr == nil && ofdCtx.Err() != nil {
			provErr = ofdCtx.Err()
		}
		if provErr != nil && errors.Is(ofdCtx.Err(), context.DeadlineExceeded) {
			provErr = fmt.Errorf("ofd_timeout: %w", ofdCtx.Err())
		}
		cancel()
	}

	now := s.now().UTC()
	previousStatus := orderRecord.Status

	row := FiscalReceiptRow{
		OrderID:       orderRecord.OrderID,
		AttemptID:     attemptID,
		SupplierID:    orderRecord.SupplierID,
		RetailerID:    orderRecord.RetailerID,
		Provider:      s.ProviderName(),
		AmountMinor:   amountMinor,
		Currency:      req.Currency,
		PaymentMethod: req.PaymentMethod,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if provErr != nil {
		row.Status = FiscalAttemptFailed
		if strings.Contains(provErr.Error(), "ofd_timeout") || errors.Is(provErr, context.DeadlineExceeded) {
			row.ErrorCode = "OFD_TIMEOUT"
		} else {
			row.ErrorCode = "OFD_ERROR"
		}
		row.ErrorMessage = provErr.Error()
		if err := ValidateStatusTransition(string(previousStatus), string(StatusFiscalFailed), TransitionOpts{}); err != nil {
			return err
		}
		orderRecord.Status = StatusFiscalFailed
		orderRecord.FiscalStatus = FiscalStatusFailed
		orderRecord.LatestFiscalAttemptID = attemptID
		orderRecord.UpdatedAt = now
		orderRecord.FiscalReceiptUpdate = &row

		ticketPayloadBytes, marshalErr := json.Marshal(map[string]any{
			"order_id":            orderRecord.OrderID,
			"supplier_id":         orderRecord.SupplierID,
			"retailer_id":         orderRecord.RetailerID,
			"attempt_id":          attemptID,
			"error_code":          row.ErrorCode,
			"money_at_risk_minor": amountMinor,
			"handler_group":       "FISCAL_AND_EXPORT_BLOCKERS",
			"retry_queued":        true,
		})
		if marshalErr != nil {
			return fmt.Errorf("marshal fiscal exception payload: %w", marshalErr)
		}
		orderRecord.PendingExceptionTicket = &ExceptionTicket{
			TicketID:     uuid.NewString(),
			Type:         "FISCAL_FAILED",
			OrderID:      orderRecord.OrderID,
			Severity:     "CRITICAL",
			Status:       "OPEN",
			Title:        "Fiscal receipt queued for retry",
			Description:  row.ErrorMessage,
			AssignedRole: "FINANCE",
			CreatedAt:    now,
			CreatedBy:    "system:fiscal-worker",
			Payload:      ticketPayloadBytes,
		}

		err = s.repo.UpdateOrderWithTxn(ctx, orderRecord, nil, nil, func(txn outbox.TxnBuffer) error {
			if err := emitOrderStatusChanged(ctx, txn, orderStatusEmitParams{
				Order:          orderRecord,
				PreviousStatus: previousStatus,
				Reason:         "fiscal_failed",
			}); err != nil {
				return err
			}
			return emitFiscalReceiptFailed(ctx, txn, row)
		})
		if err != nil {
			return fmt.Errorf("persist fiscal failure: %w", err)
		}
		s.afterOrderMutation(ctx, orderRecord)
		return nil
	}

	row.Status = FiscalAttemptSuccess
	row.FiscalReceiptID = result.FiscalReceiptID
	row.FiscalQR = result.FiscalQR
	row.ProviderPayload = result.RawPayload
	if err := ValidateStatusTransition(string(previousStatus), string(StatusCompleted), TransitionOpts{}); err != nil {
		return err
	}
	orderRecord.Status = StatusCompleted
	orderRecord.FiscalStatus = FiscalStatusSuccess
	orderRecord.LatestFiscalReceiptID = result.FiscalReceiptID
	orderRecord.LatestFiscalAttemptID = attemptID
	orderRecord.FiscalizedAt = &now
	orderRecord.UpdatedAt = now
	orderRecord.FiscalReceiptUpdate = &row
	buyerPending := stampBuyerAcceptancePending(&orderRecord, s.ProviderName(), now)
	_ = s.ApplyClaimWindowSnapshot(ctx, &orderRecord, now)

	preFee, feeErr := s.resolvePerOrderFeeBeforeTxn(ctx, orderRecord.SupplierID)
	if feeErr != nil {
		return fmt.Errorf("resolve per-order fee: %w", feeErr)
	}
	inTxn := func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return s.completeOrderTaxAndGLInTxn(ctx, txn, nil, &orderRecord, preFee, "")
	}

	err = s.repo.UpdateOrderWithTxn(ctx, orderRecord, nil, inTxn, func(txn outbox.TxnBuffer) error {
		if err := emitOrderStatusChanged(ctx, txn, orderStatusEmitParams{
			Order:          orderRecord,
			PreviousStatus: previousStatus,
			Reason:         "fiscal_succeeded",
		}); err != nil {
			return err
		}
		if err := emitFiscalReceiptSucceeded(ctx, txn, row); err != nil {
			return err
		}
		if buyerPending {
			if err := emitBuyerAcceptance(ctx, txn, orderRecord, events.EventBuyerAcceptancePending, BuyerAcceptancePending); err != nil {
				return err
			}
		}
		return emitOrderFinalized(ctx, txn, orderRecord)
	})
	if err != nil {
		return fmt.Errorf("persist fiscal success: %w", err)
	}
	s.afterOrderMutation(ctx, orderRecord)
	s.notifyFiscalSuccessAsync(ctx, orderRecord, row)
	if orderRecord.ManifestID != "" {
		if mErr := s.tryCompleteManifest(ctx, orderRecord.ManifestID); mErr != nil {
			s.log.ErrorContext(ctx, "manifest complete after fiscal failed", "manifest_id", orderRecord.ManifestID, "err", mErr)
		}
	}
	return nil
}

// SubmitFiscalReceipt executes (or retries) fiscal receipt submission for an order
// and asynchronously delivers SMS/Telegram QR notifications upon OFD success.
func (s *Service) SubmitFiscalReceipt(ctx context.Context, claims auth.Claims, orderID, attemptID string) (CollectCashResponse, error) {
	orderID = strings.TrimSpace(orderID)
	attemptID = strings.TrimSpace(attemptID)
	if orderID == "" {
		return CollectCashResponse{}, errors.New("order_id required")
	}
	orderRecord, ok, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return CollectCashResponse{}, err
	}
	if !ok {
		return CollectCashResponse{}, ErrOrderNotFound
	}
	if isMigrationFiscalExcluded(orderRecord) {
		return CollectCashResponse{}, fmt.Errorf("%w: migrated order excluded from fiscalization", ErrFiscalAlreadySucceeded)
	}
	if attemptID == "" {
		if orderRecord.Status == StatusFiscalFailed {
			resp, rErr := s.RetryFiscal(ctx, claims, orderID)
			if rErr != nil {
				return CollectCashResponse{}, rErr
			}
			attemptID = resp.AttemptID
		} else {
			attemptID = strings.TrimSpace(orderRecord.LatestFiscalAttemptID)
		}
	}
	if attemptID == "" {
		return CollectCashResponse{}, errors.New("fiscal attempt_id required")
	}
	if wErr := s.ApplyFiscalWorkerResult(ctx, orderID, attemptID); wErr != nil {
		return CollectCashResponse{}, wErr
	}
	updated, _, gErr := s.repo.GetOrder(ctx, orderID)
	if gErr != nil {
		return CollectCashResponse{}, gErr
	}
	return CollectCashResponse{
		OrderID:   updated.OrderID,
		State:     updated.Status,
		Amount:    updated.TotalMinor,
		Currency:  updated.Currency,
		AttemptID: attemptID,
		Message:   "Fiscal receipt submitted.",
	}, nil
}

func (s *Service) notifyFiscalSuccessAsync(ctx context.Context, orderRecord Order, receipt FiscalReceiptRow) {
	if s == nil {
		return
	}
	notifier := s.fiscalNotifier
	if notifier == nil {
		if s.spannerClient == nil {
			return
		}
		notifier = newDefaultFiscalSuccessNotifier(s.spannerClient, s.log)
	}
	baseCtx := context.WithoutCancel(ctx)
	s.fiscalNotifyWG.Add(1)
	go func() {
		defer s.fiscalNotifyWG.Done()
		callCtx, cancel := context.WithTimeout(baseCtx, 10*time.Second)
		defer cancel()
		if err := notifier.NotifyFiscalReceiptQR(callCtx, orderRecord, receipt); err != nil && s.log != nil {
			s.log.WarnContext(callCtx, "fiscal qr retailer notification failed",
				"order_id", orderRecord.OrderID,
				"receipt_id", receipt.FiscalReceiptID,
				"err", err)
		}
	}()
}

// DefaultFiscalSuccessNotifier resolves retailer phone/Telegram chat from Spanner
// and delivers localized whole-UZS SMS + Telegram QR receipt messages.
type DefaultFiscalSuccessNotifier struct {
	Spanner  *spanner.Client
	SMS      *notifications.SMSTransport
	Telegram *notifications.TelegramBotSender
	Log      *slog.Logger
}

func newDefaultFiscalSuccessNotifier(client *spanner.Client, log *slog.Logger) *DefaultFiscalSuccessNotifier {
	smsProvider := strings.TrimSpace(os.Getenv("SMS_PROVIDER"))
	var sms *notifications.SMSTransport
	if smsProvider != "" {
		sms = &notifications.SMSTransport{
			Provider:   smsProvider,
			BaseURL:    os.Getenv("PLAYMOBILE_BASE_URL"),
			Login:      os.Getenv("PLAYMOBILE_LOGIN"),
			Password:   os.Getenv("PLAYMOBILE_PASSWORD"),
			Originator: os.Getenv("PLAYMOBILE_ORIGINATOR"),
			AccountSID: os.Getenv("TWILIO_ACCOUNT_SID"),
			AuthToken:  os.Getenv("TWILIO_AUTH_TOKEN"),
			FromNumber: os.Getenv("TWILIO_FROM"),
			Log:        log,
		}
	}
	return &DefaultFiscalSuccessNotifier{
		Spanner:  client,
		SMS:      sms,
		Telegram: notifications.NewTelegramBotSenderFromEnv(log),
		Log:      log,
	}
}

// NotifyFiscalReceiptQR delivers localized whole-so'm fiscal receipt notifications
// via SMS (PlayMobile/Twilio) and verified Telegram links when available.
func (n *DefaultFiscalSuccessNotifier) NotifyFiscalReceiptQR(ctx context.Context, orderRecord Order, receipt FiscalReceiptRow) error {
	if n == nil {
		return nil
	}
	phone := ""
	country := ""
	locale := ""
	chatID := ""

	if n.Spanner != nil && strings.TrimSpace(orderRecord.RetailerID) != "" {
		var rPhone, rCountry spanner.NullString
		row, err := n.Spanner.Single().ReadRow(ctx, "Retailers", spanner.Key{orderRecord.RetailerID}, []string{"Phone", "CountryCode"})
		if err == nil {
			if colErr := row.Columns(&rPhone, &rCountry); colErr == nil {
				phone = strings.TrimSpace(rPhone.StringVal)
				country = strings.TrimSpace(rCountry.StringVal)
			}
		}

		tgIter := n.Spanner.Single().Query(ctx, spanner.Statement{
			SQL: `SELECT TelegramChatId, Locale
			      FROM RetailerTelegramLinks
			      WHERE RetailerId = @rid AND VerifiedAt IS NOT NULL AND TelegramChatId IS NOT NULL
			      ORDER BY VerifiedAt DESC
			      LIMIT 1`,
			Params: map[string]any{"rid": orderRecord.RetailerID},
		})
		defer tgIter.Stop()
		if tgRow, tgErr := tgIter.Next(); tgErr == nil {
			var cID, loc spanner.NullString
			if colErr := tgRow.Columns(&cID, &loc); colErr == nil {
				chatID = strings.TrimSpace(cID.StringVal)
				locale = strings.TrimSpace(loc.StringVal)
			}
		} else if tgErr != iterator.Done && n.Log != nil {
			n.Log.DebugContext(ctx, "lookup telegram link for fiscal qr", "retailer_id", orderRecord.RetailerID, "err", tgErr)
		}
	}

	if locale == "" {
		if strings.EqualFold(country, "RU") || strings.EqualFold(country, "KZ") {
			locale = "ru"
		} else {
			locale = "uz"
		}
	}

	notif := notifications.FormatFiscalReceiptQR(
		orderRecord.OrderID,
		receipt.FiscalReceiptID,
		receipt.FiscalQR,
		receipt.AmountMinor,
		orderRecord.Currency,
		locale,
	)

	var firstErr error
	if n.SMS != nil && phone != "" {
		if err := n.SMS.Deliver(ctx, phone, "RETAILER", notif); err != nil {
			firstErr = err
		}
	}
	if n.Telegram != nil && chatID != "" {
		if err := n.Telegram.SendFiscalReceiptQR(ctx, chatID, locale, notif, receipt.FiscalQR); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (s *Service) completeOrderFromExistingFiscalSuccess(ctx context.Context, orderRecord Order, existing FiscalReceiptRow) error {
	if orderRecord.Status == StatusCompleted {
		return nil
	}
	now := s.now().UTC()
	previousStatus := orderRecord.Status
	if err := ValidateStatusTransition(string(previousStatus), string(StatusCompleted), TransitionOpts{}); err != nil {
		return err
	}
	orderRecord.Status = StatusCompleted
	orderRecord.FiscalStatus = FiscalStatusSuccess
	orderRecord.LatestFiscalReceiptID = existing.FiscalReceiptID
	orderRecord.LatestFiscalAttemptID = existing.AttemptID
	orderRecord.FiscalizedAt = &now
	orderRecord.UpdatedAt = now
	buyerPending := stampBuyerAcceptancePending(&orderRecord, s.ProviderName(), now)
	_ = s.ApplyClaimWindowSnapshot(ctx, &orderRecord, now)

	preFee, feeErr := s.resolvePerOrderFeeBeforeTxn(ctx, orderRecord.SupplierID)
	if feeErr != nil {
		return fmt.Errorf("resolve per-order fee: %w", feeErr)
	}
	inTxn := func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return s.completeOrderTaxAndGLInTxn(ctx, txn, nil, &orderRecord, preFee, "")
	}

	err := s.repo.UpdateOrderWithTxn(ctx, orderRecord, nil, inTxn, func(txn outbox.TxnBuffer) error {
		if err := emitOrderStatusChanged(ctx, txn, orderStatusEmitParams{
			Order:          orderRecord,
			PreviousStatus: previousStatus,
			Reason:         "fiscal_succeeded_idempotent",
		}); err != nil {
			return err
		}
		if buyerPending {
			if err := emitBuyerAcceptance(ctx, txn, orderRecord, events.EventBuyerAcceptancePending, BuyerAcceptancePending); err != nil {
				return err
			}
		}
		return emitOrderFinalized(ctx, txn, orderRecord)
	})
	if err != nil {
		return err
	}
	s.afterOrderMutation(ctx, orderRecord)
	return nil
}

// RetryFiscal creates a new attempt when order is FISCAL_FAILED.
func (s *Service) RetryFiscal(ctx context.Context, claims auth.Claims, orderID string) (CollectCashResponse, error) {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return CollectCashResponse{}, errors.New("order_id required")
	}
	switch claims.Role {
	case auth.RoleDriver, auth.RoleAdmin, auth.RoleWarehouseAdmin:
	default:
		return CollectCashResponse{}, ErrOrderForbidden
	}

	orderRecord, ok, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return CollectCashResponse{}, err
	}
	if !ok {
		return CollectCashResponse{}, ErrOrderNotFound
	}
	if isMigrationFiscalExcluded(orderRecord) {
		return CollectCashResponse{}, fmt.Errorf("%w: migrated order excluded from fiscalization", ErrFiscalAlreadySucceeded)
	}
	isStuckFiscalizing := orderRecord.Status == StatusFiscalizing && s.now().Sub(orderRecord.UpdatedAt) > (FiscalOFDTimeout+15*time.Second)
	if orderRecord.Status != StatusFiscalFailed && !isStuckFiscalizing {
		return CollectCashResponse{}, ErrFiscalNotFailed
	}
	if err := s.requireFiscalPack(ctx, orderRecord.SupplierID); err != nil {
		return CollectCashResponse{}, err
	}
	if claims.Role == auth.RoleDriver {
		if strings.TrimSpace(orderRecord.DriverID) == "" || strings.TrimSpace(orderRecord.DriverID) != strings.TrimSpace(claims.Subject) {
			return CollectCashResponse{}, ErrOrderForbidden
		}
	}

	// Retry budget: soft signal only (order already FISCAL_FAILED).
	if failedN, cErr := s.repo.CountFiscalAttemptsByStatus(ctx, orderID, FiscalAttemptFailed); cErr == nil && failedN >= FiscalMaxFailedAttempts {
		s.log.InfoContext(ctx, "fiscal retry after max failures", "order_id", orderID, "failed_attempts", failedN)
	}
	// Preserve amount + payment method from last attempt (cash shortfall / card path).
	amountMinor := orderRecord.TotalMinor
	payMethod := "CASH"
	if orderRecord.LatestFiscalAttemptID != "" {
		if prev, found, gErr := s.repo.GetFiscalAttempt(ctx, orderID, orderRecord.LatestFiscalAttemptID); gErr == nil && found {
			if prev.AmountMinor > 0 {
				amountMinor = prev.AmountMinor
			}
			if strings.TrimSpace(prev.PaymentMethod) != "" && prev.PaymentMethod != "FORCE" {
				payMethod = prev.PaymentMethod
			}
		}
	}
	attemptID := s.newID()
	row := s.newFiscalPendingRow(ctx, orderRecord, payMethod, attemptID, amountMinor)
	previousStatus := orderRecord.Status
	orderRecord.Status = StatusFiscalizing
	orderRecord.FiscalStatus = FiscalStatusPending
	orderRecord.LatestFiscalAttemptID = attemptID
	orderRecord.UpdatedAt = s.now().UTC()
	orderRecord.PendingFiscalReceipts = []FiscalReceiptRow{row}

	err = s.repo.UpdateOrder(ctx, orderRecord, nil, func(txn outbox.TxnBuffer) error {
		if err := emitOrderStatusChanged(ctx, txn, orderStatusEmitParams{
			Claims:         claims,
			Order:          orderRecord,
			PreviousStatus: previousStatus,
			Reason:         "fiscal_retry",
			ActorID:        claims.Subject,
		}); err != nil {
			return err
		}
		return emitFiscalReceiptRequested(ctx, txn, row)
	})
	if err != nil {
		return CollectCashResponse{}, err
	}
	s.afterOrderMutation(ctx, orderRecord)

	return CollectCashResponse{
		OrderID:   orderRecord.OrderID,
		State:     orderRecord.Status,
		Amount:    orderRecord.TotalMinor,
		Currency:  orderRecord.Currency,
		Message:   "Fiscal retry requested.",
		AttemptID: attemptID,
	}, nil
}

const maxOrderVatTotalsCache = 10000

var (
	orderVatTotalsMu sync.RWMutex
	orderVatTotals   = make(map[string]int64)
)

// VatTotalMinor returns the computed total VAT in minor units for the order.
func (o *Order) VatTotalMinor() int64 {
	if o == nil {
		return 0
	}
	orderVatTotalsMu.RLock()
	defer orderVatTotalsMu.RUnlock()
	return orderVatTotals[o.OrderID]
}

// SetVatTotalMinor records the computed total VAT in minor units for the order.
func (o *Order) SetVatTotalMinor(val int64) {
	if o == nil {
		return
	}
	orderVatTotalsMu.Lock()
	defer orderVatTotalsMu.Unlock()
	if len(orderVatTotals) >= maxOrderVatTotalsCache {
		clear(orderVatTotals)
	}
	orderVatTotals[o.OrderID] = val
}

// GetOrderVatTotalMinor returns the recorded VAT total minor for the given order ID.
func GetOrderVatTotalMinor(orderID string) int64 {
	orderVatTotalsMu.RLock()
	defer orderVatTotalsMu.RUnlock()
	return orderVatTotals[orderID]
}

func (s *Service) stampTaxRegimeTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, orderRecord *Order) error {
	_, _, _, err := s.stampTaxRegimeBreakdownTxn(ctx, txn, orderRecord)
	return err
}

func (s *Service) stampTaxRegimeBreakdownTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, orderRecord *Order) (totalNetMinor, totalVatMinor, totalGrossMinor int64, err error) {
	if s.taxSvc == nil {
		return 0, 0, 0, nil
	}

	countryCode, err := auth.CountryFromContext(ctx, orderRecord.SupplierID)
	if err != nil {
		return 0, 0, 0, err
	}

	regime, found, err := s.taxSvc.Repo().GetActiveRegime(ctx, txn, countryCode, time.Now().UTC())
	if err != nil {
		return 0, 0, 0, fmt.Errorf("load active tax regime: %w", err)
	}
	if !found {
		return 0, 0, 0, fmt.Errorf("no active tax regime found for country %s", countryCode)
	}

	var defaultRegimeVat int64 = 0
	if len(regime.VatRatesBps) > 0 {
		defaultRegimeVat = regime.VatRatesBps[0]
	}

	seenSKUs := make(map[string]int)

	for idx, line := range orderRecord.LineItems {
		// Line-item specific VAT rate resolution:
		// Priority 1: Explicit line-item VatRateBps
		// Priority 2: Line-item TaxCategory ("EXEMPT" or "ZERO" -> 0 bps, "STANDARD" -> defaultRegimeVat)
		// Priority 3: Default tax regime active rate (12% = 1200 bps in UZ)
		lineVatRate := defaultRegimeVat
		if line.VatRateBps != nil {
			if *line.VatRateBps <= 0 {
				lineVatRate = 0
			} else {
				lineVatRate = *line.VatRateBps
			}
		} else if line.TaxCategory != "" {
			switch strings.ToUpper(strings.TrimSpace(line.TaxCategory)) {
			case "EXEMPT", "ZERO", "TAX_EXEMPT":
				lineVatRate = 0
			case "STANDARD":
				lineVatRate = defaultRegimeVat
			}
		}

		net := line.Quantity * line.UnitPrice
		var vat int64 = 0
		if lineVatRate > 0 && net > 0 {
			// Soliq OFD integer half-up rounding: (net * rate_bps + 5000) / 10000
			vat = (net*lineVatRate + 5000) / 10000
		}
		gross := net + vat

		totalNetMinor += net
		totalVatMinor += vat
		totalGrossMinor += gross

		orderLineID := line.SKU
		if seenSKUs[line.SKU] > 0 {
			orderLineID = fmt.Sprintf("%s#%d", line.SKU, idx)
		}
		seenSKUs[line.SKU]++

		snap := tax.OrderLineFiscalSnapshot{
			OrderId:     orderRecord.OrderID,
			OrderLineId: orderLineID,
			RegimeId:    regime.Id,
			VatRateBps:  lineVatRate,
			NetMinor:    net,
			VatMinor:    vat,
			GrossMinor:  gross,
			SnapshotAt:  time.Now().UTC(),
			CreatedAt:   time.Now().UTC(),
		}
		orderRecord.PendingFiscalSnapshots = append(orderRecord.PendingFiscalSnapshots, snap)
		if err := s.taxSvc.Repo().InsertLineSnapshot(ctx, txn, snap); err != nil {
			return 0, 0, 0, fmt.Errorf("insert tax snapshot: %w", err)
		}
	}

	orderRecord.SetVatTotalMinor(totalVatMinor)
	return totalNetMinor, totalVatMinor, totalGrossMinor, nil
}

// ForceCompleteOrder completes with FORCE_SKIPPED fiscal audit (ADMIN / WAREHOUSE_ADMIN).
// Rejects when fiscal already SUCCESS (P0 R1/R2).
func (s *Service) ForceCompleteOrder(ctx context.Context, claims auth.Claims, orderID, reasonCode string) (CollectCashResponse, error) {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return CollectCashResponse{}, errors.New("order_id required")
	}
	normalizedReason, err := NormalizeForceReasonCode(reasonCode)
	if err != nil {
		return CollectCashResponse{}, err
	}
	switch claims.Role {
	case auth.RoleAdmin, auth.RoleWarehouseAdmin:
	default:
		return CollectCashResponse{}, ErrForceCompleteForbidden
	}

	orderRecord, ok, getErr := s.repo.GetOrder(ctx, orderID)
	if getErr != nil {
		return CollectCashResponse{}, getErr
	}
	if !ok {
		return CollectCashResponse{}, ErrOrderNotFound
	}
	if isMigrationFiscalExcluded(orderRecord) {
		return CollectCashResponse{}, fmt.Errorf("%w: migrated order excluded from fiscalization", ErrFiscalAlreadySucceeded)
	}
	// Never force-skip when a real fiscal SUCCESS exists.
	if orderRecord.FiscalStatus == FiscalStatusSuccess {
		return CollectCashResponse{}, ErrFiscalAlreadySucceeded
	}
	if orderRecord.LatestFiscalAttemptID != "" {
		if existing, found, gErr := s.repo.GetFiscalAttempt(ctx, orderID, orderRecord.LatestFiscalAttemptID); gErr == nil && found && existing.Status == FiscalAttemptSuccess {
			return CollectCashResponse{}, ErrFiscalAlreadySucceeded
		}
	}
	if orderRecord.Status == StatusCompleted && orderRecord.FiscalStatus != FiscalStatusForceSkipped && orderRecord.FiscalStatus != FiscalStatusFailed && orderRecord.FiscalStatus != FiscalStatusPending {
		// Completed with SUCCESS-like fiscal — refuse silent force.
		if orderRecord.FiscalStatus == FiscalStatusSuccess || orderRecord.LatestFiscalReceiptID != "" {
			return CollectCashResponse{}, ErrFiscalAlreadySucceeded
		}
		return CollectCashResponse{
			OrderID:  orderRecord.OrderID,
			State:    orderRecord.Status,
			Amount:   orderRecord.TotalMinor,
			Currency: orderRecord.Currency,
			Message:  "Order already completed",
		}, nil
	}
	// FISCAL_* normal path; RECONCILIATION_REQUIRED = payment landed after cancel / trapped funds (must audit force, not soft-complete).
	switch orderRecord.Status {
	case StatusFiscalFailed, StatusFiscalizing, StatusReconciliationRequired:
	default:
		return CollectCashResponse{}, fmt.Errorf("%w: must be FISCAL_FAILED, FISCALIZING, or RECONCILIATION_REQUIRED (current %s)", ErrInvalidStatusTransition, orderRecord.Status)
	}

	now := s.now().UTC()
	attemptID := s.newID()
	row := FiscalReceiptRow{
		OrderID:       orderRecord.OrderID,
		AttemptID:     attemptID,
		SupplierID:    orderRecord.SupplierID,
		RetailerID:    orderRecord.RetailerID,
		Provider:      s.ProviderName(),
		Status:        FiscalAttemptForceSkipped,
		AmountMinor:   orderRecord.TotalMinor,
		Currency:      orderRecord.Currency,
		PaymentMethod: "FORCE",
		ReasonCode:    normalizedReason,
		ActorID:       claims.Subject,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if row.Currency == "" {
		if cur, err := auth.CurrencyFromContext(ctx, orderRecord.SupplierID); err == nil {
			row.Currency = cur
		}
	}

	previousStatus := orderRecord.Status
	if err := ValidateStatusTransition(string(previousStatus), string(StatusCompleted), TransitionOpts{}); err != nil {
		return CollectCashResponse{}, err
	}
	orderRecord.Status = StatusCompleted
	orderRecord.FiscalStatus = FiscalStatusForceSkipped
	orderRecord.LatestFiscalAttemptID = attemptID
	orderRecord.FiscalizedAt = &now
	orderRecord.UpdatedAt = now
	orderRecord.PendingFiscalReceipts = []FiscalReceiptRow{row}
	_ = s.ApplyClaimWindowSnapshot(ctx, &orderRecord, now)

	preFee, feeErr := s.resolvePerOrderFeeBeforeTxn(ctx, orderRecord.SupplierID)
	if feeErr != nil {
		return CollectCashResponse{}, fmt.Errorf("resolve per-order fee: %w", feeErr)
	}
	inTxn := func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		if err := s.completeOrderTaxAndGLInTxn(ctx, txn, nil, &orderRecord, preFee, ""); err != nil {
			return err
		}

		// Read money state inside this transaction (never Single() inside a
		// read-write txn) and record any remaining shortfall as an exception.
		txnOrder, found, err := s.repo.GetOrderTxn(ctx, txn, orderID)
		if err != nil {
			return fmt.Errorf("failed to get order: %w", err)
		}
		if !found {
			return fmt.Errorf("order not found: %s", orderID)
		}
		delivered := deliveredGrossFromOrder(txnOrder)
		paid, err := s.getCapturedPaymentMinorTxn(ctx, txn, orderID)
		if err != nil {
			return err
		}
		exceptions, err := s.getExceptionsTotalMinorTxn(ctx, txn, orderID)
		if err != nil {
			return err
		}
		if shortfall := delivered - paid - exceptions; shortfall > 0 {
			ex := SettlementException{
				OrderID:     orderID,
				ExceptionID: s.newID(),
				Type:        "FORCE_COMPLETE_SHORTFALL",
				AmountMinor: shortfall,
				Status:      "OPEN",
				CreatedBy:   claims.Subject,
				CreatedAt:   now,
			}
			if writeErr := s.RecordSettlementException(ctx, txn, ex); writeErr != nil {
				return writeErr
			}
		}
		return nil
	}

	err = s.repo.UpdateOrderWithTxn(ctx, orderRecord, nil, inTxn, func(txn outbox.TxnBuffer) error {
		if err := emitOrderStatusChanged(ctx, txn, orderStatusEmitParams{
			Claims:         claims,
			Order:          orderRecord,
			PreviousStatus: previousStatus,
			Reason:         "force_complete:" + normalizedReason,
			ActorID:        claims.Subject,
		}); err != nil {
			return err
		}
		if err := emitOrderForceCompleted(ctx, txn, orderRecord, normalizedReason, claims.Subject); err != nil {
			return err
		}
		return emitOrderFinalized(ctx, txn, orderRecord)
	})
	if err != nil {
		return CollectCashResponse{}, err
	}
	s.afterOrderMutation(ctx, orderRecord)
	if orderRecord.ManifestID != "" {
		_ = s.tryCompleteManifest(ctx, orderRecord.ManifestID)
		s.replanAsync(orderRecord.ManifestID, "force_skip", claims.Subject)
	}

	return CollectCashResponse{
		OrderID:   orderRecord.OrderID,
		State:     orderRecord.Status,
		Amount:    orderRecord.TotalMinor,
		Currency:  orderRecord.Currency,
		Message:   "Force-completed with fiscal FORCE_SKIPPED.",
		AttemptID: attemptID,
	}, nil
}

// isMigrationFiscalExcluded reports whether an order originated from historical
// or held-open migration ingestion and must never hit a live fiscal OFD.
func isMigrationFiscalExcluded(o Order) bool {
	return o.Source == OrderSourceMigration ||
		o.Status == StatusMigrationOpen ||
		o.FiscalStatus == FiscalStatusHistoricalSkipped
}

// SweepStuckFiscalizingOrders sweeps orders stuck in FISCALIZING status beyond staleAfter
// (defaulting to FiscalOFDTimeout + 15s) and moves them to FISCAL_FAILED so they can be
// retried by the fiscal sweeper or manually retried by operators.
func (s *Service) SweepStuckFiscalizingOrders(ctx context.Context, staleAfter time.Duration) (int, error) {
	if staleAfter <= 0 {
		staleAfter = FiscalOFDTimeout + 15*time.Second
	}
	now := s.now().UTC()
	cutoff := now.Add(-staleAfter)

	var candidates []Order
	if s.spannerClient != nil {
		stmt := spanner.Statement{
			SQL: `SELECT OrderId
			      FROM Orders
			      WHERE Status = 'FISCALIZING'
			        AND UpdatedAt <= @cutoff
			      LIMIT 100`,
			Params: map[string]any{"cutoff": cutoff},
		}
		iter := s.spannerClient.Single().Query(ctx, stmt)
		defer iter.Stop()

		var ids []string
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return 0, fmt.Errorf("query stuck fiscalizing orders: %w", err)
			}
			var id string
			if err := row.Columns(&id); err != nil {
				continue
			}
			ids = append(ids, id)
		}
		for _, id := range ids {
			o, found, err := s.repo.GetOrder(ctx, id)
			if err != nil || !found {
				continue
			}
			if o.Status == StatusFiscalizing && !o.UpdatedAt.After(cutoff) {
				candidates = append(candidates, o)
			}
		}
	} else if s.repo != nil {
		orders, err := s.repo.ListOrdersByStatus(ctx, "", string(StatusFiscalizing), 200)
		if err != nil {
			return 0, fmt.Errorf("list fiscalizing orders: %w", err)
		}
		for _, o := range orders {
			if o.Status == StatusFiscalizing && !o.UpdatedAt.After(cutoff) {
				candidates = append(candidates, o)
			}
		}
	}

	swept := 0
	for _, o := range candidates {
		prevStatus := o.Status
		o.Status = StatusFiscalFailed
		o.FiscalStatus = FiscalStatusFailed
		o.UpdatedAt = now
		err := s.repo.UpdateOrder(ctx, o, nil, func(txn outbox.TxnBuffer) error {
			return emitOrderStatusChanged(ctx, txn, orderStatusEmitParams{
				Order:          o,
				PreviousStatus: prevStatus,
				Reason:         "fiscal_timeout_swept",
				ActorID:        "system:fiscal-sweeper",
			})
		})
		if err != nil {
			s.log.WarnContext(ctx, "failed transitioning stuck fiscalizing order to fiscal_failed", "order_id", o.OrderID, "err", err)
			continue
		}
		s.afterOrderMutation(ctx, o)
		swept++
	}

	return swept, nil
}
