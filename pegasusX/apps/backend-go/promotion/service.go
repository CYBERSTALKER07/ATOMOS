package promotion

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
)

// Service implements promotion business logic.
type Service struct {
	repo  Repository
	cache *cache.Cache
	log   *slog.Logger
	now   func() time.Time
}

// NewService constructs a promotion service.
func NewService(repo Repository, c *cache.Cache, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{repo: repo, cache: c, log: log, now: func() time.Time { return time.Now().UTC() }}
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
	lines, err := s.enrichLines(ctx, lines)
	if err != nil {
		return QuoteResult{}, err
	}
	promotions, err := s.ActiveForSupplier(ctx, supplierID)
	if err != nil {
		return QuoteResult{}, fmt.Errorf("load promotions: %w", err)
	}
	return ApplyQuote(s.now(), supplierID, retailerID, lines, promotions)
}

func (s *Service) enrichLines(ctx context.Context, lines []LineInput) ([]LineInput, error) {
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
	out := make([]LineInput, len(lines))
	for i, line := range lines {
		out[i] = line
		if out[i].CategoryID == "" {
			out[i].CategoryID = categories[line.ProductID]
		}
		if listPrice, ok := listPrices[line.ProductID]; ok && listPrice > 0 {
			out[i].UnitPrice = listPrice
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
	if p.DiscountBps <= 0 || p.DiscountBps > maxDiscountBps {
		return fmt.Errorf("discount_bps out of range")
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
