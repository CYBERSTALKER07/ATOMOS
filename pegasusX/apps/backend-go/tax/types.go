package tax

import (
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrRegimeNotFound     = errors.New("tax_regime_not_found")
	ErrRegimeOverlap      = errors.New("tax_regime_overlap")
	ErrRegimeInvalid      = errors.New("tax_regime_invalid")
	ErrRegimeForbidden    = errors.New("tax_regime_forbidden")
	ErrMissingCountryCode = errors.New("country_code_required")
)

// TaxRegimeVersion is a versioned, temporal tax configuration.
type TaxRegimeVersion struct {
    Id            string          `json:"id"`
    CountryCode   string          `json:"country_code"`
    EffectiveFrom time.Time       `json:"effective_from"`
    EffectiveTo   *time.Time      `json:"effective_to,omitempty"`
    Currency      string          `json:"currency"`
    VatRateBps    int64           `json:"vat_rate_bps"`
    Simplified    bool            `json:"simplified"`
    RulesJson     json.RawMessage `json:"rules_json,omitempty"`
    CreatedAt     time.Time       `json:"created_at"`
    CreatedBy     string          `json:"created_by"`
    UpdatedAt     time.Time       `json:"updated_at"`
}

// OrderLineFiscalSnapshot is an immutable record of the tax configuration
// applied to a specific order line at COMPLETED time.
type OrderLineFiscalSnapshot struct {
    OrderId     string    `json:"order_id"`
    OrderLineId string    `json:"order_line_id"`
    RegimeId    string    `json:"regime_id"`
    VatRateBps  int64     `json:"vat_rate_bps"`
    NetMinor    int64     `json:"net_minor"`
    VatMinor    int64     `json:"vat_minor"`
    GrossMinor  int64     `json:"gross_minor"`
    SnapshotAt  time.Time `json:"snapshot_at"`
    CreatedAt   time.Time `json:"created_at"`
}

// CreateRegimeRequest is the input for creating a new tax regime version.
type CreateRegimeRequest struct {
	CountryCode   string          `json:"country_code"`
	EffectiveFrom time.Time       `json:"effective_from"`
	EffectiveTo   *time.Time      `json:"effective_to,omitempty"`
	Currency      string          `json:"currency"`
	VatRateBps    int64           `json:"vat_rate_bps"`
	Simplified    bool            `json:"simplified"`
	RulesJson     json.RawMessage `json:"rules_json,omitempty"`
}
