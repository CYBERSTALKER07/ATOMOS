package promotion

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
)

// Service implements promotion business logic.
type Service struct {
	repo        Repository
	cache       *cache.Cache
	idem        idempotency.Store
	log         *slog.Logger
	now         func() time.Time
	retailerHub *ws.Hub
}

// NewService constructs a promotion service.
func NewService(repo Repository, c *cache.Cache, idem idempotency.Store, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{repo: repo, cache: c, idem: idem, log: log, now: func() time.Time { return time.Now().UTC() }}
}

// ListForSupplier returns all promotions for the supplier portal.
func (s *Service) ListForSupplier(ctx context.Context, supplierID string) ([]Promotion, error) {
	if s.repo == nil {
		return []Promotion{}, nil
	}
	items, err := s.repo.ListBySupplier(ctx, supplierID)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []Promotion{}
	}
	return items, nil
}

// ActiveForSupplier returns promotions eligible for evaluation at now.
func (s *Service) ActiveForSupplier(ctx context.Context, supplierID string) ([]Promotion, error) {
	if s.repo == nil {
		return []Promotion{}, nil
	}
	return s.repo.ListActiveBySupplier(ctx, supplierID, s.now())
}

// QuoteCheckout prices cart lines with server-authoritative promotions.
func (s *Service) QuoteCheckout(ctx context.Context, supplierID, retailerID string, lines []LineInput) (QuoteResult, error) {
	pack, err := auth.CheckoutPackFromContext(ctx)
	if err != nil {
		return QuoteResult{}, err
	}
	for _, line := range lines {
		if c := strings.TrimSpace(line.Currency); c != "" {
			if _, err := auth.ResolveCheckoutCurrency(pack, c); err != nil {
				return QuoteResult{}, err
			}
		}
	}
	lines, err = s.enrichLines(ctx, supplierID, retailerID, lines)
	if err != nil {
		return QuoteResult{}, err
	}
	promotions, err := s.ActiveForSupplier(ctx, supplierID)
	if err != nil {
		return QuoteResult{}, fmt.Errorf("load promotions: %w", err)
	}
	if s.repo != nil {
		promotions, err = s.repo.FilterEligibleCampaignPromotions(ctx, retailerID, promotions)
		if err != nil {
			return QuoteResult{}, fmt.Errorf("filter campaign promotions: %w", err)
		}
	}
	quote, err := ApplyQuote(s.now(), supplierID, retailerID, lines, promotions)
	if err != nil {
		return QuoteResult{}, err
	}
	quote.Currency = pack.CurrencyCode
	quote.MarketCode = pack.Code
	for i := range quote.Lines {
		quote.Lines[i].Currency = pack.CurrencyCode
	}
	return quote, nil
}

// ResolveListPrice applies an active per-retailer override when present.
func (s *Service) ResolveListPrice(ctx context.Context, supplierID, retailerID, productID string, listPrice int64) (int64, bool, error) {
	if s.repo == nil || retailerID == "" || productID == "" {
		return listPrice, false, nil
	}
	overrides, err := s.repo.LookupActivePriceOverrides(ctx, supplierID, retailerID, []string{productID}, s.now())
	if err != nil {
		return listPrice, false, err
	}
	if price, ok := overrides[productID]; ok && price > 0 {
		return price, true, nil
	}
	return listPrice, false, nil
}

func (s *Service) enrichLines(ctx context.Context, supplierID, retailerID string, lines []LineInput) ([]LineInput, error) {
	if s.repo == nil || len(lines) == 0 {
		return lines, nil
	}
	ids := make([]string, 0, len(lines))
	for _, line := range lines {
		if line.ProductID != "" {
			ids = append(ids, line.ProductID)
		}
	}
	categories, err := s.repo.LookupProductCategories(ctx, ids)
	if err != nil {
		return nil, err
	}
	listPrices, err := s.repo.LookupListPrices(ctx, ids)
	if err != nil {
		return nil, err
	}
	overrides, err := s.repo.LookupActivePriceOverrides(ctx, supplierID, retailerID, ids, s.now())
	if err != nil {
		return nil, err
	}
	out := make([]LineInput, len(lines))
	for i, line := range lines {
		out[i] = line
		if out[i].CategoryID == "" {
			out[i].CategoryID = categories[line.ProductID]
		}
		if listPrice, ok := listPrices[line.ProductID]; ok && listPrice > 0 {
			out[i].UnitPrice = listPrice
		}
		if overridePrice, ok := overrides[line.ProductID]; ok && overridePrice > 0 {
			out[i].UnitPrice = overridePrice
			out[i].PriceIsOverride = true
		}
	}
	return out, nil
}

// CreatePromotion validates and persists a new promotion.
func (s *Service) CreatePromotion(ctx context.Context, p Promotion) (Promotion, error) {
	if err := validatePromotion(p); err != nil {
		return Promotion{}, err
	}
	if p.PromotionID == "" {
		p.PromotionID = uuid.NewString()
	}
	p.Version = 1
	p.IsActive = true
	if err := s.repo.Upsert(ctx, p, func(buf *spannerTxnBuffer) error {
		return emitPromotionChanged(ctx, buf, p, "created")
	}); err != nil {
		return Promotion{}, err
	}
	s.invalidate(ctx, p.SupplierID)
	return p, nil
}

// UpdatePromotion replaces an existing promotion with version bump.
func (s *Service) UpdatePromotion(ctx context.Context, p Promotion) (Promotion, error) {
	if err := validatePromotion(p); err != nil {
		return Promotion{}, err
	}
	current, found, err := s.repo.GetByID(ctx, p.SupplierID, p.PromotionID)
	if err != nil {
		return Promotion{}, err
	}
	if !found {
		return Promotion{}, fmt.Errorf("promotion not found")
	}
	p.Version = current.Version + 1
	if err := s.repo.Upsert(ctx, p, func(buf *spannerTxnBuffer) error {
		return emitPromotionChanged(ctx, buf, p, "updated")
	}); err != nil {
		return Promotion{}, err
	}
	s.invalidate(ctx, p.SupplierID)
	return p, nil
}

// DeactivatePromotion disables a promotion immediately.
func (s *Service) DeactivatePromotion(ctx context.Context, supplierID, promotionID string) error {
	current, found, err := s.repo.GetByID(ctx, supplierID, promotionID)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("promotion not found")
	}
	deactivated := current
	deactivated.IsActive = false
	deactivated.Version = current.Version + 1
	if err := s.repo.SetActive(ctx, supplierID, promotionID, false, current.Version, func(buf *spannerTxnBuffer) error {
		return emitPromotionChanged(ctx, buf, deactivated, "deactivated")
	}); err != nil {
		return err
	}
	s.invalidate(ctx, supplierID)
	return nil
}

func validatePromotion(p Promotion) error {
	if strings.TrimSpace(p.SupplierID) == "" {
		return fmt.Errorf("supplier_id required")
	}
	if strings.TrimSpace(p.Name) == "" {
		return fmt.Errorf("name required")
	}
	if len(p.Tiers) == 0 {
		return fmt.Errorf("at least one promotion tier required")
	}
	for i, tier := range p.Tiers {
		if tier.DiscountBps <= 0 || tier.DiscountBps > maxDiscountBps {
			return fmt.Errorf("tier %d discount_bps out of range", i)
		}
		if tier.MinQuantity < 0 {
			return fmt.Errorf("tier %d min_quantity cannot be negative", i)
		}
	}
	switch p.ScopeType {
	case ScopeTypeProduct:
		if strings.TrimSpace(p.ScopeProductID) == "" {
			return fmt.Errorf("scope_product_id required")
		}
	case ScopeTypeCategory:
		if strings.TrimSpace(p.ScopeCategoryID) == "" {
			return fmt.Errorf("scope_category_id required")
		}
	case ScopeTypeAllProducts:
	default:
		return fmt.Errorf("invalid scope_type")
	}
	if p.RetailerScope == RetailerScopeAllowlist && len(p.RetailerIDs) == 0 {
		return fmt.Errorf("retailer_ids required for allowlist scope")
	}
	return nil
}

// Now returns the service clock (UTC), used by catalog enrichment.
func (s *Service) Now() time.Time {
	return s.now()
}

func (s *Service) invalidate(ctx context.Context, supplierID string) {
	if s.cache == nil {
		return
	}
	s.cache.Invalidate(ctx, "promotions:supplier:"+supplierID)
}

// RedeemPromotion atomically increments a promotion's redemption count.
func (s *Service) RedeemPromotion(ctx context.Context, promotionID string) error {
	return s.repo.RedeemPromotion(ctx, promotionID)
}
