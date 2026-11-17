package partner

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/catalog"
	"github.com/pegasusx/pegasusx/apps/backend-go/inventory"
)

const maxMasterDataBatch = 500

// ProductUpsertItem is one product row in a partner catalog sync batch.
type ProductUpsertItem struct {
	ExternalID    string `json:"external_id"`
	Name          string `json:"name"`
	CategoryID    string `json:"category_id"`
	PriceMinor    int64  `json:"price_minor"`
	Currency      string `json:"currency"`
	Unit          string `json:"unit"`
	Barcode       string `json:"barcode"`
	Description   string `json:"description"`
	ImageURL      string `json:"image_url"`
	IsActive      *bool  `json:"is_active"`
	HandlingClass string `json:"handling_class"`
}

// ProductUpsertResult is the per-row outcome of a product upsert.
type ProductUpsertResult struct {
	ExternalID string `json:"external_id"`
	ProductID  string `json:"product_id,omitempty"`
	Action     string `json:"action"` // created|updated|error
	Error      string `json:"error,omitempty"`
}

// PriceUpsertItem updates list or per-retailer price by external product id.
type PriceUpsertItem struct {
	ExternalID string  `json:"external_id"`
	PriceMinor int64   `json:"price_minor"`
	Currency   string  `json:"currency"`
	RetailerID *string `json:"retailer_id"`
}

// PriceUpsertResult is the per-row outcome of a price upsert.
type PriceUpsertResult struct {
	ExternalID string `json:"external_id"`
	Action     string `json:"action"`
	Error      string `json:"error,omitempty"`
}

// StockUpsertItem sets absolute on-hand quantity for a warehouse×product.
type StockUpsertItem struct {
	ExternalID       string `json:"external_id"`
	WarehouseID      string `json:"warehouse_id"`
	QuantityOnHand   int64  `json:"quantity_on_hand"`
	ReorderThreshold *int64 `json:"reorder_threshold"`
}

// StockUpsertResult is the per-row outcome of a stock upsert.
type StockUpsertResult struct {
	ExternalID string `json:"external_id"`
	Action     string `json:"action"`
	Error      string `json:"error,omitempty"`
}

func (s *Service) requireSupplierPrincipal(p Principal) error {
	if strings.ToUpper(strings.TrimSpace(p.TenantType)) != TenantSupplier {
		return fmt.Errorf("supplier_tenant_required")
	}
	if strings.TrimSpace(p.TenantID) == "" {
		return fmt.Errorf("tenant_id_required")
	}
	return nil
}

// ValidateAndBuildProduct normalizes and constructs a catalog.Product mutation (create or update)
// shared across partner ERP APIs and migration catalog appliers.
func ValidateAndBuildProduct(
	supplierID string,
	it ProductUpsertItem,
	existing *catalog.Product,
	defaultCurrency string,
	now time.Time,
) (catalog.Product, bool, string) {
	ext := strings.TrimSpace(it.ExternalID)
	if ext == "" {
		return catalog.Product{}, false, "external_id_required"
	}
	if len(ext) > 36 {
		return catalog.Product{}, false, "external_id_too_long"
	}
	name := strings.TrimSpace(it.Name)
	if name == "" {
		return catalog.Product{}, false, "name_required"
	}
	currency := strings.TrimSpace(it.Currency)
	if currency == "" {
		currency = strings.TrimSpace(defaultCurrency)
	}
	unit := strings.TrimSpace(it.Unit)
	if unit == "" {
		unit = "UNIT"
	}
	active := true
	if it.IsActive != nil {
		active = *it.IsActive
	}
	if existing != nil {
		if existing.SupplierID != supplierID {
			return catalog.Product{}, true, "product_owned_by_other_supplier"
		}
		updated := *existing
		updated.Name = name
		if cat := strings.TrimSpace(it.CategoryID); cat != "" {
			updated.CategoryID = cat
		}
		updated.PriceMinor = it.PriceMinor
		if currency != "" {
			updated.Currency = currency
		}
		updated.Unit = unit
		updated.Barcode = strings.TrimSpace(it.Barcode)
		updated.Description = strings.TrimSpace(it.Description)
		updated.ImageURL = strings.TrimSpace(it.ImageURL)
		updated.IsActive = active
		if hc := strings.TrimSpace(it.HandlingClass); hc != "" {
			updated.HandlingClass = catalog.HandlingClass(hc)
		}
		return updated, true, ""
	}
	prod := catalog.Product{
		ProductID:     ext,
		SupplierID:    supplierID,
		CategoryID:    strings.TrimSpace(it.CategoryID),
		Name:          name,
		Description:   strings.TrimSpace(it.Description),
		ImageURL:      strings.TrimSpace(it.ImageURL),
		PriceMinor:    it.PriceMinor,
		Currency:      currency,
		Unit:          unit,
		Barcode:       strings.TrimSpace(it.Barcode),
		HandlingClass: catalog.HandlingClass(strings.TrimSpace(it.HandlingClass)),
		IsActive:      active,
		Version:       1,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if prod.HandlingClass == "" {
		prod.HandlingClass = catalog.HandlingClassGeneral
	}
	return prod, false, ""
}

// ValidateAndApplyPriceUpdate validates and applies a list price update onto an existing catalog.Product.
func ValidateAndApplyPriceUpdate(supplierID string, it PriceUpsertItem, existing *catalog.Product) (catalog.Product, string) {
	if existing == nil {
		return catalog.Product{}, "product_not_found"
	}
	if existing.SupplierID != supplierID {
		return catalog.Product{}, "product_owned_by_other_supplier"
	}
	if it.PriceMinor < 0 {
		return catalog.Product{}, "price_minor_invalid"
	}
	updated := *existing
	updated.PriceMinor = it.PriceMinor
	if c := strings.TrimSpace(it.Currency); c != "" {
		updated.Currency = c
	}
	return updated, ""
}

// ValidateAndBuildStockLevel constructs an updated or newly created inventory.Level preserving reserved quantities.
func ValidateAndBuildStockLevel(supplierID string, it StockUpsertItem, existing *inventory.Level, now time.Time) (inventory.Level, bool, string) {
	ext := strings.TrimSpace(it.ExternalID)
	wh := strings.TrimSpace(it.WarehouseID)
	if ext == "" || wh == "" {
		return inventory.Level{}, false, "external_id_and_warehouse_id_required"
	}
	if it.QuantityOnHand < 0 {
		return inventory.Level{}, false, "quantity_on_hand_invalid"
	}
	level := inventory.Level{
		InventoryID:      uuid.NewString(),
		WarehouseID:      wh,
		ProductID:        ext,
		SupplierID:       supplierID,
		QuantityOnHand:   it.QuantityOnHand,
		QuantityReserved: 0,
		ReorderThreshold: 0,
		Version:          1,
		UpdatedAt:        now,
	}
	if it.ReorderThreshold != nil {
		level.ReorderThreshold = *it.ReorderThreshold
	}
	if existing != nil {
		level.InventoryID = existing.InventoryID
		level.QuantityReserved = existing.QuantityReserved
		level.Version = existing.Version
		if it.ReorderThreshold == nil {
			level.ReorderThreshold = existing.ReorderThreshold
		}
		return level, true, ""
	}
	return level, false, ""
}

// UpsertProducts creates or updates products. ProductId == external_id (import-wizard convention).
func (s *Service) UpsertProducts(ctx context.Context, p Principal, items []ProductUpsertItem) ([]ProductUpsertResult, error) {
	if err := s.requireSupplierPrincipal(p); err != nil {
		return nil, err
	}
	if s.catalog == nil {
		return nil, fmt.Errorf("catalog_unavailable")
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("items_required")
	}
	if len(items) > maxMasterDataBatch {
		return nil, fmt.Errorf("batch_too_large")
	}
	defaultCurrency := currencyFromTenantOrEmpty(p.TenantType, p.TenantID)
	out := make([]ProductUpsertResult, 0, len(items))
	for _, it := range items {
		ext := strings.TrimSpace(it.ExternalID)
		res := ProductUpsertResult{ExternalID: ext}
		var existing *catalog.Product
		if ext != "" && len(ext) <= 36 {
			existing, _ = s.catalog.GetProduct(ctx, ext)
		}
		prod, isUpdate, errCode := ValidateAndBuildProduct(p.TenantID, it, existing, defaultCurrency, s.now())
		if errCode != "" {
			res.Action = "error"
			res.Error = errCode
			out = append(out, res)
			continue
		}
		if isUpdate {
			if err := s.catalog.UpdateProduct(ctx, prod); err != nil {
				res.Action = "error"
				res.Error = err.Error()
				out = append(out, res)
				continue
			}
			res.ProductID = prod.ProductID
			res.Action = "updated"
			out = append(out, res)
			continue
		}
		if err := s.catalog.CreateProduct(ctx, prod); err != nil {
			res.Action = "error"
			res.Error = err.Error()
			out = append(out, res)
			continue
		}
		res.ProductID = prod.ProductID
		res.Action = "created"
		out = append(out, res)
	}
	return out, nil
}

// UpsertPrices updates list prices (retailer_id null) on Products.
// Per-retailer overrides require the supplier pricing portal path in this phase
// (RetailerPricingOverrides) — list-price only here to keep the surface tight.
func (s *Service) UpsertPrices(ctx context.Context, p Principal, items []PriceUpsertItem) ([]PriceUpsertResult, error) {
	if err := s.requireSupplierPrincipal(p); err != nil {
		return nil, err
	}
	if s.catalog == nil {
		return nil, fmt.Errorf("catalog_unavailable")
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("items_required")
	}
	if len(items) > maxMasterDataBatch {
		return nil, fmt.Errorf("batch_too_large")
	}
	out := make([]PriceUpsertResult, 0, len(items))
	for _, it := range items {
		ext := strings.TrimSpace(it.ExternalID)
		res := PriceUpsertResult{ExternalID: ext}
		if ext == "" {
			res.Action = "error"
			res.Error = "external_id_required"
			out = append(out, res)
			continue
		}
		if it.RetailerID != nil && strings.TrimSpace(*it.RetailerID) != "" {
			res.Action = "error"
			res.Error = "retailer_override_use_supplier_portal"
			out = append(out, res)
			continue
		}
		existing, err := s.catalog.GetProduct(ctx, ext)
		if err != nil {
			existing = nil
		}
		updated, errCode := ValidateAndApplyPriceUpdate(p.TenantID, it, existing)
		if errCode != "" {
			res.Action = "error"
			res.Error = errCode
			out = append(out, res)
			continue
		}
		if err := s.catalog.UpdateProduct(ctx, updated); err != nil {
			res.Action = "error"
			res.Error = err.Error()
			out = append(out, res)
			continue
		}
		res.Action = "updated"
		out = append(out, res)
	}
	return out, nil
}

// UpsertStock sets absolute InventoryLevels quantity (fails closed when inventory svc missing).
func (s *Service) UpsertStock(ctx context.Context, p Principal, items []StockUpsertItem) ([]StockUpsertResult, error) {
	if err := s.requireSupplierPrincipal(p); err != nil {
		return nil, err
	}
	if s.inventory == nil {
		return nil, fmt.Errorf("inventory_unavailable")
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("items_required")
	}
	if len(items) > maxMasterDataBatch {
		return nil, fmt.Errorf("batch_too_large")
	}
	out := make([]StockUpsertResult, 0, len(items))
	for _, it := range items {
		ext := strings.TrimSpace(it.ExternalID)
		wh := strings.TrimSpace(it.WarehouseID)
		res := StockUpsertResult{ExternalID: ext}
		if ext != "" && wh != "" && s.catalog != nil {
			if prod, err := s.catalog.GetProduct(ctx, ext); err == nil && prod != nil && prod.SupplierID != p.TenantID {
				res.Action = "error"
				res.Error = "product_owned_by_other_supplier"
				out = append(out, res)
				continue
			}
		}
		var existing *inventory.Level
		if ext != "" && wh != "" {
			existing, _ = s.inventory.FindByWarehouseProduct(ctx, wh, ext)
		}
		level, isUpdate, errCode := ValidateAndBuildStockLevel(p.TenantID, it, existing, time.Now().UTC())
		if errCode != "" {
			res.Action = "error"
			res.Error = errCode
			out = append(out, res)
			continue
		}
		if err := s.inventory.Upsert(ctx, level); err != nil {
			res.Action = "error"
			res.Error = err.Error()
			out = append(out, res)
			continue
		}
		if isUpdate {
			res.Action = "updated"
		} else {
			res.Action = "created"
		}
		out = append(out, res)
	}
	return out, nil
}
