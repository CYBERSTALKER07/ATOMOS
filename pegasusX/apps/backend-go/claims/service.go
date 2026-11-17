package claims

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/storage"
)

// DefaultPostDeliveryClaimWindow is how long after COMPLETED a retailer may file.
const DefaultPostDeliveryClaimWindow = 48 * time.Hour

// OrderStatusCompleted is the post-delivery gate (mirrors order.StatusCompleted without importing order).
const OrderStatusCompleted = "COMPLETED"

// OrderSnapshot is the minimal order view claims needs (avoids order↔claims import cycle).
//
// Lifecycle note: orders become COMPLETED at delivery handshake + fiscal success
// (same day as delivery), NOT after a multi-day wait. The claim window (default 48h)
// starts at that completion timestamp (UpdatedAt on COMPLETED).
type OrderSnapshot struct {
	OrderID            string
	SupplierID         string
	RetailerID         string
	WarehouseID        string
	Currency           string
	Status             string
	TotalMinor         int64
	OriginalTotalMinor int64 // commercial total before OS&D amend; preferred chargeback ceiling
	LineItems          []OrderLine
	CreatedAt          time.Time
	UpdatedAt          time.Time
	// G3 immutable claim window (preferred over env duration when set).
	ClaimWindowHours        int64
	ClaimWindowEndsAt       *time.Time
	ClaimWindowPolicySource string
}

// OrderLookup loads order snapshots for claim authorization.
type OrderLookup interface {
	GetOrder(ctx context.Context, orderID string) (OrderSnapshot, bool, error)
}

// ReverseLogisticsOpener creates warehouse inbound tickets for damaged goods.
// Implemented by returns.Service via bootstrap (avoids claims↔returns import cycle).
type ReverseLogisticsOpener interface {
	OpenFromClaim(ctx context.Context, in ReverseLogisticsInput) error
}

// ReverseLogisticsInput is the claim → dock ticket payload.
type ReverseLogisticsInput struct {
	OrderID     string
	WarehouseID string
	SupplierID  string
	DriverID    string
	ClaimID     string
	Source      string
	Note        string
	Lines       []ClaimLine
}

// StoreCreditApplier reduces retailer credit balance due (store-credit settlement).
type StoreCreditApplier interface {
	ClearBalance(ctx context.Context, retailerID, supplierID string, amountMinor int64, orderID string) error
}

// CreditNoteCreator optionally creates financial credit notes from approved claims.
type CreditNoteCreator interface {
	CreateFromClaim(ctx context.Context, claimID, actor string) error
}

// StoreStockClaimPort moves retailer store stock into/out of QUARANTINE for claims.
// Implemented by retailer.Service via bootstrap (avoids claims↔retailer import cycle).
type StoreStockClaimPort interface {
	HoldForClaim(ctx context.Context, retailerID, claimID, orderID string, lines []ClaimLine, actor string) error
	ResolveClaimStock(ctx context.Context, retailerID, claimID string, lines []ClaimLine, disposition, actor string) error
}

func creditNoteAutoFromClaimEnabled() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("CREDIT_NOTE_AUTO_FROM_CLAIM")), "true")
}

// Service is the claims domain application service.
type Service struct {
	repo            Repository
	orders          OrderLookup
	settler         ChargebackSettler
	rl              ReverseLogisticsOpener
	storeCr         StoreCreditApplier
	creditNotes     CreditNoteCreator
	storeStock      StoreStockClaimPort
	inspectEvidence storage.EvidenceInspector
	idem            idempotency.Store
	now             func() time.Time
	newID           func() string
	log             *slog.Logger
	window          time.Duration
}

// Config wires claims Service dependencies.
type Config struct {
	Repo              Repository
	Orders            OrderLookup
	Settler           ChargebackSettler
	RL                ReverseLogisticsOpener
	StoreCr           StoreCreditApplier
	CreditNotes       CreditNoteCreator
	StoreStock        StoreStockClaimPort
	EvidenceInspector storage.EvidenceInspector
	Idem              idempotency.Store
	Now               func() time.Time
	NewID             func() string
	Log               *slog.Logger
	Window            time.Duration
}

// NewService builds a claims service.
func NewService(cfg Config) *Service {
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	newID := cfg.NewID
	if newID == nil {
		newID = func() string {
			return strings.ReplaceAll(uuid.NewString(), "-", "")
		}
	}
	log := cfg.Log
	if log == nil {
		log = slog.Default()
	}
	window := cfg.Window
	if window <= 0 {
		window = claimWindowFromEnv()
	}
	return &Service{
		repo: cfg.Repo, orders: cfg.Orders, settler: cfg.Settler, rl: cfg.RL,
		storeCr: cfg.StoreCr, creditNotes: cfg.CreditNotes, storeStock: cfg.StoreStock,
		inspectEvidence: cfg.EvidenceInspector,
		idem:            cfg.Idem, now: now, newID: newID, log: log, window: window,
	}
}

// SetEvidenceInspector overrides bounded GCS streaming evidence inspection (used in tests).
func (s *Service) SetEvidenceInspector(fn storage.EvidenceInspector) {
	if s != nil {
		s.inspectEvidence = fn
	}
}

// SetReverseLogistics wires the warehouse inbound ticket opener (optional).
func (s *Service) SetReverseLogistics(op ReverseLogisticsOpener) {
	if s != nil {
		s.rl = op
	}
}

// SetStoreCredit wires store-credit settlement (optional credit domain).
func (s *Service) SetStoreCredit(sc StoreCreditApplier) {
	if s != nil {
		s.storeCr = sc
	}
}

// SetStoreStock wires retailer store quarantine bridge (optional).
func (s *Service) SetStoreStock(ss StoreStockClaimPort) {
	if s != nil {
		s.storeStock = ss
	}
}

// SetCreditNotes wires optional auto credit note creation from approved claims.
func (s *Service) SetCreditNotes(cn CreditNoteCreator) {
	if s != nil {
		s.creditNotes = cn
	}
}

// openReverseLogistics attempts a sync dock-ticket open. On failure increments
// claim_reverse_open_fail_total; the returns Kafka consumer retries via outbox (G12).
func (s *Service) openReverseLogistics(ctx context.Context, in ReverseLogisticsInput) {
	if s == nil || s.rl == nil {
		return
	}
	if err := s.rl.OpenFromClaim(ctx, in); err != nil {
		incClaimReverseOpenFail()
		s.log.ErrorContext(ctx, "reverse logistics ticket open failed (async retry via REVERSE_LOGISTICS_REQUIRED)",
			"err", err, "claim_id", in.ClaimID, "order_id", in.OrderID, "warehouse_id", in.WarehouseID)
	}
}

// SetSettler wires payment chargeback settlement after construction.
func (s *Service) SetSettler(settler ChargebackSettler) {
	if s != nil {
		s.settler = settler
	}
}

// ListOrderClaims returns claims for an order with role-scoped authorization.
func (s *Service) ListOrderClaims(ctx context.Context, actor auth.Claims, orderID string) ([]Claim, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("claims service unavailable")
	}
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return nil, ErrOrderNotFound
	}
	// Retailers may only inspect their own orders.
	if actor.Role == auth.RoleRetailer {
		if s.orders == nil {
			return nil, fmt.Errorf("claims service unavailable")
		}
		o, ok, err := s.orders.GetOrder(ctx, orderID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrOrderNotFound
		}
		retailerOrgID := auth.ResolveRetailerOrgID(actor)
		if retailerOrgID == "" || strings.TrimSpace(o.RetailerID) != retailerOrgID {
			return nil, ErrForbidden
		}
	} else if actor.Role == auth.RoleAdmin {
		// Supplier portal admin: only claims for their supplier on this order.
		if sid := strings.TrimSpace(actor.SupplierID); sid != "" {
			list, err := s.repo.ListByOrder(ctx, orderID)
			if err != nil {
				return nil, err
			}
			out := make([]Claim, 0, len(list))
			for _, c := range list {
				if c.SupplierID == sid {
					out = append(out, c)
				}
			}
			return out, nil
		}
	} else if actor.Role != auth.RoleWarehouseAdmin {
		return nil, ErrForbidden
	}
	return s.repo.ListByOrder(ctx, orderID)
}

// ListOrderClaimsPage returns a keyset-paginated slice of claims for an order.
func (s *Service) ListOrderClaimsPage(ctx context.Context, actor auth.Claims, orderID string, limit int, cursor string) ([]Claim, string, error) {
	if s == nil || s.repo == nil {
		return nil, "", fmt.Errorf("claims service unavailable")
	}
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return nil, "", ErrOrderNotFound
	}
	if actor.Role == auth.RoleRetailer {
		if s.orders == nil {
			return nil, "", fmt.Errorf("claims service unavailable")
		}
		o, ok, err := s.orders.GetOrder(ctx, orderID)
		if err != nil {
			return nil, "", err
		}
		if !ok {
			return nil, "", ErrOrderNotFound
		}
		retailerOrgID := auth.ResolveRetailerOrgID(actor)
		if retailerOrgID == "" || strings.TrimSpace(o.RetailerID) != retailerOrgID {
			return nil, "", ErrForbidden
		}
	} else if actor.Role != auth.RoleAdmin && actor.Role != auth.RoleWarehouseAdmin {
		return nil, "", ErrForbidden
	}

	var list []Claim
	var nextCursor string
	var err error
	if pager, ok := s.repo.(OrderClaimsPager); ok {
		list, nextCursor, err = pager.ListByOrderPage(ctx, orderID, limit, cursor)
	} else {
		if strings.TrimSpace(cursor) != "" {
			return nil, "", ErrInvalidCursor
		}
		list, err = s.repo.ListByOrder(ctx, orderID)
	}
	if err != nil {
		return nil, "", err
	}
	if actor.Role == auth.RoleAdmin {
		if sid := strings.TrimSpace(actor.SupplierID); sid != "" {
			filtered := make([]Claim, 0, len(list))
			for _, c := range list {
				if c.SupplierID == sid {
					filtered = append(filtered, c)
				}
			}
			return filtered, nextCursor, nil
		}
	}
	return list, nextCursor, nil
}

// ListSupplierClaims is the supplier HQ adjudication queue.
func (s *Service) ListSupplierClaims(ctx context.Context, actor auth.Claims, status Status, limit int) ([]Claim, error) {
	list, _, err := s.ListSupplierClaimsPage(ctx, actor, status, limit, "")
	return list, err
}

// ListSupplierClaimsPage returns keyset-paginated supplier claims.
func (s *Service) ListSupplierClaimsPage(ctx context.Context, actor auth.Claims, status Status, limit int, cursor string) ([]Claim, string, error) {
	if s == nil || s.repo == nil {
		return nil, "", fmt.Errorf("claims service unavailable")
	}
	if actor.Role != auth.RoleAdmin && actor.Role != auth.RoleWarehouseAdmin {
		return nil, "", ErrForbidden
	}
	supplierID := strings.TrimSpace(actor.SupplierID)
	if supplierID == "" {
		if sid, ok := auth.ResolveSupplierID(ctx); ok {
			supplierID = sid
		}
	}
	if supplierID == "" {
		return nil, "", ErrForbidden
	}
	if pager, ok := s.repo.(SupplierClaimsPager); ok {
		return pager.ListBySupplierPage(ctx, supplierID, status, limit, cursor)
	}
	if strings.TrimSpace(cursor) != "" {
		return nil, "", ErrInvalidCursor
	}
	list, err := s.repo.ListBySupplier(ctx, supplierID, status, limit)
	return list, "", err
}

func actorMaySettleClaim(actor auth.Claims, c Claim) bool {
	if actor.Role == auth.RoleWarehouseAdmin {
		return true
	}
	if actor.Role != auth.RoleAdmin {
		return false
	}
	// Supplier portal admin must own the claim's supplier.
	sid := strings.TrimSpace(actor.SupplierID)
	if sid == "" {
		return true // legacy unscoped admin
	}
	return sid == strings.TrimSpace(c.SupplierID)
}

func claimWindowFromEnv() time.Duration {
	h := strings.TrimSpace(os.Getenv("CLAIM_WINDOW_HOURS"))
	if h == "" {
		return DefaultPostDeliveryClaimWindow
	}
	n, err := strconv.Atoi(h)
	if err != nil || n <= 0 {
		return DefaultPostDeliveryClaimWindow
	}
	return time.Duration(n) * time.Hour
}

// CLAIM_AUTO_APPROVE_MAX_MINOR: 0/empty = disabled. When set, non-escalated claims with
// AmountMinor <= threshold auto-approve inside the single atomic filing txn (and settle LEDGER_ONLY).
func autoApproveMaxMinor() int64 {
	raw := strings.TrimSpace(os.Getenv("CLAIM_AUTO_APPROVE_MAX_MINOR"))
	if raw == "" {
		return 0
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

// EvidenceDHashEnabled keeps perceptual dHash strictly disabled ("off") while
// CLAIM_AUTO_APPROVE_MAX_MINOR == 0 per Arbiter YAGNI ruling.
func EvidenceDHashEnabled() bool {
	if autoApproveMaxMinor() <= 0 {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(os.Getenv("CLAIM_EVIDENCE_DHASH_ENABLED")), "true")
}

func claimEscalationThresholdMinor() int64 {
	raw := strings.TrimSpace(os.Getenv("CLAIM_ESCALATION_THRESHOLD_MINOR"))
	if raw == "" {
		return 0
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

func claimRepeatEscalationCount() int {
	raw := strings.TrimSpace(os.Getenv("CLAIM_REPEAT_ESCALATION_COUNT"))
	if raw == "" {
		return 3
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 3
	}
	return n
}

func resolveSettlementFlags(req ApproveClaimRequest) (skipGateway bool, storeCredit bool) {
	mode := strings.ToUpper(strings.TrimSpace(req.SettlementMode))
	switch mode {
	case "LEDGER_ONLY":
		return true, false
	case "STORE_CREDIT":
		return true, true
	case "GATEWAY_REFUND":
		return false, false
	default:
		return req.SkipGatewayRefund, false
	}
}

// EvidenceInput is a normalized evidence attachment passed to bounded pre-txn verification.
type EvidenceInput struct {
	EvidenceType EvidenceType
	URI          string
	MimeType     string
}

func (s *Service) inspectEvidenceURI(ctx context.Context, uri string) (storage.EvidenceInspection, error) {
	if s != nil && s.inspectEvidence != nil {
		return s.inspectEvidence(ctx, uri)
	}
	return storage.InspectEvidenceObject(ctx, uri)
}

func (s *Service) checkEvidenceNotReused(ctx context.Context, uri, uriSha256, contentSha256 string) error {
	if s == nil || s.repo == nil {
		return fmt.Errorf("claims repository unavailable")
	}
	if deduper, ok := s.repo.(EvidenceDedupeRepository); ok {
		reused, err := deduper.IsEvidenceReused(ctx, uri, uriSha256, contentSha256)
		if err != nil {
			return err
		}
		if reused {
			return ErrReusedPhotoEvidence
		}
		return nil
	}
	reused, err := s.repo.IsEvidenceURIReused(ctx, uri)
	if err != nil {
		return err
	}
	if reused {
		return ErrReusedPhotoEvidence
	}
	return nil
}

// inspectAndPrepareEvidences performs bounded streaming GCS SHA-256 verification and
// fail-closed dedupe verification BEFORE opening any Spanner read-write transaction.
func (s *Service) inspectAndPrepareEvidences(
	ctx context.Context,
	claimID string,
	inputs []EvidenceInput,
	capturedBy string,
	now time.Time,
	requirePhoto bool,
) ([]Evidence, bool, error) {
	hasPhoto := false
	evs := make([]Evidence, 0, len(inputs))
	seenURIs := make(map[string]struct{}, len(inputs))
	seenContentHashes := make(map[string]struct{}, len(inputs))

	for _, in := range inputs {
		uri := strings.TrimSpace(in.URI)
		if uri == "" {
			continue
		}
		if err := storage.ValidateEvidenceURI(uri); err != nil {
			return nil, false, ErrInvalidEvidenceURI
		}
		et := in.EvidenceType
		if et == "" {
			et = EvidencePhoto
		}
		if et == EvidencePhoto {
			hasPhoto = true
		}

		insp, err := s.inspectEvidenceURI(ctx, uri)
		if err != nil {
			if errors.Is(err, storage.ErrInvalidEvidenceURI) {
				return nil, false, ErrInvalidEvidenceURI
			}
			if errors.Is(err, storage.ErrMediaStorageUnavailable) && !storage.EvidenceFailClosed() {
				uriSha, _, uerr := storage.ComputeEvidenceURISha256(uri)
				if uerr != nil {
					return nil, false, ErrInvalidEvidenceURI
				}
				insp = storage.EvidenceInspection{
					URI:       uri,
					UriSha256: uriSha,
				}
			} else {
				return nil, false, err
			}
		} else {
			if strings.TrimSpace(insp.UriSha256) == "" {
				uriSha, _, uerr := storage.ComputeEvidenceURISha256(uri)
				if uerr != nil {
					return nil, false, ErrInvalidEvidenceURI
				}
				insp.UriSha256 = uriSha
			}
			if strings.TrimSpace(insp.ContentSha256) == "" {
				return nil, false, storage.ErrMediaStorageUnavailable
			}
		}

		uriKey := insp.UriSha256
		if uriKey == "" {
			uriKey = uri
		}
		if _, dup := seenURIs[uriKey]; dup {
			return nil, false, ErrReusedPhotoEvidence
		}
		seenURIs[uriKey] = struct{}{}
		if insp.ContentSha256 != "" {
			if _, dup := seenContentHashes[insp.ContentSha256]; dup {
				return nil, false, ErrReusedPhotoEvidence
			}
			seenContentHashes[insp.ContentSha256] = struct{}{}
		}

		evs = append(evs, Evidence{
			EvidenceID:       s.newID(),
			ClaimID:          claimID,
			EvidenceType:     et,
			URI:              uri,
			MimeType:         strings.TrimSpace(in.MimeType),
			ContentSha256:    insp.ContentSha256,
			UriSha256:        insp.UriSha256,
			ObjectGeneration: insp.ObjectGeneration,
			CapturedBy:       capturedBy,
			CreatedAt:        now,
		})
	}
	if requirePhoto && !hasPhoto {
		return nil, false, ErrEvidenceRequired
	}
	for _, ev := range evs {
		if err := s.checkEvidenceNotReused(ctx, ev.URI, ev.UriSha256, ev.ContentSha256); err != nil {
			return nil, false, err
		}
	}
	return evs, hasPhoto, nil
}

func (s *Service) buildEscalationMetadata(
	ctx context.Context,
	retailerID string,
	amountMinor int64,
	prior []Claim,
	now time.Time,
) (*EscalationMetadata, error) {
	dhashStatus := "off"
	if EvidenceDHashEnabled() {
		dhashStatus = "enabled"
	}
	priorOrderClaims := 0
	for _, p := range prior {
		if p.Status != StatusRejected {
			priorOrderClaims++
		}
	}
	recentCount := 0
	if counter, ok := s.repo.(RetailerClaimCounter); ok && strings.TrimSpace(retailerID) != "" {
		cnt, err := counter.CountRecentClaimsByRetailer(ctx, retailerID, now.Add(-30*24*time.Hour))
		if err != nil {
			return nil, err
		}
		recentCount = cnt
	}

	threshold := claimEscalationThresholdMinor()
	repeatLimit := claimRepeatEscalationCount()
	var reasons []string
	if threshold > 0 && amountMinor >= threshold {
		reasons = append(reasons, "amount_threshold_exceeded")
	}
	if priorOrderClaims >= 1 {
		reasons = append(reasons, "multiple_claims_same_order")
	}
	if recentCount >= repeatLimit {
		reasons = append(reasons, "repeat_retailer_claim_pattern")
	}
	return &EscalationMetadata{
		EscalationRequired: len(reasons) > 0,
		Reasons:            reasons,
		DHashStatus:        dhashStatus,
		RecentClaimCount:   recentCount,
		PriorOrderClaims:   priorOrderClaims,
		ThresholdMinor:     threshold,
	}, nil
}

func emitClaimLifecycleOutbox(
	ctx context.Context,
	txn outbox.TxnBuffer,
	c Claim,
	warehouseID, driverID string,
	needsReverseLogistics bool,
	now time.Time,
) error {
	payload := map[string]any{
		"type":         events.EventClaimFiled,
		"claim_id":     c.ClaimID,
		"order_id":     c.OrderID,
		"supplier_id":  c.SupplierID,
		"retailer_id":  c.RetailerID,
		"warehouse_id": warehouseID,
		"claim_type":   string(c.ClaimType),
		"status":       string(StatusOpen),
		"source":       string(c.Source),
		"amount_minor": c.AmountMinor,
		"currency":     c.Currency,
		"line_items":   c.LineItems,
		"photo_count":  len(c.Evidences),
		"escalation":   c.Escalation,
		"timestamp":    now.Format(time.RFC3339Nano),
	}
	if driverID != "" {
		payload["driver_id"] = driverID
	}
	if err := outbox.EmitJSON(ctx, txn, events.AggregateClaim, c.ClaimID, events.TopicExceptions, payload); err != nil {
		return err
	}
	if err := outbox.EmitJSON(ctx, txn, events.AggregateClaim, c.ClaimID, events.TopicMain, payload); err != nil {
		return err
	}
	if needsReverseLogistics {
		rlPayload := map[string]any{
			"type":         events.EventReverseLogisticsRequired,
			"claim_id":     c.ClaimID,
			"order_id":     c.OrderID,
			"warehouse_id": warehouseID,
			"supplier_id":  c.SupplierID,
			"retailer_id":  c.RetailerID,
			"claim_type":   string(c.ClaimType),
			"source":       string(c.Source),
			"line_items":   c.LineItems,
			"timestamp":    now.Format(time.RFC3339Nano),
		}
		if driverID != "" {
			rlPayload["driver_id"] = driverID
		}
		if err := outbox.EmitJSON(ctx, txn, events.AggregateOrder, c.OrderID, events.TopicExceptions, rlPayload); err != nil {
			return err
		}
	}
	if c.Status == StatusApproved {
		resPayload := map[string]any{
			"type":            events.EventClaimResolved,
			"claim_id":        c.ClaimID,
			"order_id":        c.OrderID,
			"supplier_id":     c.SupplierID,
			"retailer_id":     c.RetailerID,
			"status":          string(c.Status),
			"amount_minor":    c.AmountMinor,
			"currency":        c.Currency,
			"settlement_mode": "LEDGER_ONLY",
			"line_items":      c.LineItems,
			"resolution_note": c.ResolutionNote,
			"resolved_by":     c.ResolvedBy,
			"timestamp":       now.Format(time.RFC3339Nano),
		}
		if err := outbox.EmitJSON(ctx, txn, events.AggregateClaim, c.ClaimID, events.TopicExceptions, resPayload); err != nil {
			return err
		}
		if err := outbox.EmitJSON(ctx, txn, events.AggregateClaim, c.ClaimID, events.TopicMain, resPayload); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) commitClaimSingleTxn(
	ctx context.Context,
	c Claim,
	holdStoreStock bool,
	holdActor string,
	warehouseID, driverID string,
	needsReverseLogistics bool,
	now time.Time,
) (Claim, error) {
	maxApprove := autoApproveMaxMinor()
	autoApprove := maxApprove > 0 &&
		c.AmountMinor > 0 &&
		c.AmountMinor <= maxApprove &&
		(c.Escalation == nil || !c.Escalation.EscalationRequired)

	if atomicRepo, ok := s.repo.(AtomicClaimRepository); ok {
		return atomicRepo.CreateClaimAtomic(ctx, c, AtomicClaimOptions{
			HoldStoreStock:      holdStoreStock,
			HoldActor:           holdActor,
			AutoApprove:         autoApprove,
			AutoApproveMaxMinor: maxApprove,
			Now:                 now,
		}, s.storeStock, func(txn outbox.TxnBuffer, committed Claim) error {
			return emitClaimLifecycleOutbox(ctx, txn, committed, warehouseID, driverID, needsReverseLogistics, now)
		})
	}

	// Custom non-AtomicClaimRepository test double fallback.
	if holdStoreStock {
		if err := s.holdStoreStockForClaim(ctx, c, holdActor); err != nil {
			return Claim{}, fmt.Errorf("%w: %v", ErrClaimStockHoldFailed, err)
		}
	}
	if autoApprove {
		c.Status = StatusApproved
		c.ResolvedBy = "system:auto_approve"
		c.ResolutionNote = fmt.Sprintf("auto_approved_below_%d", maxApprove)
		resolvedAt := now
		c.ResolvedAt = &resolvedAt
		c.UpdatedAt = now
	}
	if err := s.repo.CreateClaim(ctx, c, func(txn outbox.TxnBuffer) error {
		return emitClaimLifecycleOutbox(ctx, txn, c, warehouseID, driverID, needsReverseLogistics, now)
	}); err != nil {
		return Claim{}, err
	}
	return c, nil
}

func (s *Service) settleAutoApprovedClaim(ctx context.Context, c Claim) {
	if s == nil || c.Status != StatusApproved || c.AmountMinor <= 0 {
		return
	}
	if s.settler != nil {
		if _, err := s.settler.SettleClaimChargeback(ctx, ClaimSettlement{
			ClaimID:           c.ClaimID,
			OrderID:           c.OrderID,
			SupplierID:        c.SupplierID,
			RetailerID:        c.RetailerID,
			AmountMinor:       c.AmountMinor,
			Currency:          c.Currency,
			SkipGatewayRefund: true,
		}); err != nil {
			s.log.WarnContext(ctx, "auto-approved claim post-commit settler failed", "err", err, "claim_id", c.ClaimID)
		}
	}
	if s.creditNotes != nil && creditNoteAutoFromClaimEnabled() {
		if err := s.creditNotes.CreateFromClaim(ctx, c.ClaimID, c.ResolvedBy); err != nil {
			s.log.WarnContext(ctx, "auto credit note from claim failed", "err", err, "claim_id", c.ClaimID)
		}
	}
	disposition := "RETURN"
	if c.ClaimType == ClaimTypeDamaged || c.ClaimType == ClaimTypeConcealedDamage {
		disposition = "WASTE"
	}
	s.resolveStoreStockForClaim(ctx, c, disposition, c.ResolvedBy)
}

// FileRetailerClaim creates an OPEN claim for a completed order within the window.
func (s *Service) FileRetailerClaim(ctx context.Context, claims auth.Claims, orderID string, req FileClaimRequest) (Claim, error) {
	if s == nil || s.repo == nil || s.orders == nil {
		return Claim{}, fmt.Errorf("claims service unavailable")
	}
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return Claim{}, ErrOrderNotFound
	}
	if claims.Role != auth.RoleRetailer && claims.Role != auth.RoleAdmin {
		return Claim{}, ErrForbidden
	}
	if !req.ClaimType.Valid() {
		return Claim{}, ErrInvalidClaimType
	}
	if len(req.LineItems) == 0 {
		return Claim{}, ErrInvalidLineItems
	}
	for _, li := range req.LineItems {
		if strings.TrimSpace(li.SKU) == "" || li.Quantity <= 0 {
			return Claim{}, ErrInvalidLineItems
		}
	}

	// Post-delivery claims require photo evidence for damage types.
	needsPhoto := req.ClaimType == ClaimTypeDamaged || req.ClaimType == ClaimTypeConcealedDamage ||
		req.ClaimType == ClaimTypeTemperature || req.ClaimType == ClaimTypeTamper
	now := s.now().UTC()
	claimID := s.newID()
	if !strings.HasPrefix(claimID, "clm_") {
		claimID = "clm_" + claimID
	}
	capturedBy := auth.ResolveRetailerUserID(claims)

	evInputs := make([]EvidenceInput, 0, len(req.Evidences))
	for _, e := range req.Evidences {
		evInputs = append(evInputs, EvidenceInput{
			EvidenceType: e.EvidenceType,
			URI:          e.URI,
			MimeType:     e.MimeType,
		})
	}
	// Bounded streaming GCS SHA-256 verification + fail-closed dedupe BEFORE Spanner RW txn.
	evs, _, err := s.inspectAndPrepareEvidences(ctx, claimID, evInputs, capturedBy, now, needsPhoto)
	if err != nil {
		return Claim{}, err
	}

	o, ok, err := s.orders.GetOrder(ctx, orderID)
	if err != nil {
		return Claim{}, err
	}
	if !ok {
		return Claim{}, ErrOrderNotFound
	}
	if claims.Role == auth.RoleRetailer {
		org := auth.ResolveRetailerOrgID(claims)
		if strings.TrimSpace(o.RetailerID) != org {
			return Claim{}, ErrForbidden
		}
		if !auth.HasRetailerPerm(claims, auth.PermClaimFile) {
			return Claim{}, ErrForbidden
		}
	}
	elig := EvaluateClaimEligibility(o, now, s.window)
	if !elig.Eligible {
		switch elig.Reason {
		case "claim_window_expired":
			return Claim{}, ErrClaimWindowExpired
		case "order_not_completed":
			return Claim{}, fmt.Errorf("%w: order status %s (claims only after delivery complete)", ErrClaimNotAllowed, o.Status)
		default:
			return Claim{}, fmt.Errorf("%w: order status %s (claims only after delivery complete)", ErrClaimNotAllowed, o.Status)
		}
	}

	// Cumulative liability: prior non-rejected claims reserve qty on the order.
	prior, err := s.repo.ListByOrder(ctx, orderID)
	if err != nil {
		return Claim{}, err
	}
	priorClaimed := ClaimedQtyBySKU(prior, "")

	// Price from order lines — never trust client-supplied totals for chargeback.
	pricedLines, pricedTotal, err := PriceClaimLinesWithPrior(o.LineItems, req.LineItems, priorClaimed)
	if err != nil {
		return Claim{}, err
	}
	amountMinor := pricedTotal
	if req.AmountMinor > 0 && req.AmountMinor < pricedTotal {
		// Retailer may claim a lower amount; never higher than order-priced total.
		amountMinor = req.AmountMinor
	}
	// Canonical integer-safe residual check.
	rem, err := s.GetRemainingClaimable(ctx, orderID)
	if err != nil {
		return Claim{}, err
	}
	if rem.RemainingClaimableMinor <= 0 {
		return Claim{}, fmt.Errorf("%w: order has no remaining claimable balance", ErrPricingFailed)
	}
	amountMinor = CapAmount(amountMinor, rem.RemainingClaimableMinor)

	stored := strings.TrimSpace(req.Currency)
	if stored == "" {
		stored = o.Currency
	}
	currency, err := claimCurrency(ctx, o.SupplierID, stored)
	if err != nil {
		return Claim{}, err
	}

	escalation, err := s.buildEscalationMetadata(ctx, o.RetailerID, amountMinor, prior, now)
	if err != nil {
		return Claim{}, err
	}

	c := Claim{
		ClaimID:     claimID,
		OrderID:     o.OrderID,
		SupplierID:  o.SupplierID,
		RetailerID:  o.RetailerID,
		FiledBy:     capturedBy,
		FiledByRole: string(claims.Role),
		ClaimType:   req.ClaimType,
		Status:      StatusOpen,
		Description: strings.TrimSpace(req.Description),
		AmountMinor: amountMinor,
		Currency:    currency,
		LineItems:   pricedLines,
		Evidences:   evs,
		Escalation:  escalation,
		Source:      SourceRetailerClaim,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	needsRL := claimNeedsStoreHold(c.ClaimType) || needsPhoto
	committed, err := s.commitClaimSingleTxn(ctx, c, true, capturedBy, o.WarehouseID, "", needsRL, now)
	if err != nil {
		return Claim{}, err
	}
	// Best-effort dock tickets for physical reverse logistics (damage types).
	s.openReverseLogistics(ctx, ReverseLogisticsInput{
		OrderID:     committed.OrderID,
		WarehouseID: o.WarehouseID,
		SupplierID:  committed.SupplierID,
		ClaimID:     committed.ClaimID,
		Source:      "RETAILER_CLAIM",
		Note:        string(committed.ClaimType),
		Lines:       pricedLines,
	})
	s.settleAutoApprovedClaim(ctx, committed)
	s.log.InfoContext(ctx, "claim filed", "claim_id", committed.ClaimID, "order_id", committed.OrderID, "type", committed.ClaimType, "status", committed.Status)
	return committed, nil
}

// CreateFromDriverException opens a claim from a driver OS&D exception report.
func (s *Service) CreateFromDriverException(ctx context.Context, o OrderSnapshot, driverID string, claimType ClaimType, lines []ClaimLine, photoURIs []string, note string) (Claim, error) {
	if s == nil || s.repo == nil {
		return Claim{}, fmt.Errorf("claims service unavailable")
	}
	if !claimType.Valid() {
		claimType = ClaimTypeMissing
	}
	now := s.now().UTC()
	claimID := s.newID()
	if !strings.HasPrefix(claimID, "clm_") {
		claimID = "clm_" + claimID
	}

	evInputs := make([]EvidenceInput, 0, len(photoURIs))
	for _, uri := range photoURIs {
		evInputs = append(evInputs, EvidenceInput{
			EvidenceType: EvidencePhoto,
			URI:          uri,
		})
	}
	// Bounded streaming GCS SHA-256 verification + fail-closed dedupe BEFORE Spanner RW txn.
	evs, _, err := s.inspectAndPrepareEvidences(ctx, claimID, evInputs, driverID, now, false)
	if err != nil {
		return Claim{}, err
	}

	var prior []Claim
	var priorClaimed map[string]int64
	if s.repo != nil && o.OrderID != "" {
		listed, lerr := s.repo.ListByOrder(ctx, o.OrderID)
		if lerr != nil {
			return Claim{}, lerr
		}
		prior = listed
		priorClaimed = ClaimedQtyBySKU(prior, "")
	}
	pricedLines := AggregateClaimLines(lines)
	var amount int64
	if len(o.LineItems) > 0 {
		pl, total, err := PriceClaimLinesWithPrior(o.LineItems, lines, priorClaimed)
		if err != nil {
			// Driver edges already amended the order; fail closed so we don't open
			// an unpriced liability row.
			return Claim{}, err
		}
		pricedLines = pl
		rem, err := s.GetRemainingClaimable(ctx, o.OrderID)
		if err != nil {
			return Claim{}, err
		}
		if rem.RemainingClaimableMinor <= 0 {
			return Claim{}, fmt.Errorf("%w: order has no remaining claimable balance", ErrPricingFailed)
		}
		amount = CapAmount(total, rem.RemainingClaimableMinor)
	}
	currency, err := claimCurrency(ctx, o.SupplierID, o.Currency)
	if err != nil {
		return Claim{}, err
	}
	escalation, err := s.buildEscalationMetadata(ctx, o.RetailerID, amount, prior, now)
	if err != nil {
		return Claim{}, err
	}
	c := Claim{
		ClaimID:     claimID,
		OrderID:     o.OrderID,
		SupplierID:  o.SupplierID,
		RetailerID:  o.RetailerID,
		FiledBy:     driverID,
		FiledByRole: string(auth.RoleDriver),
		ClaimType:   claimType,
		Status:      StatusOpen,
		Description: strings.TrimSpace(note),
		AmountMinor: amount,
		LineItems:   pricedLines,
		Evidences:   evs,
		Escalation:  escalation,
		Source:      SourceDriverException,
		Currency:    currency,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	committed, err := s.commitClaimSingleTxn(ctx, c, false, driverID, o.WarehouseID, driverID, claimNeedsStoreHold(claimType), now)
	if err != nil {
		return Claim{}, err
	}
	// Dedupe with amend-created returns when present; fill gaps for damage lines.
	s.openReverseLogistics(ctx, ReverseLogisticsInput{
		OrderID:     committed.OrderID,
		WarehouseID: o.WarehouseID,
		SupplierID:  committed.SupplierID,
		DriverID:    driverID,
		ClaimID:     committed.ClaimID,
		Source:      "DRIVER_EXCEPTION",
		Note:        strings.TrimSpace(note),
		Lines:       pricedLines,
	})
	s.settleAutoApprovedClaim(ctx, committed)
	return committed, nil
}

// ApproveClaim adjudicates an OPEN/UNDER_REVIEW claim, debits supplier ledger,
// and optionally refunds the retailer card via Global Pay (partial refund).
//
// Idempotent: re-approving a RESOLVED claim returns the prior settlement shape
// without double-charging (deterministic chargeback id = chargeback_<claimID>).
func (s *Service) ApproveClaim(ctx context.Context, actor auth.Claims, claimID string, req ApproveClaimRequest) (Claim, SettlementResult, error) {
	if s == nil || s.repo == nil {
		return Claim{}, SettlementResult{}, fmt.Errorf("claims service unavailable")
	}
	c, ok, err := s.repo.GetClaim(ctx, claimID)
	if err != nil {
		return Claim{}, SettlementResult{}, err
	}
	if !ok {
		return Claim{}, SettlementResult{}, ErrClaimNotFound
	}
	if !actorMaySettleClaim(actor, c) {
		return Claim{}, SettlementResult{}, ErrForbidden
	}

	// Idempotent replay: already settled.
	if c.Status == StatusResolved || c.Status == StatusApproved {
		settlement := SettlementResult{
			ChargebackID: DeterministicChargebackID(c.ClaimID),
			AmountMinor:  c.AmountMinor,
			Currency:     c.Currency,
			Mode:         "IDEMPOTENT_REPLAY",
			Idempotent:   true,
		}
		return c, settlement, nil
	}
	if c.Status == StatusRejected {
		return Claim{}, SettlementResult{}, fmt.Errorf("%w: status %s", ErrInvalidClaimState, c.Status)
	}
	if c.Status != StatusOpen && c.Status != StatusUnderReview {
		return Claim{}, SettlementResult{}, fmt.Errorf("%w: status %s", ErrInvalidClaimState, c.Status)
	}

	amount := c.AmountMinor
	if req.AmountMinor > 0 {
		if req.AmountMinor > c.AmountMinor && c.AmountMinor > 0 {
			return Claim{}, SettlementResult{}, fmt.Errorf("%w: approve amount exceeds priced claim", ErrPricingFailed)
		}
		amount = req.AmountMinor
	}
	
	rem, err := s.GetRemainingClaimable(ctx, c.OrderID, c.ClaimID)
	if err != nil {
		return Claim{}, SettlementResult{}, err
	}
	if amount > rem.RemainingClaimableMinor {
		return Claim{}, SettlementResult{}, fmt.Errorf("%w: approve amount %d exceeds remaining claimable %d", ErrPricingFailed, amount, rem.RemainingClaimableMinor)
	}

	if amount <= 0 {
		return Claim{}, SettlementResult{}, fmt.Errorf("%w: non-positive chargeback amount", ErrPricingFailed)
	}

	origStatus := c.Status
	origAmount := c.AmountMinor
	now := s.now().UTC()
	// CAS OPEN → UNDER_REVIEW so concurrent approves fail closed.
	// B1 M-P0-13: emit CLAIM_UNDER_REVIEW so consumers see the intermediate state
	// (settlement may still fail and leave the claim under review).
	c.Status = StatusUnderReview
	c.AmountMinor = amount
	c.ResolutionNote = strings.TrimSpace(req.ResolutionNote)
	c.UpdatedAt = now
	if err := s.repo.TransitionStatus(ctx, c.ClaimID, []Status{StatusOpen, StatusUnderReview}, c, func(txn outbox.TxnBuffer) error {
		payload := map[string]any{
			"type":            events.EventClaimUnderReview,
			"claim_id":        c.ClaimID,
			"order_id":        c.OrderID,
			"supplier_id":     c.SupplierID,
			"retailer_id":     c.RetailerID,
			"status":          string(StatusUnderReview),
			"amount_minor":    c.AmountMinor,
			"currency":        c.Currency,
			"resolution_note": c.ResolutionNote,
			"timestamp":       now.Format(time.RFC3339Nano),
		}
		if err := outbox.EmitJSON(ctx, txn, events.AggregateClaim, c.ClaimID, events.TopicExceptions, payload); err != nil {
			return err
		}
		return outbox.EmitJSON(ctx, txn, events.AggregateClaim, c.ClaimID, events.TopicMain, payload)
	}); err != nil {
		return Claim{}, SettlementResult{}, err
	}

	// Verify post-transition residual so concurrent approvals cannot exceed 100% of delivered gross.
	remPost, err := s.GetRemainingClaimable(ctx, c.OrderID, c.ClaimID)
	if err == nil && amount > remPost.RemainingClaimableMinor {
		rollbackClaim := c
		rollbackClaim.Status = origStatus
		rollbackClaim.AmountMinor = origAmount
		_ = s.repo.TransitionStatus(ctx, c.ClaimID, []Status{StatusUnderReview}, rollbackClaim, nil)
		return Claim{}, SettlementResult{}, fmt.Errorf("%w: approve amount %d exceeds remaining claimable %d (concurrent claim under review)", ErrPricingFailed, amount, remPost.RemainingClaimableMinor)
	}

	skipGP, wantStoreCredit := resolveSettlementFlags(req)
	var settlement SettlementResult
	if s.settler != nil {
		settlement, err = s.settler.SettleClaimChargeback(ctx, ClaimSettlement{
			ClaimID:           c.ClaimID,
			OrderID:           c.OrderID,
			SupplierID:        c.SupplierID,
			RetailerID:        c.RetailerID,
			AmountMinor:       amount,
			Currency:          c.Currency,
			LineItems:         c.LineItems,
			SkipGatewayRefund: skipGP,
		})
		if err != nil {
			// Leave UNDER_REVIEW for ops retry; do not flip to RESOLVED.
			return c, SettlementResult{}, fmt.Errorf("chargeback settlement: %w", err)
		}
	} else {
		settlement = SettlementResult{
			ChargebackID: DeterministicChargebackID(c.ClaimID),
			AmountMinor:  amount,
			Currency:     c.Currency,
			Mode:         "LEDGER_PENDING_SETTLER",
		}
	}
	if wantStoreCredit && s.storeCr != nil {
		if err := s.storeCr.ClearBalance(ctx, c.RetailerID, c.SupplierID, amount, c.OrderID); err != nil {
			s.log.WarnContext(ctx, "store credit clear balance failed", "err", err, "claim_id", c.ClaimID)
			if settlement.Mode == "LEDGER_ONLY" || settlement.Mode == "" {
				settlement.Mode = "LEDGER_ONLY_STORE_CREDIT_FAILED"
			}
		} else {
			settlement.Mode = "LEDGER_AND_STORE_CREDIT"
		}
	}

	// Terminal resolved after money movement recorded (CAS from UNDER_REVIEW).
	c.Status = StatusResolved
	c.ResolvedBy = actor.Subject
	c.ResolvedAt = &now
	c.UpdatedAt = now
	if err := s.repo.TransitionStatus(ctx, c.ClaimID, []Status{StatusUnderReview}, c, func(txn outbox.TxnBuffer) error {
		payload := map[string]any{
			"type":             events.EventClaimResolved,
			"claim_id":         c.ClaimID,
			"order_id":         c.OrderID,
			"supplier_id":      c.SupplierID,
			"retailer_id":      c.RetailerID,
			"status":           string(c.Status),
			"amount_minor":     c.AmountMinor,
			"currency":         c.Currency,
			"chargeback_id":    settlement.ChargebackID,
			"gateway":          settlement.Gateway,
			"gateway_refunded": settlement.GatewayRefunded,
			"provider_ref":     settlement.ProviderRef,
			"settlement_mode":  settlement.Mode,
			"line_items":       c.LineItems,
			"resolution_note":  c.ResolutionNote,
			"resolved_by":      c.ResolvedBy,
			"timestamp":        now.Format(time.RFC3339Nano),
		}
		if err := outbox.EmitJSON(ctx, txn, events.AggregateClaim, c.ClaimID, events.TopicExceptions, payload); err != nil {
			return err
		}
		return outbox.EmitJSON(ctx, txn, events.AggregateClaim, c.ClaimID, events.TopicMain, payload)
	}); err != nil {
		return Claim{}, SettlementResult{}, err
	}
	s.log.InfoContext(ctx, "claim approved and settled",
		"claim_id", c.ClaimID, "amount_minor", amount, "mode", settlement.Mode, "gateway_refunded", settlement.GatewayRefunded)
	if s.creditNotes != nil && creditNoteAutoFromClaimEnabled() {
		if err := s.creditNotes.CreateFromClaim(ctx, c.ClaimID, actor.Subject); err != nil {
			s.log.WarnContext(ctx, "auto credit note from claim failed", "err", err, "claim_id", c.ClaimID)
		}
	}
	// Leave quarantine for reverse logistics / waste after money settlement.
	disposition := "RETURN"
	if c.ClaimType == ClaimTypeDamaged || c.ClaimType == ClaimTypeConcealedDamage {
		disposition = "WASTE"
	}
	s.resolveStoreStockForClaim(ctx, c, disposition, actor.Subject)
	return c, settlement, nil
}

// DeterministicChargebackID makes approve retries safe against double ledger rows.
func DeterministicChargebackID(claimID string) string {
	claimID = strings.TrimSpace(claimID)
	if claimID == "" {
		return ""
	}
	if strings.HasPrefix(claimID, "clm_") {
		return "chargeback_" + claimID
	}
	return "chargeback_clm_" + claimID
}

// RejectClaim closes a claim without chargeback.
func (s *Service) RejectClaim(ctx context.Context, actor auth.Claims, claimID string, req RejectClaimRequest) (Claim, error) {
	if s == nil || s.repo == nil {
		return Claim{}, fmt.Errorf("claims service unavailable")
	}
	c, ok, err := s.repo.GetClaim(ctx, claimID)
	if err != nil {
		return Claim{}, err
	}
	if !ok {
		return Claim{}, ErrClaimNotFound
	}
	if !actorMaySettleClaim(actor, c) {
		return Claim{}, ErrForbidden
	}
	if c.Status != StatusOpen && c.Status != StatusUnderReview {
		return Claim{}, fmt.Errorf("%w: status %s", ErrInvalidClaimState, c.Status)
	}
	now := s.now().UTC()
	c.Status = StatusRejected
	c.ResolutionNote = strings.TrimSpace(req.ResolutionNote)
	c.ResolvedBy = actor.Subject
	c.ResolvedAt = &now
	c.UpdatedAt = now
	err = s.repo.TransitionStatus(ctx, c.ClaimID, []Status{StatusOpen, StatusUnderReview}, c, func(txn outbox.TxnBuffer) error {
		payload := map[string]any{
			"type":            events.EventClaimResolved,
			"claim_id":        c.ClaimID,
			"order_id":        c.OrderID,
			"status":          string(c.Status),
			"resolution_note": c.ResolutionNote,
			"resolved_by":     c.ResolvedBy,
			"timestamp":       now.Format(time.RFC3339Nano),
		}
		if err := outbox.EmitJSON(ctx, txn, events.AggregateClaim, c.ClaimID, events.TopicExceptions, payload); err != nil {
			return err
		}
		return outbox.EmitJSON(ctx, txn, events.AggregateClaim, c.ClaimID, events.TopicMain, payload)
	})
	if err == nil {
		// Restore sellable stock when claim is rejected.
		s.resolveStoreStockForClaim(ctx, c, "RESTORE", actor.Subject)
	}
	return c, err
}

// claimNeedsStoreHold is true for physical goods that may already be on store shelves.
func claimNeedsStoreHold(t ClaimType) bool {
	switch t {
	case ClaimTypeDamaged, ClaimTypeConcealedDamage, ClaimTypeTemperature, ClaimTypeTamper:
		return true
	default:
		return false // MISSING/OTHER: typically never received into store stock
	}
}

// holdStoreStockForClaim quarantines sellable stock for physical claim types.
// Returns error when the store-stock bridge is wired and HoldForClaim fails (G8 fail-closed).
// When the bridge is unset (unit tests / partial bootstrap), hold is skipped.
func (s *Service) holdStoreStockForClaim(ctx context.Context, c Claim, actor string) error {
	if s == nil || s.storeStock == nil || !claimNeedsStoreHold(c.ClaimType) {
		return nil
	}
	if err := s.storeStock.HoldForClaim(ctx, c.RetailerID, c.ClaimID, c.OrderID, c.LineItems, actor); err != nil {
		s.log.ErrorContext(ctx, "store claim hold failed", "claim_id", c.ClaimID, "err", err)
		return err
	}
	return nil
}

func (s *Service) resolveStoreStockForClaim(ctx context.Context, c Claim, disposition, actor string) {
	if s == nil || s.storeStock == nil || !claimNeedsStoreHold(c.ClaimType) {
		return
	}
	if err := s.storeStock.ResolveClaimStock(ctx, c.RetailerID, c.ClaimID, c.LineItems, disposition, actor); err != nil {
		s.log.WarnContext(ctx, "store claim stock resolve failed", "claim_id", c.ClaimID, "disposition", disposition, "err", err)
	}
}

// claimCurrency is empty-currency law: stored ISO code, else the shipped pack.
// Planned/unknown packs fail closed — never invent UZS.
func claimCurrency(ctx context.Context, supplierID, stored string) (string, error) {
	c, err := auth.CoalesceCurrency(ctx, supplierID, stored)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(c) == "" {
		return "", auth.ErrMarketPackNotShipped
	}
	return c, nil
}

// OrderLookup returns the order lookup interface.
func (s *Service) OrderLookup() OrderLookup {
	return s.orders
}
