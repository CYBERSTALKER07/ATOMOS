package compliance

import (
	"context"
	"encoding/json"
	"time"
)

type DashboardFilter struct {
	SupplierID string
	From       time.Time
	To         time.Time
}

type DashboardStats struct {
	Fiscalizing             int64 `json:"fiscalizing"`
	FiscalFailed            int64 `json:"fiscalFailed"`
	ForceCompleted          int64 `json:"forceCompleted"`
	BuyerAcceptancePending  int64 `json:"buyerAcceptancePending"`
	BuyerAcceptanceRejected int64 `json:"buyerAcceptanceRejected"`
	ClaimMismatches         int64 `json:"claimMismatches"`
	CreditFrozen            int64 `json:"creditFrozen"`
}

type ProblemOrder struct {
	OrderID               string     `json:"orderId"`
	Status                string     `json:"status"`
	FiscalStatus          string     `json:"fiscalStatus"`
	EhfID                 string     `json:"ehfId,omitempty"`
	BuyerAcceptanceStatus string     `json:"buyerAcceptanceStatus,omitempty"`
	ForceCompletedAt      *time.Time `json:"forceCompletedAt,omitempty"`
	ForceReason           string     `json:"forceReason,omitempty"`
	ClaimID               string     `json:"claimId,omitempty"`
	ClaimedAmountMinor    int64      `json:"claimedAmountMinor,omitempty"`
	CreatedAt             time.Time  `json:"createdAt"`
}

type Repository interface {
	FetchDashboardStats(ctx context.Context, f DashboardFilter) (DashboardStats, error)
	ListProblemOrders(ctx context.Context, f DashboardFilter, limit int) ([]ProblemOrder, error)
	ExportProblemOrders(ctx context.Context, f DashboardFilter) ([]ProblemOrder, error) // unbounded within date range
	ListExceptions(ctx context.Context, f ExceptionFilter) ([]ExceptionTicket, error)
	UpdateExceptionStatus(ctx context.Context, ticketID, newStatus string) error
}

type ExceptionTicket struct {
	TicketID     string          `json:"ticketId"`
	Type         string          `json:"type"`
	OrderID      string          `json:"orderId"`
	EhfID        string          `json:"ehfId,omitempty"`
	Severity     string          `json:"severity"`
	Status       string          `json:"status"`
	Title        string          `json:"title"`
	Description  string          `json:"description"`
	AssignedRole string          `json:"assignedRole"`
	CreatedAt    time.Time       `json:"createdAt"`
	CreatedBy    string          `json:"createdBy"`
	Payload      json.RawMessage `json:"payload"`
}

type ExceptionFilter struct {
	Status   string
	Severity string
}
