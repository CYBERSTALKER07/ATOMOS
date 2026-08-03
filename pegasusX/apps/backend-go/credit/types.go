package credit

import (
	"errors"
	"time"
)

// RiskTier classifies a retailer's credit worthiness.
type RiskTier string

const (
	RiskTierLow    RiskTier = "LOW"
	RiskTierMedium RiskTier = "MEDIUM"
	RiskTierHigh   RiskTier = "HIGH"
	RiskTierBlock  RiskTier = "BLOCK"
)

// Valid returns true for known risk tiers.
func (r RiskTier) Valid() bool {
	switch r {
	case RiskTierLow, RiskTierMedium, RiskTierHigh, RiskTierBlock:
		return true
	}
	return false
}

// Status is the lifecycle state of a credit profile.
type Status string

const (
	StatusInactive    Status = "INACTIVE" // never enabled or admin-disabled
	StatusActive      Status = "ACTIVE"
	StatusFrozen      Status = "FROZEN" // temporary hold; relationship still ON
	StatusClosed      Status = "CLOSED"
	StatusBlacklisted Status = "BLACKLISTED"
)

// Valid returns true for known statuses.
func (s Status) Valid() bool {
	switch s {
	case StatusInactive, StatusActive, StatusFrozen, StatusClosed, StatusBlacklisted:
		return true
	}
	return false
}

// ReservationStatus tracks order-level credit hold.
type ReservationStatus string

const (
	ReservationReserved  ReservationStatus = "RESERVED"
	ReservationConverted ReservationStatus = "CONVERTED"
	ReservationReleased  ReservationStatus = "RELEASED"
)

// RetailerCreditScore represents the overall credit score and risk tier.
type RetailerCreditScore struct {
	RetailerID          string    `json:"retailer_id"`
	Score               int64     `json:"score"`
	RiskTier            RiskTier  `json:"risk_tier"`
	SuggestedLimitMinor int64     `json:"suggested_limit_minor"`
	FactorsJSON         string    `json:"factors_json"`
	WindowStart         time.Time `json:"window_start"`
	WindowEnd           time.Time `json:"window_end"`
	ComputedAt          time.Time `json:"computed_at"`
}

// Profile is the retailer credit profile aggregate per supplier.
type Profile struct {
	RetailerID           string    `json:"retailer_id"`
	SupplierID           string    `json:"supplier_id"`
	CreditLimitMinor     int64     `json:"credit_limit_minor"`
	CurrentBalanceMinor  int64     `json:"current_balance_minor"`
	ReservedMinor        int64     `json:"reserved_minor"`
	AvailableCreditMinor int64     `json:"available_credit_minor"`
	RiskScore            int64     `json:"risk_score"`
	RiskTier             RiskTier  `json:"risk_tier"`
	DelinquencyCount     int64     `json:"delinquency_count"`
	Status               Status    `json:"status"`
	LastEvaluatedAt      time.Time `json:"last_evaluated_at,omitempty"`
	Version              int64     `json:"version"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// Available computes headroom: limit - balance - reserved.
func (p Profile) Available() int64 {
	avail := p.CreditLimitMinor - p.CurrentBalanceMinor - p.ReservedMinor
	if avail < 0 {
		return 0
	}
	return avail
}

// CheckResult is the outcome of an order credit check.
type CheckResult struct {
	Allowed          bool   `json:"allowed"`
	CreditLimitMinor int64  `json:"credit_limit_minor"`
	CurrentBalance   int64  `json:"current_balance"`
	ReservedMinor    int64  `json:"reserved_minor,omitempty"`
	RequestedAmount  int64  `json:"requested_amount"`
	Shortfall        int64  `json:"shortfall,omitempty"`
	Reason           string `json:"reason,omitempty"`
	DueAt            string `json:"due_at,omitempty"`
	TermsDays        int64  `json:"terms_days,omitempty"`
}

// OrderReservation is a per-order credit hold.
type OrderReservation struct {
	OrderID     string            `json:"order_id"`
	RetailerID  string            `json:"retailer_id"`
	SupplierID  string            `json:"supplier_id"`
	AmountMinor int64             `json:"amount_minor"`
	Status      ReservationStatus `json:"status"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// Common errors.
var (
	ErrProfileNotFound    = errors.New("credit_profile_not_found")
	ErrProfileFrozen      = errors.New("credit_profile_frozen")
	ErrProfileBlacklisted = errors.New("credit_profile_blacklisted")
	ErrLimitBreached      = errors.New("credit_limit_breached")
	ErrReservationExists  = errors.New("credit_reservation_exists")
	ErrNoReservation      = errors.New("credit_reservation_not_found")
	ErrWarningAckRequired = errors.New("warning_ack_required")
	ErrDisableRequiresSupport = errors.New("credit_disable_requires_support")
	ErrCreditNotEnabled   = errors.New("credit_relationship_not_enabled")
	ErrProgramDisabled    = errors.New("credit_program_not_enabled")
)
