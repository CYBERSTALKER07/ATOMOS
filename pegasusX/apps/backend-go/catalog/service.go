package catalog

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/promotion"
)

// Service implements catalog business logic. Handlers call service; service
// calls repository.
type Service struct {
	repo       Repository
	cache      *cache.Cache
	log        *slog.Logger
	promotions *promotion.Service
	stock      *StockEnricher
}

// NewService creates a catalog service with the given dependencies.
func NewService(repo Repository, c *cache.Cache, log *slog.Logger, promotions *promotion.Service, stock *StockEnricher) *Service {
	return &Service{repo: repo, cache: c, log: log, promotions: promotions, stock: stock}
}

// ListCategories returns all categories for a supplier.
func (s *Service) ListCategories(ctx context.Context, supplierID string) ([]Category, error) {
	cats, err := s.repo.ListCategories(ctx, supplierID)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	if cats == nil {
		cats = []Category{}
	}
	return cats, nil
}

// GetCategory returns a single category.
func (s *Service) GetCategory(ctx context.Context, categoryID string) (*Category, error) {
	return s.repo.GetCategory(ctx, categoryID)
}

// CreateCategory adds a new category and invalidates the cache.
func (s *Service) CreateCategory(ctx context.Context, cat Category) error {
	if err := s.repo.CreateCategory(ctx, cat); err != nil {
		return err
	}
	if s.cache != nil {
		s.cache.Invalidate(ctx, "catalog:categories:"+cat.SupplierID)
	}
	return nil
}

// RetailerProduct is a catalog row plus optional promotion metadata.
type RetailerProduct struct {
	Product
	Offer            *promotion.ProductOffer `json:"offer,omitempty"`
	AvailableStock   *int64                  `json:"available_stock,omitempty"`
	IsOutOfStock     bool                    `json:"is_out_of_stock,omitempty"`
	AcceptsBackorder bool                    `json:"accepts_backorder,omitempty"`
}

// ListProducts returns products for a supplier, optionally filtered.
func (s *Service) ListProducts(ctx context.Context, supplierID, categoryID string, activeOnly bool) ([]Product, error) {
	products, err := s.repo.ListProducts(ctx, supplierID, categoryID, activeOnly)
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}
	if products == nil {
		products = []Product{}
	}
	return products, nil
}

// ListProductsForRetailer returns catalog rows enriched with promotion offers.
func (s *Service) ListProductsForRetailer(ctx context.Context, supplierID, retailerID, categoryID string) ([]RetailerProduct, error) {
	products, err := s.ListProducts(ctx, supplierID, categoryID, true)
	if err != nil {
		return nil, err
	}
	return s.enrichProductsForRetailer(ctx, retailerID, products)
}

// ListProductsDiscovery returns active catalog rows across suppliers for retailer browse surfaces.
func (s *Service) ListProductsDiscovery(ctx context.Context, retailerID, categoryID string, limit, offset int64) ([]RetailerProduct, error) {
	products, err := s.repo.ListDiscoverableProducts(ctx, categoryID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list discoverable products: %w", err)
	}
	return s.enrichProductsForRetailer(ctx, retailerID, products)
}

func (s *Service) enrichProductsForRetailer(ctx context.Context, retailerID string, products []Product) ([]RetailerProduct, error) {
	if len(products) == 0 {
		return []RetailerProduct{}, nil
	}
	if s.promotions == nil || retailerID == "" {
		out := make([]RetailerProduct, len(products))
		for i, p := range products {
			out[i] = RetailerProduct{Product: p}
		}
		return out, nil
	}
	promosBySupplier := make(map[string][]promotion.Promotion)
	now := s.promotions.Now()
	out := make([]RetailerProduct, len(products))
	for i, p := range products {
		promos, ok := promosBySupplier[p.SupplierID]
		if !ok {
			var err error
			promos, err = s.promotions.ActiveForSupplier(ctx, p.SupplierID)
			if err != nil {
				return nil, fmt.Errorf("active promotions for supplier %s: %w", p.SupplierID, err)
			}
			promosBySupplier[p.SupplierID] = promos
		}
		listPrice := p.PriceMinor
		isOverride := false
		if s.promotions != nil && retailerID != "" {
			resolved, overridden, err := s.promotions.ResolveListPrice(ctx, p.SupplierID, retailerID, p.ProductID, p.PriceMinor)
			if err != nil {
				return nil, fmt.Errorf("resolve retailer price override: %w", err)
			}
			listPrice = resolved
			isOverride = overridden
		}
		var offer promotion.ProductOffer
		if isOverride && listPrice < p.PriceMinor {
			sale := listPrice
			offer = promotion.ProductOffer{
				ProductID:      p.ProductID,
				ListPriceMinor: p.PriceMinor,
				SalePriceMinor: &sale,
			}
		} else {
			offer = promotion.CatalogOffer(now, retailerID, p.ProductID, p.CategoryID, listPrice, promos)
		}
		out[i] = RetailerProduct{Product: p, Offer: &offer}
	}
	if s.stock != nil && retailerID != "" {
		snaps := s.stock.Enrich(ctx, retailerID, products)
		for i := range out {
			key := out[i].SupplierID + ":" + out[i].ProductID
			if snap, ok := snaps[key]; ok {
				avail := snap.AvailableStock
				out[i].AvailableStock = &avail
				out[i].IsOutOfStock = snap.IsOutOfStock
				out[i].AcceptsBackorder = snap.AcceptsBackorder
			}
		}
	}
	return out, nil
}

// GetProduct returns a single product.
func (s *Service) GetProduct(ctx context.Context, productID string) (*Product, error) {
	return s.repo.GetProduct(ctx, productID)
}

// CreateProduct adds a new product and invalidates the cache.
func (s *Service) CreateProduct(ctx context.Context, p Product) error {
	if err := s.repo.CreateProduct(ctx, p); err != nil {
		return err
	}
	if s.cache != nil {
		s.cache.Invalidate(ctx, "catalog:products:"+p.SupplierID)
	}
	return nil
}

// SearchSuppliers returns suppliers matching a name query.
func (s *Service) SearchSuppliers(ctx context.Context, query string) ([]CategorySupplier, error) {
	query = strings.TrimSpace(query)
	if len(query) < 2 {
		return []CategorySupplier{}, nil
	}
	suppliers, err := s.repo.SearchSuppliers(ctx, query, 20)
	if err != nil {
		return nil, fmt.Errorf("search suppliers: %w", err)
	}
	if suppliers == nil {
		suppliers = []CategorySupplier{}
	}
	return suppliers, nil
}

// ListCategorySuppliers returns suppliers stocking products in a category.
func (s *Service) ListCategorySuppliers(ctx context.Context, categoryID string) ([]CategorySupplier, error) {
	suppliers, err := s.repo.ListCategorySuppliers(ctx, categoryID)
	if err != nil {
		return nil, fmt.Errorf("list category suppliers: %w", err)
	}
	if suppliers == nil {
		suppliers = []CategorySupplier{}
	}
	return suppliers, nil
}

// UpdateProduct modifies an existing product with optimistic concurrency.
func (s *Service) UpdateProduct(ctx context.Context, p Product) error {
	if err := s.repo.UpdateProduct(ctx, p); err != nil {
		return err
	}
	if s.cache != nil {
		s.cache.Invalidate(ctx,
			"catalog:products:"+p.SupplierID,
			"catalog:product:"+p.ProductID,
		)
	}
	return nil
}
