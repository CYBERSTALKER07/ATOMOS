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
	StatusActive      Status = "ACTIVE"
	StatusFrozen      Status = "FROZEN"
	StatusClosed      Status = "CLOSED"
	StatusBlacklisted Status = "BLACKLISTED"
)

// Valid returns true for known statuses.
func (s Status) Valid() bool {
	switch s {
	case StatusActive, StatusFrozen, StatusClosed, StatusBlacklisted:
		return true
	}
	return false
}

// Profile is the retailer credit profile aggregate per supplier.
type Profile struct {
	RetailerID           string    `json:"retailer_id"`
	SupplierID           string    `json:"supplier_id"`
	CreditLimitMinor     int64     `json:"credit_limit_minor"`
	CurrentBalanceMinor  int64     `json:"current_balance_minor"`
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

// CheckResult is the outcome of an order credit check.
type CheckResult struct {
	Allowed          bool   `json:"allowed"`
	CreditLimitMinor int64  `json:"credit_limit_minor"`
	CurrentBalance   int64  `json:"current_balance"`
	RequestedAmount  int64  `json:"requested_amount"`
	Shortfall        int64  `json:"shortfall,omitempty"`
	Reason           string `json:"reason,omitempty"`
}

// Common errors.
var (
	ErrProfileNotFound    = errors.New("credit_profile_not_found")
	ErrProfileFrozen      = errors.New("credit_profile_frozen")
	ErrProfileBlacklisted = errors.New("credit_profile_blacklisted")
	ErrLimitBreached      = errors.New("credit_limit_breached")
)
