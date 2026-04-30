package models

import (
	"time"
)

// ==========================================
// 1. SPANNER DATABASE MODELS
// ==========================================

// RetailerGlobalSettings dictates the master switch for the Empathy Engine.
type RetailerGlobalSettings struct {
	RetailerID             string    `spanner:"RetailerId"`
	GlobalAutoOrderEnabled bool      `spanner:"GlobalAutoOrderEnabled"`
	AnalyticsStartDate     time.Time `spanner:"AnalyticsStartDate"` // Zero = use all history
	UpdatedAt              time.Time `spanner:"UpdatedAt"`
}

// RetailerSupplierSettings dictates overrides for specific suppliers.
type RetailerSupplierSettings struct {
	RetailerID         string    `spanner:"RetailerId"`
	SupplierID         string    `spanner:"SupplierId"`
	AutoOrderEnabled   bool      `spanner:"AutoOrderEnabled"`
	AnalyticsStartDate time.Time `spanner:"AnalyticsStartDate"` // Zero = inherit from global
	CreditEnabled      bool      `spanner:"CreditEnabled"`
	CreditLimit     int64     `spanner:"CreditLimit"`   // 0 = unlimited
	CreditBalance   int64     `spanner:"CreditBalance"` // Running credit balance
	UpdatedAt          time.Time `spanner:"UpdatedAt"`
}

// RetailerCategorySettings dictates overrides for specific product categories.
type RetailerCategorySettings struct {
	RetailerID         string    `spanner:"RetailerId"`
	CategoryID         string    `spanner:"CategoryId"`
	AutoOrderEnabled   bool      `spanner:"AutoOrderEnabled"`
	AnalyticsStartDate time.Time `spanner:"AnalyticsStartDate"` // Zero = inherit
	UpdatedAt          time.Time `spanner:"UpdatedAt"`
}

// RetailerProductSettings dictates granular overrides for specific products.
type RetailerProductSettings struct {
	RetailerID         string    `spanner:"RetailerId"`
	ProductID          string    `spanner:"ProductId"`
	AutoOrderEnabled   bool      `spanner:"AutoOrderEnabled"`
	AnalyticsStartDate time.Time `spanner:"AnalyticsStartDate"` // Zero = inherit
	UpdatedAt          time.Time `spanner:"UpdatedAt"`
}

// RetailerVariantSettings dictates SKU-level overrides (highest precedence).
type RetailerVariantSettings struct {
	RetailerID         string    `spanner:"RetailerId"`
	SkuID              string    `spanner:"SkuId"`
	AutoOrderEnabled   bool      `spanner:"AutoOrderEnabled"`
	AnalyticsStartDate time.Time `spanner:"AnalyticsStartDate"` // Zero = inherit
	UpdatedAt          time.Time `spanner:"UpdatedAt"`
}

// ==========================================
// 2. API REQUEST PAYLOADS (PATCH)
// ==========================================

// UpdateGlobalSettingsReq handles PATCH /v1/retailer/settings/auto-order/global
type UpdateGlobalSettingsReq struct {
	Enabled    bool  `json:"enabled"`
	UseHistory *bool `json:"use_history,omitempty"` // nil = not specified, true = keep analytics, false = start fresh
}

// UpdateSupplierSettingsReq handles PATCH /v1/retailer/settings/auto-order/supplier/{supplier_id}
type UpdateSupplierSettingsReq struct {
	Enabled    bool  `json:"enabled"`
	UseHistory *bool `json:"use_history,omitempty"`
}

// UpdateCategorySettingsReq handles PATCH /v1/retailer/settings/auto-order/category/{category_id}
type UpdateCategorySettingsReq struct {
	Enabled    bool  `json:"enabled"`
	UseHistory *bool `json:"use_history,omitempty"`
}

// UpdateProductSettingsReq handles PATCH /v1/retailer/settings/auto-order/product/{product_id}
type UpdateProductSettingsReq struct {
	Enabled    bool  `json:"enabled"`
	UseHistory *bool `json:"use_history,omitempty"`
}

// UpdateVariantSettingsReq handles PATCH /v1/retailer/settings/auto-order/variant/{sku_id}
type UpdateVariantSettingsReq struct {
	Enabled    bool  `json:"enabled"`
	UseHistory *bool `json:"use_history,omitempty"`
}

// ==========================================
// 3. API RESPONSE: GET ALL SETTINGS
// ==========================================

// AutoOrderSettingsResponse is returned by GET /v1/retailer/settings/auto-order
type AutoOrderSettingsResponse struct {
	GlobalEnabled      bool                       `json:"global_enabled"`
	HasAnyHistory      bool                       `json:"has_any_history"`
	AnalyticsStartDate *string                    `json:"analytics_start_date,omitempty"`
	SupplierOverrides  []SupplierOverrideResponse `json:"supplier_overrides"`
	CategoryOverrides  []CategoryOverrideResponse `json:"category_overrides"`
	ProductOverrides   []ProductOverrideResponse  `json:"product_overrides"`
	VariantOverrides   []VariantOverrideResponse  `json:"variant_overrides"`
}

type SupplierOverrideResponse struct {
	SupplierID         string  `json:"supplier_id"`
	Enabled            bool    `json:"enabled"`
	HasHistory         bool    `json:"has_history"`
	AnalyticsStartDate *string `json:"analytics_start_date,omitempty"`
}

type CategoryOverrideResponse struct {
	CategoryID         string  `json:"category_id"`
	Enabled            bool    `json:"enabled"`
	HasHistory         bool    `json:"has_history"`
	AnalyticsStartDate *string `json:"analytics_start_date,omitempty"`
}

type ProductOverrideResponse struct {
	ProductID          string  `json:"product_id"`
	Enabled            bool    `json:"enabled"`
	HasHistory         bool    `json:"has_history"`
	AnalyticsStartDate *string `json:"analytics_start_date,omitempty"`
}

type VariantOverrideResponse struct {
	SkuID              string  `json:"sku_id"`
	Enabled            bool    `json:"enabled"`
	HasHistory         bool    `json:"has_history"`
	AnalyticsStartDate *string `json:"analytics_start_date,omitempty"`
}

// ==========================================
// 4. CRON JOB EVALUATION ENGINE
// ==========================================

// EmpathyEngineResolution represents the compiled state in memory when the
// Field General Cron runs to determine if an auto-order should fire.
type EmpathyEngineResolution struct {
	RetailerID             string
	GlobalAutoOrderEnabled bool
	SupplierOverrides      map[string]bool // Key: SupplierID, Value: Enabled
	CategoryOverrides      map[string]bool // Key: CategoryID, Value: Enabled
	ProductOverrides       map[string]bool // Key: ProductID, Value: Enabled
	VariantOverrides       map[string]bool // Key: SkuID, Value: Enabled
}

// ShouldAutoOrder resolves the hierarchy: Variant > Product > Category > Supplier > Global
func (e *EmpathyEngineResolution) ShouldAutoOrder(supplierID, categoryID, productID, skuID string) bool {
	// 1. Check Variant (SKU) level override (Highest precedence)
	if enabled, exists := e.VariantOverrides[skuID]; exists {
		return enabled
	}

	// 2. Check Product-level override
	if enabled, exists := e.ProductOverrides[productID]; exists {
		return enabled
	}

	// 3. Check Category-level override
	if enabled, exists := e.CategoryOverrides[categoryID]; exists {
		return enabled
	}

	// 4. Check Supplier-level override
	if enabled, exists := e.SupplierOverrides[supplierID]; exists {
		return enabled
	}

	// 5. Fallback to Global master switch (Lowest precedence)
	return e.GlobalAutoOrderEnabled
}
