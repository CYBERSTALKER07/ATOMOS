package catalog

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// scopedSupplierID resolves the caller's supplier scope from JWT claims and
// rejects any body-supplied supplier_id that disagrees. Returns ("", false)
// when the response has already been written.
func scopedSupplierID(w http.ResponseWriter, r *http.Request, bodySupplierID string) (string, bool) {
	supplierID, ok := auth.ResolveSupplierID(r.Context())
	if !ok || supplierID == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "supplier_scope_required"})
		return "", false
	}
	if body := strings.TrimSpace(bodySupplierID); body != "" && body != supplierID {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "supplier_scope_violation"})
		return "", false
	}
	return supplierID, true
}

// HandleListCategories serves GET /v1/catalog/categories.
func (s *Service) HandleListCategories(w http.ResponseWriter, r *http.Request) {
	supplierID := r.URL.Query().Get("supplier_id")
	if supplierID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "supplier_id required"})
		return
	}
	cats, err := s.ListCategories(r.Context(), supplierID)
	if err != nil {
		slog.ErrorContext(r.Context(), "list categories failed", "err", err, "supplier_id", supplierID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusOK, cats)
}

// HandleListProducts serves GET /v1/catalog/products.
func (s *Service) HandleListProducts(w http.ResponseWriter, r *http.Request) {
	supplierID := r.URL.Query().Get("supplier_id")
	if supplierID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "supplier_id required"})
		return
	}
	categoryID := r.URL.Query().Get("category_id")
	retailerID := strings.TrimSpace(r.URL.Query().Get("retailer_id"))
	if retailerID == "" {
		if claims, ok := auth.FromContext(r.Context()); ok {
			retailerID = strings.TrimSpace(claims.Subject)
		}
	}
	if retailerID != "" {
		products, err := s.ListProductsForRetailer(r.Context(), supplierID, retailerID, categoryID)
		if err != nil {
			slog.ErrorContext(r.Context(), "list retailer products failed", "err", err, "supplier_id", supplierID)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
			return
		}
		writeJSON(w, http.StatusOK, products)
		return
	}
	products, err := s.ListProducts(r.Context(), supplierID, categoryID, true)
	if err != nil {
		slog.ErrorContext(r.Context(), "list products failed", "err", err, "supplier_id", supplierID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusOK, products)
}

// HandleGetProduct serves GET /v1/catalog/products/{productID}.
func (s *Service) HandleGetProduct(w http.ResponseWriter, r *http.Request) {
	productID := extractPathParam(r, "productID")
	if productID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "product_id required"})
		return
	}
	p, err := s.GetProduct(r.Context(), productID)
	if err != nil {
		slog.ErrorContext(r.Context(), "get product failed", "err", err, "product_id", productID)
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// HandleCreateProduct serves POST /v1/catalog/products.
func (s *Service) HandleCreateProduct(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SupplierID    string `json:"supplier_id"`
		CategoryID    string `json:"category_id"`
		Name          string `json:"name"`
		Description   string `json:"description"`
		ImageURL      string `json:"image_url"`
		PriceMinor    int64  `json:"price_minor"`
		Currency      string `json:"currency"`
		StockQuantity int64  `json:"stock_quantity"`
		Unit          string `json:"unit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	supplierID, ok := scopedSupplierID(w, r, req.SupplierID)
	if !ok {
		return
	}
	if req.CategoryID == "" || req.Name == "" || req.Currency == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing required fields"})
		return
	}
	unit := strings.TrimSpace(req.Unit)
	if unit == "" {
		unit = "UNIT"
	}
	p := Product{
		ProductID:     uuid.NewString(),
		SupplierID:    supplierID,
		CategoryID:    req.CategoryID,
		Name:          req.Name,
		Description:   req.Description,
		ImageURL:      req.ImageURL,
		PriceMinor:    req.PriceMinor,
		Currency:      req.Currency,
		StockQuantity: req.StockQuantity,
		Unit:          unit,
		IsActive:      true,
	}
	if err := s.CreateProduct(r.Context(), p); err != nil {
		slog.ErrorContext(r.Context(), "create product failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

// HandleUpdateProduct serves PUT /v1/catalog/products/{productID}.
func (s *Service) HandleUpdateProduct(w http.ResponseWriter, r *http.Request) {
	productID := extractPathParam(r, "productID")
	if productID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "product_id required"})
		return
	}
	var req struct {
		Name          string `json:"name"`
		Description   string `json:"description"`
		ImageURL      string `json:"image_url"`
		PriceMinor    int64  `json:"price_minor"`
		Currency      string `json:"currency"`
		StockQuantity int64  `json:"stock_quantity"`
		Unit          string `json:"unit"`
		IsActive      *bool  `json:"is_active"`
		Version       int64  `json:"version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	existing, err := s.GetProduct(r.Context(), productID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
		return
	}
	if _, ok := scopedSupplierID(w, r, existing.SupplierID); !ok {
		return
	}
	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Description != "" {
		existing.Description = req.Description
	}
	if req.ImageURL != "" {
		existing.ImageURL = req.ImageURL
	}
	if req.PriceMinor > 0 {
		existing.PriceMinor = req.PriceMinor
	}
	if req.Currency != "" {
		existing.Currency = req.Currency
	}
	if req.StockQuantity > 0 {
		existing.StockQuantity = req.StockQuantity
	}
	if req.Unit != "" {
		existing.Unit = req.Unit
	}
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}
	existing.Version = req.Version
	if err := s.UpdateProduct(r.Context(), *existing); err != nil {
		slog.ErrorContext(r.Context(), "update product failed", "err", err, "product_id", productID)
		writeJSON(w, http.StatusConflict, map[string]string{"error": "version_conflict"})
		return
	}
	writeJSON(w, http.StatusOK, existing)
}

// HandleSearchSuppliers serves GET /v1/catalog/suppliers/search.
func (s *Service) HandleSearchSuppliers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	suppliers, err := s.SearchSuppliers(r.Context(), q)
	if err != nil {
		slog.ErrorContext(r.Context(), "search suppliers failed", "err", err, "q", q)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusOK, suppliers)
}

// HandleListProductsAlias serves GET /v1/products (legacy retailer iOS path).
func (s *Service) HandleListProductsAlias(w http.ResponseWriter, r *http.Request) {
	s.HandleListProducts(w, r)
}

// HandleListCategorySuppliers serves GET /v1/catalog/categories/{categoryID}/suppliers.
func (s *Service) HandleListCategorySuppliers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	categoryID := strings.TrimSpace(extractPathParam(r, "categoryID"))
	if categoryID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "category_id_required"})
		return
	}
	suppliers, err := s.ListCategorySuppliers(r.Context(), categoryID)
	if err != nil {
		slog.ErrorContext(r.Context(), "list category suppliers failed", "err", err, "category_id", categoryID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusOK, suppliers)
}

// HandleCreateCategory serves POST /v1/catalog/categories.
func (s *Service) HandleCreateCategory(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SupplierID       string `json:"supplier_id"`
		Name             string `json:"name"`
		ParentCategoryID string `json:"parent_category_id"`
		IconKey          string `json:"icon_key"`
		SortOrder        int64  `json:"sort_order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	supplierID, ok := scopedSupplierID(w, r, req.SupplierID)
	if !ok {
		return
	}
	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name required"})
		return
	}
	cat := Category{
		CategoryID:       uuid.NewString(),
		SupplierID:       supplierID,
		Name:             req.Name,
		ParentCategoryID: req.ParentCategoryID,
		IconKey:          req.IconKey,
		SortOrder:        req.SortOrder,
	}
	if err := s.CreateCategory(r.Context(), cat); err != nil {
		slog.ErrorContext(r.Context(), "create category failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusCreated, cat)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func extractPathParam(r *http.Request, name string) string {
	return r.PathValue(name)
}
