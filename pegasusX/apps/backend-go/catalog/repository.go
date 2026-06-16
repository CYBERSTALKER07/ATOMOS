package catalog

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// Product mirrors the Products Spanner row.
type Product struct {
	ProductID     string    `json:"product_id" spanner:"ProductId"`
	SupplierID    string    `json:"supplier_id" spanner:"SupplierId"`
	CategoryID    string    `json:"category_id" spanner:"CategoryId"`
	Name          string    `json:"name" spanner:"Name"`
	Description   string    `json:"description" spanner:"Description"`
	ImageURL      string    `json:"image_url" spanner:"ImageURL"`
	PriceMinor    int64     `json:"price_minor" spanner:"PriceMinor"`
	Currency      string    `json:"currency" spanner:"Currency"`
	StockQuantity int64     `json:"stock_quantity" spanner:"StockQuantity"`
	Unit          string    `json:"unit" spanner:"Unit"`
	UnitVolumeVU  float64   `json:"unit_volume_vu" spanner:"UnitVolumeVU"`
	Barcode       string    `json:"barcode,omitempty" spanner:"Barcode"`
	IsActive      bool      `json:"is_active" spanner:"IsActive"`
	Version       int64     `json:"version" spanner:"Version"`
	CreatedAt     time.Time `json:"created_at" spanner:"CreatedAt"`
	UpdatedAt     time.Time `json:"updated_at" spanner:"UpdatedAt"`
}

// Category mirrors the ProductCategories Spanner row.
type Category struct {
	CategoryID       string    `json:"category_id" spanner:"CategoryId"`
	SupplierID       string    `json:"supplier_id" spanner:"SupplierId"`
	Name             string    `json:"name" spanner:"Name"`
	ParentCategoryID string    `json:"parent_category_id" spanner:"ParentCategoryId"`
	IconKey          string    `json:"icon_key" spanner:"IconKey"`
	SortOrder        int64     `json:"sort_order" spanner:"SortOrder"`
	CreatedAt        time.Time `json:"created_at" spanner:"CreatedAt"`
	UpdatedAt        time.Time `json:"updated_at" spanner:"UpdatedAt"`
}

// CategorySupplier is a supplier that stocks products in a category.
type CategorySupplier struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	ProductCount int64  `json:"product_count"`
	IsActive     bool   `json:"is_active"`
}

// Repository defines the data access contract for catalog operations.
type Repository interface {
	ListCategories(ctx context.Context, supplierID string) ([]Category, error)
	GetCategory(ctx context.Context, categoryID string) (*Category, error)
	CreateCategory(ctx context.Context, cat Category) error
	ListProducts(ctx context.Context, supplierID, categoryID string, activeOnly bool) ([]Product, error)
	ListDiscoverableProducts(ctx context.Context, categoryID string, limit, offset int64) ([]Product, error)
	GetProduct(ctx context.Context, productID string) (*Product, error)
	CreateProduct(ctx context.Context, p Product) error
	UpdateProduct(ctx context.Context, p Product) error
	ListCategorySuppliers(ctx context.Context, categoryID string) ([]CategorySupplier, error)
	SearchSuppliers(ctx context.Context, query string, limit int) ([]CategorySupplier, error)
}

// SpannerRepository implements Repository backed by Cloud Spanner.
type SpannerRepository struct {
	client *spanner.Client
}

// NewSpannerRepository creates a Spanner-backed catalog repository.
func NewSpannerRepository(client *spanner.Client) *SpannerRepository {
	return &SpannerRepository{client: client}
}

// ListCategories returns all categories for a supplier, ordered by SortOrder.
func (r *SpannerRepository) ListCategories(ctx context.Context, supplierID string) ([]Category, error) {
	stmt := spanner.Statement{
		SQL:    "SELECT CategoryId, SupplierId, Name, ParentCategoryId, IconKey, SortOrder, CreatedAt, UpdatedAt FROM ProductCategories WHERE SupplierId = @sid ORDER BY SortOrder",
		Params: map[string]any{"sid": supplierID},
	}
	iter := r.client.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)).Query(ctx, stmt)
	defer iter.Stop()

	var cats []Category
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list categories for supplier %s: %w", supplierID, err)
		}
		var c Category
		var parentID spanner.NullString
		var iconKey spanner.NullString
		if err := row.Columns(&c.CategoryID, &c.SupplierID, &c.Name, &parentID, &iconKey, &c.SortOrder, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan category row: %w", err)
		}
		c.ParentCategoryID = parentID.StringVal
		c.IconKey = iconKey.StringVal
		cats = append(cats, c)
	}
	return cats, nil
}

// GetCategory reads a single category by PK.
func (r *SpannerRepository) GetCategory(ctx context.Context, categoryID string) (*Category, error) {
	row, err := r.client.Single().ReadRow(ctx, "ProductCategories", spanner.Key{categoryID},
		[]string{"CategoryId", "SupplierId", "Name", "ParentCategoryId", "IconKey", "SortOrder", "CreatedAt", "UpdatedAt"})
	if err != nil {
		return nil, fmt.Errorf("get category %s: %w", categoryID, err)
	}
	var c Category
	var parentID, iconKey spanner.NullString
	if err := row.Columns(&c.CategoryID, &c.SupplierID, &c.Name, &parentID, &iconKey, &c.SortOrder, &c.CreatedAt, &c.UpdatedAt); err != nil {
		return nil, fmt.Errorf("scan category %s: %w", categoryID, err)
	}
	c.ParentCategoryID = parentID.StringVal
	c.IconKey = iconKey.StringVal
	return &c, nil
}

// CreateCategory inserts a new category row inside a ReadWriteTransaction.
func (r *SpannerRepository) CreateCategory(ctx context.Context, cat Category) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		m := spanner.InsertOrUpdateMap("ProductCategories", map[string]any{
			"CategoryId":       cat.CategoryID,
			"SupplierId":       cat.SupplierID,
			"Name":             cat.Name,
			"ParentCategoryId": spanner.NullString{StringVal: cat.ParentCategoryID, Valid: cat.ParentCategoryID != ""},
			"IconKey":          spanner.NullString{StringVal: cat.IconKey, Valid: cat.IconKey != ""},
			"SortOrder":        cat.SortOrder,
			"CreatedAt":        spanner.CommitTimestamp,
			"UpdatedAt":        spanner.CommitTimestamp,
		})
		return txn.BufferWrite([]*spanner.Mutation{m})
	})
	if err != nil {
		return fmt.Errorf("create category %s: %w", cat.CategoryID, err)
	}
	return nil
}

// ListProducts returns products for a supplier, optionally filtered by category and active status.
func (r *SpannerRepository) ListProducts(ctx context.Context, supplierID, categoryID string, activeOnly bool) ([]Product, error) {
	sql := "SELECT ProductId, SupplierId, CategoryId, Name, Description, ImageURL, PriceMinor, Currency, StockQuantity, Unit, UnitVolumeVU, IsActive, Version, CreatedAt, UpdatedAt FROM Products WHERE SupplierId = @sid"
	params := map[string]any{"sid": supplierID}
	if categoryID != "" {
		sql += " AND CategoryId = @cid"
		params["cid"] = categoryID
	}
	if activeOnly {
		sql += " AND IsActive = TRUE"
	}
	sql += " ORDER BY UpdatedAt DESC"

	stmt := spanner.Statement{SQL: sql, Params: params}
	iter := r.client.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)).Query(ctx, stmt)
	defer iter.Stop()

	var products []Product
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list products for supplier %s: %w", supplierID, err)
		}
		var p Product
		var desc, imageURL spanner.NullString
		if err := row.Columns(&p.ProductID, &p.SupplierID, &p.CategoryID, &p.Name, &desc, &imageURL,
			&p.PriceMinor, &p.Currency, &p.StockQuantity, &p.Unit, &p.UnitVolumeVU, &p.IsActive, &p.Version,
			&p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan product row: %w", err)
		}
		p.Description = desc.StringVal
		p.ImageURL = imageURL.StringVal
		products = append(products, p)
	}
	return products, nil
}

// ListDiscoverableProducts returns active products for retailer browse when no supplier is selected.
func (r *SpannerRepository) ListDiscoverableProducts(ctx context.Context, categoryID string, limit, offset int64) ([]Product, error) {
	if limit <= 0 {
		limit = 500
	}
	if limit > 5000 {
		limit = 5000
	}
	if offset < 0 {
		offset = 0
	}
	sql := "SELECT ProductId, SupplierId, CategoryId, Name, Description, ImageURL, PriceMinor, Currency, StockQuantity, Unit, UnitVolumeVU, IsActive, Version, CreatedAt, UpdatedAt FROM Products WHERE IsActive = TRUE"
	params := map[string]any{"lim": limit, "off": offset}
	if categoryID != "" {
		sql += " AND CategoryId = @cid"
		params["cid"] = categoryID
	}
	sql += " ORDER BY UpdatedAt DESC LIMIT @lim OFFSET @off"

	stmt := spanner.Statement{SQL: sql, Params: params}
	iter := r.client.Single().WithTimestampBound(spanner.ExactStaleness(15 * time.Second)).Query(ctx, stmt)
	defer iter.Stop()

	var products []Product
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list discoverable products: %w", err)
		}
		var p Product
		var desc, imageURL spanner.NullString
		if err := row.Columns(&p.ProductID, &p.SupplierID, &p.CategoryID, &p.Name, &desc, &imageURL,
			&p.PriceMinor, &p.Currency, &p.StockQuantity, &p.Unit, &p.UnitVolumeVU, &p.IsActive, &p.Version,
			&p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan discoverable product row: %w", err)
		}
		p.Description = desc.StringVal
		p.ImageURL = imageURL.StringVal
		products = append(products, p)
	}
	return products, nil
}

// GetProduct reads a single product by PK.
func (r *SpannerRepository) GetProduct(ctx context.Context, productID string) (*Product, error) {
	row, err := r.client.Single().ReadRow(ctx, "Products", spanner.Key{productID},
		[]string{"ProductId", "SupplierId", "CategoryId", "Name", "Description", "ImageURL",
			"PriceMinor", "Currency", "StockQuantity", "Unit", "UnitVolumeVU", "IsActive", "Version", "CreatedAt", "UpdatedAt"})
	if err != nil {
		return nil, fmt.Errorf("get product %s: %w", productID, err)
	}
	var p Product
	var desc, imageURL spanner.NullString
	if err := row.Columns(&p.ProductID, &p.SupplierID, &p.CategoryID, &p.Name, &desc, &imageURL,
		&p.PriceMinor, &p.Currency, &p.StockQuantity, &p.Unit, &p.UnitVolumeVU, &p.IsActive, &p.Version,
		&p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, fmt.Errorf("scan product %s: %w", productID, err)
	}
	p.Description = desc.StringVal
	p.ImageURL = imageURL.StringVal
	return &p, nil
}

// CreateProduct inserts a new product row inside a ReadWriteTransaction.
func (r *SpannerRepository) CreateProduct(ctx context.Context, p Product) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		m := spanner.InsertOrUpdateMap("Products", map[string]any{
			"ProductId":     p.ProductID,
			"SupplierId":    p.SupplierID,
			"CategoryId":    p.CategoryID,
			"Name":          p.Name,
			"Description":   spanner.NullString{StringVal: p.Description, Valid: p.Description != ""},
			"ImageURL":      spanner.NullString{StringVal: p.ImageURL, Valid: p.ImageURL != ""},
			"PriceMinor":    p.PriceMinor,
			"Currency":      p.Currency,
			"StockQuantity": p.StockQuantity,
			"Unit":          p.Unit,
			"UnitVolumeVU":  normalizeUnitVolumeVU(p.UnitVolumeVU),
			"Barcode":       spanner.NullString{StringVal: p.Barcode, Valid: strings.TrimSpace(p.Barcode) != ""},
			"IsActive":      p.IsActive,
			"Version":       int64(1),
			"CreatedAt":     spanner.CommitTimestamp,
			"UpdatedAt":     spanner.CommitTimestamp,
		})
		return txn.BufferWrite([]*spanner.Mutation{m})
	})
	if err != nil {
		return fmt.Errorf("create product %s: %w", p.ProductID, err)
	}
	return nil
}

// UpdateProduct updates a product row with optimistic concurrency.
func (r *SpannerRepository) UpdateProduct(ctx context.Context, p Product) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, readErr := txn.ReadRow(ctx, "Products", spanner.Key{p.ProductID}, []string{"Version"})
		if readErr != nil {
			return fmt.Errorf("read product version %s: %w", p.ProductID, readErr)
		}
		var currentVersion int64
		if err := row.Columns(&currentVersion); err != nil {
			return fmt.Errorf("scan product version %s: %w", p.ProductID, err)
		}
		if currentVersion != p.Version {
			return fmt.Errorf("product %s version conflict: expected %d, got %d", p.ProductID, p.Version, currentVersion)
		}
		m := spanner.UpdateMap("Products", map[string]any{
			"ProductId":     p.ProductID,
			"Name":          p.Name,
			"Description":   spanner.NullString{StringVal: p.Description, Valid: p.Description != ""},
			"ImageURL":      spanner.NullString{StringVal: p.ImageURL, Valid: p.ImageURL != ""},
			"PriceMinor":    p.PriceMinor,
			"Currency":      p.Currency,
			"StockQuantity": p.StockQuantity,
			"Unit":          p.Unit,
			"UnitVolumeVU":  normalizeUnitVolumeVU(p.UnitVolumeVU),
			"Barcode":       spanner.NullString{StringVal: p.Barcode, Valid: strings.TrimSpace(p.Barcode) != ""},
			"IsActive":      p.IsActive,
			"Version":       currentVersion + 1,
			"UpdatedAt":     spanner.CommitTimestamp,
		})
		return txn.BufferWrite([]*spanner.Mutation{m})
	})
	if err != nil {
		return fmt.Errorf("update product %s: %w", p.ProductID, err)
	}
	return nil
}

// ListCategorySuppliers returns suppliers with active products in the category.
func (r *SpannerRepository) ListCategorySuppliers(ctx context.Context, categoryID string) ([]CategorySupplier, error) {
	stmt := spanner.Statement{
		SQL: `SELECT s.SupplierId, s.Name, COUNT(p.ProductId) AS ProductCount
		      FROM Suppliers s
		      INNER JOIN Products p ON s.SupplierId = p.SupplierId
		      WHERE p.CategoryId = @cid AND p.IsActive = TRUE
		      GROUP BY s.SupplierId, s.Name
		      ORDER BY s.Name ASC`,
		Params: map[string]any{"cid": categoryID},
	}
	iter := r.client.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)).Query(ctx, stmt)
	defer iter.Stop()

	var out []CategorySupplier
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list category suppliers %s: %w", categoryID, err)
		}
		var s CategorySupplier
		if err := row.Columns(&s.ID, &s.Name, &s.ProductCount); err != nil {
			return nil, fmt.Errorf("scan category supplier: %w", err)
		}
		s.IsActive = true
		out = append(out, s)
	}
	return out, nil
}

// SearchSuppliers finds suppliers by name prefix for retailer discovery.
func (r *SpannerRepository) SearchSuppliers(ctx context.Context, query string, limit int) ([]CategorySupplier, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	pattern := strings.TrimSpace(query) + "%"
	stmt := spanner.Statement{
		SQL: `SELECT SupplierId, Name, 0 AS ProductCount
		      FROM Suppliers
		      WHERE LOWER(Name) LIKE LOWER(@pattern)
		      ORDER BY Name ASC
		      LIMIT @limit`,
		Params: map[string]any{"pattern": pattern, "limit": int64(limit)},
	}
	iter := r.client.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)).Query(ctx, stmt)
	defer iter.Stop()

	var out []CategorySupplier
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("search suppliers %q: %w", query, err)
		}
		var s CategorySupplier
		if err := row.Columns(&s.ID, &s.Name, &s.ProductCount); err != nil {
			return nil, err
		}
		s.IsActive = true
		out = append(out, s)
	}
	return out, nil
}

func normalizeUnitVolumeVU(v float64) float64 {
	if v <= 0 {
		return 1.0
	}
	return v
}
