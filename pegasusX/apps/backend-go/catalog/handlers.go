package catalog

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/pkg/httppagination"
	"github.com/pegasusx/pegasusx/apps/backend-go/storage"
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
	supplierID := strings.TrimSpace(r.URL.Query().Get("supplier_id"))
	if supplierID == "" {
		if sid, ok := auth.ResolveSupplierID(r.Context()); ok {
			supplierID = sid
		}
	}
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
	supplierID := strings.TrimSpace(r.URL.Query().Get("supplier_id"))
	if supplierID == "" {
		if sid, ok := auth.ResolveSupplierID(r.Context()); ok {
			supplierID = sid
		}
	}
	categoryID := strings.TrimSpace(r.URL.Query().Get("category_id"))
	retailerID := strings.TrimSpace(r.URL.Query().Get("retailer_id"))
	if retailerID == "" {
		if claims, ok := auth.FromContext(r.Context()); ok && claims.Role == auth.RoleRetailer {
			retailerID = auth.ResolveRetailerOrgID(claims)
		}
	}
	if supplierID == "" {
		if retailerID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "supplier_id required"})
			return
		}
		limit, offset := httppagination.ParseLimitOffset(r, 500, 5000)
		marketCode := ""
		if claims, ok := auth.FromContext(r.Context()); ok {
			marketCode = claims.MarketCode
		}
		if marketCode == "" {
			marketCode = auth.DefaultMarketCodeFromEnv()
		}
		products, err := s.ListProductsDiscovery(r.Context(), retailerID, categoryID, marketCode, int64(limit), int64(offset))
		if err != nil {
			slog.ErrorContext(r.Context(), "list discovery products failed", "err", err, "retailer_id", retailerID)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
			return
		}
		w.Header().Set("X-Page-Limit", strconv.Itoa(limit))
		w.Header().Set("X-Page-Offset", strconv.Itoa(offset))
		w.Header().Set("X-Page-Has-More", strconv.FormatBool(len(products) == limit))
		writeJSON(w, http.StatusOK, products)
		return
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
		SupplierID        string   `json:"supplier_id"`
		CategoryID        string   `json:"category_id"`
		Name              string   `json:"name"`
		Description       string   `json:"description"`
		ImageURL          string   `json:"image_url"`
		PriceMinor        int64    `json:"price_minor"`
		Currency          string   `json:"currency"`
		StockQuantity     int64    `json:"stock_quantity"`
		Unit              string   `json:"unit"`
		UnitVolumeVU      float64  `json:"unit_volume_vu"`
		SaleUnit          string   `json:"sale_unit"`
		UnitsPerPack      *int64   `json:"units_per_pack"`
		UnitsPerCase      *int64   `json:"units_per_case"`
		Barcode           string   `json:"barcode"`
		HandlingClass     string   `json:"handling_class"`
		RequiresColdChain bool     `json:"requires_cold_chain"`
		IsHazardous       bool     `json:"is_hazardous"`
		IsPerishable      bool     `json:"is_perishable"`
		StorageTempMinC   *float64 `json:"storage_temp_min_c"`
		StorageTempMaxC   *float64 `json:"storage_temp_max_c"`
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
	handlingClass := HandlingClass(strings.ToUpper(strings.TrimSpace(req.HandlingClass)))
	if !handlingClass.Valid() {
		handlingClass = HandlingClassGeneral
	}
	if err := validateHandlingTemperatures(handlingClass, req.StorageTempMinC, req.StorageTempMaxC); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	p := Product{
		ProductID:         uuid.NewString(),
		SupplierID:        supplierID,
		CategoryID:        req.CategoryID,
		Name:              req.Name,
		Description:       req.Description,
		ImageURL:          req.ImageURL,
		PriceMinor:        req.PriceMinor,
		Currency:          req.Currency,
		StockQuantity:     req.StockQuantity,
		Unit:              unit,
		UnitVolumeVU:      req.UnitVolumeVU,
		SaleUnit:          req.SaleUnit,
		UnitsPerPack:      coalesceUnitsPerPack(req.UnitsPerPack, req.UnitsPerCase),
		Barcode:           strings.TrimSpace(req.Barcode),
		HandlingClass:     handlingClass,
		RequiresColdChain: req.RequiresColdChain || handlingClass == HandlingClassColdChain || handlingClass == HandlingClassPerishable,
		IsHazardous:       req.IsHazardous || handlingClass == HandlingClassHazardous,
		IsPerishable:      req.IsPerishable || handlingClass == HandlingClassPerishable,
		StorageTempMinC:   req.StorageTempMinC,
		StorageTempMaxC:   req.StorageTempMaxC,
		IsActive:          true,
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
		Name              string   `json:"name"`
		Description       string   `json:"description"`
		ImageURL          string   `json:"image_url"`
		PriceMinor        int64    `json:"price_minor"`
		Currency          string   `json:"currency"`
		StockQuantity     int64    `json:"stock_quantity"`
		Unit              string   `json:"unit"`
		UnitVolumeVU      *float64 `json:"unit_volume_vu"`
		SaleUnit          *string  `json:"sale_unit"`
		UnitsPerPack      *int64   `json:"units_per_pack"`
		UnitsPerCase      *int64   `json:"units_per_case"`
		Barcode           *string  `json:"barcode"`
		IsActive          *bool    `json:"is_active"`
		HandlingClass     *string  `json:"handling_class"`
		RequiresColdChain *bool    `json:"requires_cold_chain"`
		IsHazardous       *bool    `json:"is_hazardous"`
		IsPerishable      *bool    `json:"is_perishable"`
		StorageTempMinC   *float64 `json:"storage_temp_min_c"`
		StorageTempMaxC   *float64 `json:"storage_temp_max_c"`
		Version           int64    `json:"version"`
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
	if req.UnitVolumeVU != nil {
		existing.UnitVolumeVU = *req.UnitVolumeVU
	}
	if req.SaleUnit != nil {
		existing.SaleUnit = *req.SaleUnit
	}
	if req.UnitsPerPack != nil || req.UnitsPerCase != nil {
		existing.UnitsPerPack = coalesceUnitsPerPack(req.UnitsPerPack, req.UnitsPerCase)
	}
	if req.Barcode != nil {
		existing.Barcode = strings.TrimSpace(*req.Barcode)
	}
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}
	if req.HandlingClass != nil {
		handlingClass := HandlingClass(strings.ToUpper(strings.TrimSpace(*req.HandlingClass)))
		if !handlingClass.Valid() {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_handling_class"})
			return
		}
		if err := validateHandlingTemperatures(handlingClass, req.StorageTempMinC, req.StorageTempMaxC); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		existing.HandlingClass = handlingClass
		existing.RequiresColdChain = handlingClass == HandlingClassColdChain || handlingClass == HandlingClassPerishable
		existing.IsHazardous = handlingClass == HandlingClassHazardous
		existing.IsPerishable = handlingClass == HandlingClassPerishable
	}
	if req.RequiresColdChain != nil {
		existing.RequiresColdChain = *req.RequiresColdChain
	}
	if req.IsHazardous != nil {
		existing.IsHazardous = *req.IsHazardous
	}
	if req.IsPerishable != nil {
		existing.IsPerishable = *req.IsPerishable
	}
	if req.StorageTempMinC != nil {
		existing.StorageTempMinC = req.StorageTempMinC
	}
	if req.StorageTempMaxC != nil {
		existing.StorageTempMaxC = req.StorageTempMaxC
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

// HandleGetUploadTicket serves GET /v1/catalog/products/upload-ticket?ext=jpg.
// Returns a signed GCS PUT URL and the eventual public image_url for catalog create/update.
func (s *Service) HandleGetUploadTicket(w http.ResponseWriter, r *http.Request) {
	supplierID, ok := auth.ResolveSupplierID(r.Context())
	if !ok || supplierID == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "supplier_scope_required"})
		return
	}
	extension := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("ext")))
	if extension == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "ext required"})
		return
	}
	allowed := map[string]bool{"jpg": true, "jpeg": true, "png": true, "webp": true}
	if !allowed[extension] {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported extension"})
		return
	}
	uploadURL, imageURL, err := storage.GenerateUploadTicket(supplierID, extension)
	if err != nil {
		slog.ErrorContext(r.Context(), "upload ticket failed", "err", err, "supplier_id", supplierID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"upload_url": uploadURL,
		"image_url":  imageURL,
	})
}

func validateHandlingTemperatures(hc HandlingClass, minC, maxC *float64) error {
	if hc == HandlingClassColdChain || hc == HandlingClassPerishable {
		if minC == nil || maxC == nil {
			return errors.New("temperature_range_required")
		}
		if *maxC <= *minC {
			return errors.New("invalid_temperature_range")
		}
	}
	if hc == HandlingClassGeneral && (minC != nil || maxC != nil) {
		return errors.New("temperature_not_allowed_for_general")
	}
	return nil
}

func coalesceUnitsPerPack(primary, legacy *int64) *int64 {
	if primary != nil {
		return primary
	}
	return legacy
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func extractPathParam(r *http.Request, name string) string {
	return r.PathValue(name)
}
