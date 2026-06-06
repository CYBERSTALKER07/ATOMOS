package catalog

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
)

// Service implements catalog business logic. Handlers call service; service
// calls repository.
type Service struct {
	repo  Repository
	cache *cache.Cache
	log   *slog.Logger
}

// NewService creates a catalog service with the given dependencies.
func NewService(repo Repository, c *cache.Cache, log *slog.Logger) *Service {
	return &Service{repo: repo, cache: c, log: log}
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
