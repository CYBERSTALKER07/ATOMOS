package promotion

import "time"

// ScopeType defines which catalog rows a promotion targets.
type ScopeType string

const (
	ScopeTypeProduct     ScopeType = "PRODUCT"
	ScopeTypeCategory    ScopeType = "CATEGORY"
	ScopeTypeAllProducts ScopeType = "ALL_PRODUCTS"
)

// RetailerScope defines whether a promotion applies to every retailer or an allowlist.
type RetailerScope string

const (
	RetailerScopeAll       RetailerScope = "ALL"
	RetailerScopeAllowlist RetailerScope = "ALLOWLIST"
)

// Promotion is the supplier promotion aggregate persisted in Spanner.
type Promotion struct {
	PromotionID         string        `json:"promotion_id"`
	SupplierID          string        `json:"supplier_id"`
	Name                string        `json:"name"`
	Description         string        `json:"description,omitempty"`
	DiscountBps         int64         `json:"discount_bps"`
	ScopeType           ScopeType     `json:"scope_type"`
	ScopeProductID      string        `json:"scope_product_id,omitempty"`
	ScopeCategoryID     string        `json:"scope_category_id,omitempty"`
	RetailerScope       RetailerScope `json:"retailer_scope"`
	RetailerIDs         []string      `json:"retailer_ids,omitempty"`
	MinLineQuantity     int64         `json:"min_line_quantity,omitempty"`
	MinOrderAmountMinor int64         `json:"min_order_amount_minor,omitempty"`
	StartsAt            *time.Time    `json:"starts_at,omitempty"`
	EndsAt              *time.Time    `json:"ends_at,omitempty"`
	MaxRedemptions      int64         `json:"max_redemptions,omitempty"`
	CurrentRedemptions  int64         `json:"current_redemptions"`
	IsActive            bool          `json:"is_active"`
	Priority            int64         `json:"priority"`
	Version             int64         `json:"version"`
	CreatedAt           time.Time     `json:"created_at"`
	UpdatedAt           time.Time     `json:"updated_at"`
}

// LineInput is one checkout or quote line before promotion application.
type LineInput struct {
	ProductID       string `json:"product_id"`
	CategoryID      string `json:"category_id"`
	Quantity        int64  `json:"quantity"`
	UnitPrice       int64  `json:"unit_price_minor"`
	Currency        string `json:"currency"`
	PriceIsOverride bool   `json:"-"`
}

// QuotedLine is a priced line after the best applicable promotion is chosen.
type QuotedLine struct {
	ProductID        string  `json:"product_id"`
	Quantity         int64   `json:"quantity"`
	ListUnitPrice    int64   `json:"list_unit_price_minor"`
	UnitPrice        int64   `json:"unit_price_minor"`
	LineTotal        int64   `json:"line_total_minor"`
	Currency         string  `json:"currency"`
	DiscountBps      int64   `json:"discount_bps,omitempty"`
	PromotionID      string  `json:"promotion_id,omitempty"`
	PromotionName    string  `json:"promotion_name,omitempty"`
	PromotionLabel   string  `json:"promotion_label,omitempty"`
}

// QuoteResult is the checkout quote payload for retailer clients.
type QuoteResult struct {
	SupplierID       string       `json:"supplier_id"`
	RetailerID       string       `json:"retailer_id"`
	Lines            []QuotedLine `json:"lines"`
	SubtotalMinor    int64        `json:"subtotal_minor"`
	DiscountMinor    int64        `json:"discount_minor"`
	TotalMinor       int64        `json:"total_minor"`
	Currency         string       `json:"currency"`
}

// ProductOffer enriches catalog products with optional sale pricing for a retailer.
type ProductOffer struct {
	ProductID        string  `json:"product_id"`
	ListPriceMinor   int64   `json:"list_price_minor"`
	SalePriceMinor   *int64  `json:"sale_price_minor,omitempty"`
	DiscountBps      *int64  `json:"discount_bps,omitempty"`
	PromotionID      *string `json:"promotion_id,omitempty"`
	PromotionName    *string `json:"promotion_name,omitempty"`
	PromotionLabel   *string `json:"promotion_label,omitempty"`
	PromotionEndsAt  *string `json:"promotion_ends_at,omitempty"`
}
