package retailer

import (
	"os"
	"strings"
)

const (
	AutoOrderModeOff    = "off"
	AutoOrderModeShadow = "shadow"
	AutoOrderModeDraft  = "draft"
	AutoOrderModePlace  = "place"
)

// NormalizeExecutionMode returns a canonical mode. Empty → draft (backward compat).
func NormalizeExecutionMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case AutoOrderModeOff:
		return AutoOrderModeOff
	case AutoOrderModeShadow:
		return AutoOrderModeShadow
	case AutoOrderModePlace:
		return AutoOrderModePlace
	case AutoOrderModeDraft, "":
		return AutoOrderModeDraft
	default:
		return ""
	}
}

// ValidExecutionMode reports whether mode is one of the supported values.
func ValidExecutionMode(mode string) bool {
	return NormalizeExecutionMode(mode) != "" || strings.TrimSpace(mode) == ""
}

// AutoOrderShadowEnabled gates shadow proposal persistence (SSMR on by default via env).
func AutoOrderShadowEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("AUTO_ORDER_SHADOW_ENABLED")))
	if v == "" {
		return true // shadow path available unless explicitly disabled
	}
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// AutoOrderInventoryGrounded prefers (R,s,S) proposals over synthesis /2 orders.
func AutoOrderInventoryGrounded() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("AUTO_ORDER_INVENTORY_GROUNDED")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// IsAutoOrderActive is true when the retailer has any path that can produce candidates.
func IsAutoOrderActive(s AutoOrderSettings) bool {
	mode := NormalizeExecutionMode(s.ExecutionMode)
	if mode == AutoOrderModeOff {
		return false
	}
	return s.GlobalEnabled || hasAnyScopedEnable(s)
}

// candidateAllowed: most-specific override wins; Enabled=false at any matching scope blocks.
// When global off, only scoped enables allow. Category is honored.
func candidateAllowed(settings AutoOrderSettings, c AutoOrderCandidate) bool {
	// Variant (size/SKU) — most specific
	for _, o := range settings.VariantOverrides {
		if o.VariantID == c.SKU || (c.ProductID != "" && o.VariantID == c.ProductID) {
			return o.Enabled
		}
	}
	// Product
	for _, o := range settings.ProductOverrides {
		if o.ProductID == c.ProductID || o.ProductID == c.SKU {
			return o.Enabled
		}
	}
	// Category
	if cat := strings.TrimSpace(c.CategoryID); cat != "" {
		for _, o := range settings.CategoryOverrides {
			if o.CategoryID == cat {
				return o.Enabled
			}
		}
	}
	// Supplier
	if sid := strings.TrimSpace(c.SupplierID); sid != "" {
		for _, o := range settings.SupplierOverrides {
			if o.SupplierID == sid {
				return o.Enabled
			}
		}
	}
	if settings.GlobalEnabled {
		return true
	}
	// Global off: any scoped enable for this candidate already returned above;
	// remaining path is "no override matched" → deny.
	return false
}
