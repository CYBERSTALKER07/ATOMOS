package controltower

import (
	"encoding/json"
	"time"
)

const (
	RunModeSuggested = "SUGGESTED"
	RunModeAuto      = "AUTO"

	RunStatusSuggested = "SUGGESTED"
	RunStatusApproved  = "APPROVED"
	RunStatusExecuted  = "EXECUTED"
	RunStatusFailed    = "FAILED"
	RunStatusSkipped   = "SKIPPED"

	ExceptionStatusOpen = "OPEN"
)

// MatchRules is declarative exception filter (Phase 1 — exact match, no expression language).
type MatchRules struct {
	Types            []string `json:"types,omitempty"`
	Severities       []string `json:"severities,omitempty"`
	MinAmountMinor   int64    `json:"minAmountMinor,omitempty"`
	RetailerSegments []string `json:"retailerSegments,omitempty"`
	MinAgeMinutes    int64    `json:"minAgeMinutes,omitempty"`
	MaxAgeMinutes    int64    `json:"maxAgeMinutes,omitempty"`
}

// ActionSpec is one step in a playbook action list.
type ActionSpec struct {
	Type   string          `json:"type"`
	Params json.RawMessage `json:"params,omitempty"`
}

// Playbook is a persisted control-tower rule definition.
type Playbook struct {
	PlaybookID    string          `json:"playbook_id"`
	SupplierID    string          `json:"supplier_id,omitempty"`
	Name          string          `json:"name"`
	Description   string          `json:"description,omitempty"`
	IsActive      bool            `json:"is_active"`
	Priority      int64           `json:"priority"`
	MatchRules    MatchRules      `json:"match_rules"`
	Actions       []ActionSpec    `json:"actions"`
	AutoExecute   bool            `json:"auto_execute"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
	CreatedBy     string          `json:"created_by"`
	MatchRulesRaw json.RawMessage `json:"-"`
	ActionsRaw    json.RawMessage `json:"-"`
}

// PlaybookRun is a matched playbook instance for a concrete exception.
type PlaybookRun struct {
	RunID           string          `json:"run_id"`
	PlaybookID      string          `json:"playbook_id"`
	ExceptionID     string          `json:"exception_id"`
	SupplierID      string          `json:"supplier_id"`
	Mode            string          `json:"mode"`
	Status          string          `json:"status"`
	ActionsResult   []ActionResult  `json:"actions_result,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	ExecutedAt      *time.Time      `json:"executed_at,omitempty"`
	ExecutedBy      string          `json:"executed_by,omitempty"`
	PlaybookName    string          `json:"playbook_name,omitempty"`
	ExceptionType   string          `json:"exception_type,omitempty"`
	ExceptionOrderID string         `json:"exception_order_id,omitempty"`
	ActionsResultRaw json.RawMessage `json:"-"`
}

// ActionResult records per-action execution outcome.
type ActionResult struct {
	Index   int    `json:"index"`
	Type    string `json:"type"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// Exception is the normalized read model for playbook matching.
type Exception struct {
	ExceptionID   string
	Type          string
	Severity      string
	EntityType    string
	EntityID      string
	OrderID       string
	SupplierID    string
	RetailerID    string
	WarehouseID   string
	RouteID       string
	Status        string
	AmountMinor   int64
	RetailerSegment string
	AssignedRole  string
	ClaimID       string
	Payload       json.RawMessage
	CreatedAt     time.Time
}

// CreatePlaybookInput is POST body for new playbook.
type CreatePlaybookInput struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Priority    int64        `json:"priority"`
	MatchRules  MatchRules   `json:"match_rules"`
	Actions     []ActionSpec `json:"actions"`
	AutoExecute bool         `json:"auto_execute"`
}

// UpdatePlaybookInput is PATCH body.
type UpdatePlaybookInput struct {
	Name        *string       `json:"name,omitempty"`
	Description *string       `json:"description,omitempty"`
	Priority    *int64        `json:"priority,omitempty"`
	MatchRules  *MatchRules   `json:"match_rules,omitempty"`
	Actions     *[]ActionSpec `json:"actions,omitempty"`
	AutoExecute *bool         `json:"auto_execute,omitempty"`
	IsActive    *bool         `json:"is_active,omitempty"`
}
