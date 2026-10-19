package globalproducts

import (
	"os"
	"strings"
	"time"
)

const (
	StatusLinked   = "LINKED"
	StatusPending  = "PENDING"
	StatusRejected = "REJECTED"
	StatusAccepted = "ACCEPTED"

	MethodExactGTIN = "EXACT_GTIN"
	MethodFuzzy     = "FUZZY"
	MethodManual    = "MANUAL"

	UomEachID   = "uom-each"
	UomInnerID  = "uom-inner"
	UomCaseID   = "uom-case"
	UomPalletID = "uom-pallet"
)

// Enabled reports whether GLOBAL_PRODUCTS_ENABLED is on.
func Enabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("GLOBAL_PRODUCTS_ENABLED")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// UnitOfMeasure is a pack hierarchy node.
type UnitOfMeasure struct {
	UomID        string `json:"uom_id"`
	Code         string `json:"code"`
	Name         string `json:"name"`
	FactorToBase int64  `json:"factor_to_base"`
	ParentUomID  string `json:"parent_uom_id,omitempty"`
}

// GlobalBrand represents a canonical brand entity.
type GlobalBrand struct {
	BrandID         string    `json:"brand_id"`
	Name            string    `json:"name"`
	NormalizedName  string    `json:"normalized_name"`
	OwnerSupplierID string    `json:"owner_supplier_id,omitempty"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// GlobalProduct is the cross-supplier product master.
type GlobalProduct struct {
	GlobalProductID string    `json:"global_product_id"`
	Gtin            string    `json:"gtin,omitempty"`
	BrandID         string    `json:"brand_id"`
	Manufacturer    string    `json:"manufacturer,omitempty"`
	Name            string    `json:"name"`
	PackQty         int64     `json:"pack_qty"`
	BaseUomID       string    `json:"base_uom_id"`
	NormalizedKey   string    `json:"normalized_key,omitempty"`
	Version         int64     `json:"version"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Offer maps a supplier SKU onto a global product.
type Offer struct {
	SupplierID      string    `json:"supplier_id"`
	ProductID       string    `json:"product_id"`
	GlobalProductID string    `json:"global_product_id"`
	PriceMinor      int64     `json:"price_minor"`
	Currency        string    `json:"currency"`
	Moq             int64     `json:"moq"`
	LeadTimeDays    int64     `json:"lead_time_days"`
	Status          string    `json:"status"`
	Version         int64     `json:"version"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// MatchQueueItem is a human-review row for ambiguous matches.
type MatchQueueItem struct {
	QueueID                  string    `json:"queue_id"`
	SupplierID               string    `json:"supplier_id"`
	ProductID                string    `json:"product_id"`
	CandidateGlobalProductID string    `json:"candidate_global_product_id,omitempty"`
	MatchMethod              string    `json:"match_method"`
	Score                    float64   `json:"score"`
	Status                   string    `json:"status"`
	Reason                   string    `json:"reason,omitempty"`
	Version                  int64     `json:"version"`
	CreatedAt                time.Time `json:"created_at"`
	UpdatedAt                time.Time `json:"updated_at"`
}

// ProductInput is the catalog SKU snapshot used for matching.
type ProductInput struct {
	ProductID    string
	SupplierID   string
	Name         string
	Brand        string
	Barcode      string
	PriceMinor   int64
	Currency     string
	UnitsPerPack int64
	UomCode      string
}
