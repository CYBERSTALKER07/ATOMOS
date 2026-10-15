package catalog

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
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
	globalHook ProductUpsertHook
	idem       idempotency.Store
}

// ProductUpsertHook is invoked after successful product create/update (GlobalProducts).
type ProductUpsertHook interface {
	OnProductUpserted(ctx context.Context, productID, supplierID, name, barcode string, priceMinor int64, currency string, unitsPerPack int64) error
}

// SetGlobalProductHook wires the optional GlobalProducts linker.
func (s *Service) SetGlobalProductHook(h ProductUpsertHook) {
	s.globalHook = h
}

// SetIdempotencyStore wires the shared idempotency store used to honor Idempotency-Key.
func (s *Service) SetIdempotencyStore(store idempotency.Store) {
	if s == nil {
		return
	}
	s.idem = store
}

// IdempotencyStore exposes the configured store.
func (s *Service) IdempotencyStore() idempotency.Store {
	if s == nil {
		return nil
	}
	return s.idem
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

// CreateCategory adds a new category, emits CATEGORY_CREATED, and invalidates the cache.
func (s *Service) CreateCategory(ctx context.Context, cat Category) error {
	emit := func(txn outbox.TxnBuffer) error {
		catEvt := events.CategoryEvent{
			BaseEvent: events.BaseEvent{
				Type:      events.EventCategoryCreated,
				Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
			},
			CategoryID:       cat.CategoryID,
			SupplierID:       cat.SupplierID,
			Name:             cat.Name,
			ParentCategoryID: cat.ParentCategoryID,
			IconKey:          cat.IconKey,
			SortOrder:        cat.SortOrder,
		}
		return outbox.EmitJSON(ctx, txn, events.AggregateCategory, cat.CategoryID, events.TopicMain, catEvt)
	}
	if err := s.repo.CreateCategory(ctx, cat, emit); err != nil {
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
	ShowStockCounts  bool                    `json:"show_stock_counts,omitempty"`
	MaxQuantity      *int64                  `json:"max_quantity,omitempty"`
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
func (s *Service) ListProductsDiscovery(ctx context.Context, retailerID, categoryID, marketCode string, limit, offset int64) ([]RetailerProduct, error) {
	products, err := s.repo.ListDiscoverableProducts(ctx, categoryID, marketCode, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list discoverable products: %w", err)
	}
	return s.enrichProductsForRetailer(ctx, retailerID, products)
}

func (s *Service) enrichProductsForRetailer(ctx context.Context, retailerID string, products []Product) ([]RetailerProduct, error) {
	if len(products) == 0 {
		return []RetailerProduct{}, nil
	}

	out := make([]RetailerProduct, len(products))
	for i, p := range products {
		out[i] = RetailerProduct{Product: p}
	}

	if s.promotions != nil && retailerID != "" {
		promosBySupplier := make(map[string][]promotion.Promotion)
		now := s.promotions.Now()
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
			resolved, overridden, err := s.promotions.ResolveListPrice(ctx, p.SupplierID, retailerID, p.ProductID, p.PriceMinor)
			if err != nil {
				return nil, fmt.Errorf("resolve retailer price override: %w", err)
			}
			listPrice = resolved
			isOverride = overridden
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
			out[i].Offer = &offer
		}
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
				out[i].ShowStockCounts = snap.ShowStockCounts
				if !snap.AcceptsBackorder {
					maxQ := avail
					out[i].MaxQuantity = &maxQ
				}
			}
		}
	}
	return out, nil
}

// GetProduct returns a single product.
func (s *Service) GetProduct(ctx context.Context, productID string) (*Product, error) {
	return s.repo.GetProduct(ctx, productID)
}

// CreateProduct adds a new product, emits PRODUCT_CREATED and PRODUCT_HANDLING_UPDATED, and invalidates the cache.
func (s *Service) CreateProduct(ctx context.Context, p Product) error {
	emit := func(txn outbox.TxnBuffer) error {
		createdEvt := events.ProductEvent{
			BaseEvent: events.BaseEvent{
				Type:      events.EventProductCreated,
				Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
			},
			ProductID:         p.ProductID,
			SupplierID:        p.SupplierID,
			HandlingClass:     string(p.HandlingClass),
			RequiresColdChain: p.RequiresColdChain,
			IsHazardous:       p.IsHazardous,
			IsPerishable:      p.IsPerishable,
			StorageTempMinC:   p.StorageTempMinC,
			StorageTempMaxC:   p.StorageTempMaxC,
			Name:              p.Name,
			CategoryID:        p.CategoryID,
			PriceMinor:        p.PriceMinor,
			Currency:          p.Currency,
			StockQuantity:     p.StockQuantity,
			Unit:              p.Unit,
			Barcode:           p.Barcode,
			IsActive:          p.IsActive,
			Version:           1,
		}
		if err := outbox.EmitJSON(ctx, txn, events.AggregateProduct, p.ProductID, events.TopicMain, createdEvt); err != nil {
			return err
		}
		return outbox.EmitJSON(ctx, txn, events.AggregateProduct, p.ProductID, events.TopicMain, productHandlingEvent(p))
	}
	if err := s.repo.CreateProduct(ctx, p, emit); err != nil {
		return err
	}
	if s.cache != nil {
		s.cache.Invalidate(ctx, "catalog:products:"+p.SupplierID)
	}
	s.notifyGlobalHook(ctx, p)
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

// UpdateProduct modifies an existing product with optimistic concurrency and emits lifecycle outbox events.
func (s *Service) UpdateProduct(ctx context.Context, p Product) error {
	prev, _ := s.repo.GetProduct(ctx, p.ProductID)
	var prevPrice int64
	if prev != nil {
		prevPrice = prev.PriceMinor
	}

	emit := func(txn outbox.TxnBuffer) error {
		eventType := events.EventProductUpdated
		if prev != nil && prev.PriceMinor != p.PriceMinor {
			eventType = events.EventProductPriceChanged
		}
		evt := events.ProductEvent{
			BaseEvent: events.BaseEvent{
				Type:      eventType,
				Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
			},
			ProductID:          p.ProductID,
			SupplierID:         p.SupplierID,
			HandlingClass:      string(p.HandlingClass),
			RequiresColdChain:  p.RequiresColdChain,
			IsHazardous:        p.IsHazardous,
			IsPerishable:       p.IsPerishable,
			StorageTempMinC:    p.StorageTempMinC,
			StorageTempMaxC:    p.StorageTempMaxC,
			Name:               p.Name,
			CategoryID:         p.CategoryID,
			PriceMinor:         p.PriceMinor,
			PreviousPriceMinor: prevPrice,
			Currency:           p.Currency,
			StockQuantity:      p.StockQuantity,
			Unit:               p.Unit,
			Barcode:            p.Barcode,
			IsActive:           p.IsActive,
			Version:            p.Version + 1,
		}
		if err := outbox.EmitJSON(ctx, txn, events.AggregateProduct, p.ProductID, events.TopicMain, evt); err != nil {
			return err
		}

		if prev != nil && prev.IsActive && !p.IsActive {
			deactEvt := evt
			deactEvt.BaseEvent.Type = events.EventProductDeactivated
			deactEvt.BaseEvent.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)
			if err := outbox.EmitJSON(ctx, txn, events.AggregateProduct, p.ProductID, events.TopicMain, deactEvt); err != nil {
				return err
			}
		}

		if prev == nil || prev.HandlingClass != p.HandlingClass || prev.RequiresColdChain != p.RequiresColdChain || prev.IsHazardous != p.IsHazardous || prev.IsPerishable != p.IsPerishable {
			if err := outbox.EmitJSON(ctx, txn, events.AggregateProduct, p.ProductID, events.TopicMain, productHandlingEvent(p)); err != nil {
				return err
			}
		}
		return nil
	}
	if err := s.repo.UpdateProduct(ctx, p, emit); err != nil {
		return err
	}
	if s.cache != nil {
		s.cache.Invalidate(ctx,
			"catalog:products:"+p.SupplierID,
			"catalog:product:"+p.ProductID,
		)
	}
	s.notifyGlobalHook(ctx, p)
	return nil
}

func (s *Service) notifyGlobalHook(ctx context.Context, p Product) {
	if s.globalHook == nil {
		return
	}
	var pack int64
	if p.UnitsPerPack != nil {
		pack = *p.UnitsPerPack
	}
	if err := s.globalHook.OnProductUpserted(ctx, p.ProductID, p.SupplierID, p.Name, p.Barcode, p.PriceMinor, p.Currency, pack); err != nil {
		if s.log != nil {
			s.log.WarnContext(ctx, "global products hook failed", "product_id", p.ProductID, "err", err)
		}
	}
}

func productHandlingEvent(p Product) events.ProductEvent {
	return events.ProductEvent{
		BaseEvent: events.BaseEvent{
			Type:      events.EventProductHandlingUpdated,
			Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		},
		ProductID:         p.ProductID,
		SupplierID:        p.SupplierID,
		HandlingClass:     string(p.HandlingClass),
		RequiresColdChain: p.RequiresColdChain,
		IsHazardous:       p.IsHazardous,
		IsPerishable:      p.IsPerishable,
		StorageTempMinC:   p.StorageTempMinC,
		StorageTempMaxC:   p.StorageTempMaxC,
	}
}
