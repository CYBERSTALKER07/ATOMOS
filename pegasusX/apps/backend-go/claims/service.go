package claims

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
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
}

// OrderLookup loads order snapshots for claim authorization.
type OrderLookup interface {
	GetOrder(ctx context.Context, orderID string) (OrderSnapshot, bool, error)
}

// Service is the claims domain application service.
type Service struct {
	repo     Repository
	orders   OrderLookup
	settler  ChargebackSettler
	now      func() time.Time
	newID    func() string
	log      *slog.Logger
	window   time.Duration
}

// Config wires claims Service dependencies.
type Config struct {
	Repo     Repository
	Orders   OrderLookup
	Settler  ChargebackSettler
	Now      func() time.Time
	NewID    func() string
	Log      *slog.Logger
	Window   time.Duration
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
			return fmt.Sprintf("clm_%d", time.Now().UnixNano())
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
	return &Service{repo: cfg.Repo, orders: cfg.Orders, settler: cfg.Settler, now: now, newID: newID, log: log, window: window}
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
		if strings.TrimSpace(o.RetailerID) != strings.TrimSpace(actor.Subject) {
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

// ListSupplierClaims is the supplier HQ adjudication queue.
func (s *Service) ListSupplierClaims(ctx context.Context, actor auth.Claims, status Status, limit int) ([]Claim, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("claims service unavailable")
	}
	if actor.Role != auth.RoleAdmin && actor.Role != auth.RoleWarehouseAdmin {
		return nil, ErrForbidden
	}
	supplierID := strings.TrimSpace(actor.SupplierID)
	if supplierID == "" {
		if sid, ok := auth.ResolveSupplierID(ctx); ok {
			supplierID = sid
		}
	}
	if supplierID == "" {
		return nil, ErrForbidden
	}
	return s.repo.ListBySupplier(ctx, supplierID, status, limit)
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
	hasPhoto := false
	for _, e := range req.Evidences {
		if e.EvidenceType == EvidencePhoto && strings.TrimSpace(e.URI) != "" {
			hasPhoto = true
			break
		}
	}
	if needsPhoto && !hasPhoto {
		return Claim{}, ErrEvidenceRequired
	}

	o, ok, err := s.orders.GetOrder(ctx, orderID)
	if err != nil {
		return Claim{}, err
	}
	if !ok {
		return Claim{}, ErrOrderNotFound
	}
	if claims.Role == auth.RoleRetailer && strings.TrimSpace(o.RetailerID) != strings.TrimSpace(claims.Subject) {
		return Claim{}, ErrForbidden
	}
	if strings.ToUpper(strings.TrimSpace(o.Status)) != OrderStatusCompleted {
		return Claim{}, fmt.Errorf("%w: order status %s (claims only after delivery complete)", ErrClaimNotAllowed, o.Status)
	}
	// Window starts at COMPLETED (delivery+handshake+fiscal), not at first ship day.
	completedAt := o.UpdatedAt
	if completedAt.IsZero() {
		completedAt = o.CreatedAt
	}
	now := s.now().UTC()
	if !completedAt.IsZero() && now.After(completedAt.UTC().Add(s.window)) {
		return Claim{}, ErrClaimWindowExpired
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
	// Prefer original commercial total when present (post-amend TotalMinor may be lower).
	ceiling := o.OriginalTotalMinor
	if ceiling <= 0 {
		ceiling = o.TotalMinor
	}
	amountMinor = CapAmount(amountMinor, ceiling)

	currency := strings.TrimSpace(req.Currency)
	if currency == "" {
		currency = o.Currency
	}
	if currency == "" {
		currency = "UZS"
	}

	claimID := s.newID()
	if !strings.HasPrefix(claimID, "clm_") {
		claimID = "clm_" + claimID
	}
	evs := make([]Evidence, 0, len(req.Evidences))
	for _, e := range req.Evidences {
		uri := strings.TrimSpace(e.URI)
		if uri == "" {
			continue
		}
		et := e.EvidenceType
		if et == "" {
			et = EvidencePhoto
		}
		evs = append(evs, Evidence{
			EvidenceID:   s.newID(),
			ClaimID:      claimID,
			EvidenceType: et,
			URI:          uri,
			MimeType:     strings.TrimSpace(e.MimeType),
			CapturedBy:   claims.Subject,
			CreatedAt:    now,
		})
	}

	c := Claim{
		ClaimID:     claimID,
		OrderID:     o.OrderID,
		SupplierID:  o.SupplierID,
		RetailerID:  o.RetailerID,
		FiledBy:     claims.Subject,
		FiledByRole: string(claims.Role),
		ClaimType:   req.ClaimType,
		Status:      StatusOpen,
		Description: strings.TrimSpace(req.Description),
		AmountMinor: amountMinor,
		Currency:    currency,
		LineItems:   pricedLines,
		Evidences:   evs,
		Source:      SourceRetailerClaim,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	err = s.repo.CreateClaim(ctx, c, func(txn outbox.TxnBuffer) error {
		payload := map[string]any{
			"type":         events.EventClaimFiled,
			"claim_id":     c.ClaimID,
			"order_id":     c.OrderID,
			"supplier_id":  c.SupplierID,
			"retailer_id":  c.RetailerID,
			"claim_type":   string(c.ClaimType),
			"status":       string(c.Status),
			"source":       string(c.Source),
			"amount_minor": c.AmountMinor,
			"currency":     c.Currency,
			"line_items":   c.LineItems,
			"timestamp":    now.Format(time.RFC3339Nano),
		}
		// Exceptions topic (new) + main (existing consumers).
		if err := outbox.EmitJSON(ctx, txn, events.AggregateClaim, c.ClaimID, events.TopicExceptions, payload); err != nil {
			return err
		}
		if err := outbox.EmitJSON(ctx, txn, events.AggregateClaim, c.ClaimID, events.TopicMain, payload); err != nil {
			return err
		}
		if needsPhoto {
			return outbox.EmitJSON(ctx, txn, events.AggregateOrder, c.OrderID, events.TopicExceptions, map[string]any{
				"type":         events.EventReverseLogisticsRequired,
				"claim_id":     c.ClaimID,
				"order_id":     c.OrderID,
				"warehouse_id": o.WarehouseID,
				"supplier_id":  c.SupplierID,
				"retailer_id":  c.RetailerID,
				"claim_type":   string(c.ClaimType),
				"timestamp":    now.Format(time.RFC3339Nano),
			})
		}
		return nil
	})
	if err != nil {
		return Claim{}, err
	}
	s.log.InfoContext(ctx, "claim filed", "claim_id", c.ClaimID, "order_id", c.OrderID, "type", c.ClaimType)
	return c, nil
}

// CreateFromDriverException opens a claim from a driver OS&D exception report.
func (s *Service) CreateFromDriverException(ctx context.Context, o OrderSnapshot, driverID string, claimType ClaimType, lines []ClaimLine, photoURIs []string, note string) (Claim, error) {
	if s == nil || s.repo == nil {
		return Claim{}, fmt.Errorf("claims service unavailable")
	}
	if !claimType.Valid() {
		claimType = ClaimTypeMissing
	}
	var priorClaimed map[string]int64
	if s.repo != nil && o.OrderID != "" {
		if prior, err := s.repo.ListByOrder(ctx, o.OrderID); err == nil {
			priorClaimed = ClaimedQtyBySKU(prior, "")
		}
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
		ceiling := o.OriginalTotalMinor
		if ceiling <= 0 {
			ceiling = o.TotalMinor
		}
		amount = CapAmount(total, ceiling)
	}
	now := s.now().UTC()
	claimID := "clm_" + s.newID()
	evs := make([]Evidence, 0, len(photoURIs))
	for _, uri := range photoURIs {
		uri = strings.TrimSpace(uri)
		if uri == "" {
			continue
		}
		evs = append(evs, Evidence{
			EvidenceID:   s.newID(),
			ClaimID:      claimID,
			EvidenceType: EvidencePhoto,
			URI:          uri,
			CapturedBy:   driverID,
			CreatedAt:    now,
		})
	}
	currency := o.Currency
	if currency == "" {
		currency = "UZS"
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
		Source:      SourceDriverException,
		Currency:    currency,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	err := s.repo.CreateClaim(ctx, c, func(txn outbox.TxnBuffer) error {
		payload := map[string]any{
			"type":         events.EventClaimFiled,
			"claim_id":     c.ClaimID,
			"order_id":     c.OrderID,
			"source":       string(c.Source),
			"claim_type":   string(c.ClaimType),
			"driver_id":    driverID,
			"amount_minor": c.AmountMinor,
			"line_items":   pricedLines,
			"photo_count":  len(evs),
			"timestamp":    now.Format(time.RFC3339Nano),
		}
		if err := outbox.EmitJSON(ctx, txn, events.AggregateClaim, c.ClaimID, events.TopicExceptions, payload); err != nil {
			return err
		}
		return outbox.EmitJSON(ctx, txn, events.AggregateClaim, c.ClaimID, events.TopicMain, payload)
	})
	return c, err
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
	if amount <= 0 {
		return Claim{}, SettlementResult{}, fmt.Errorf("%w: non-positive chargeback amount", ErrPricingFailed)
	}

	now := s.now().UTC()
	// CAS OPEN → UNDER_REVIEW so concurrent approves fail closed.
	c.Status = StatusUnderReview
	c.AmountMinor = amount
	c.ResolutionNote = strings.TrimSpace(req.ResolutionNote)
	c.UpdatedAt = now
	if err := s.repo.TransitionStatus(ctx, c.ClaimID, []Status{StatusOpen, StatusUnderReview}, c, nil); err != nil {
		return Claim{}, SettlementResult{}, err
	}

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
			SkipGatewayRefund: req.SkipGatewayRefund,
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
	return c, err
}
