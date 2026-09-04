package factory

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// Factory SLA honesty labels (G7.1) — handoff timing vs RequestedDeliveryDate / default window.
const (
	SLAStatusOnTime   = "ON_TIME"
	SLAStatusAtRisk   = "AT_RISK"
	SLAStatusBreached = "BREACHED"
	SLAStatusMet      = "MET"
	SLAStatusNA       = "N/A"
)

// Open supply states still counting against handoff SLA.
var slaOpenStates = map[string]struct{}{
	"SUBMITTED":     {},
	"ACKNOWLEDGED":  {},
	"IN_PRODUCTION": {},
	"READY":         {},
}

// Terminal success states (handoff complete).
var slaMetStates = map[string]struct{}{
	"SHIPPED":   {},
	"FULFILLED": {},
	"COMPLETED": {},
	"RECEIVED":  {},
}

// SLAEval is the pure SLA snapshot attached to a supply request.
type SLAEval struct {
	DueAt          time.Time
	Status         string
	HoursRemaining *float64 // nil when N/A
	Open           bool
}

// FactorySLADefaultHours is the due window when RequestedDeliveryDate is empty.
// Env override wins; otherwise the shipped pack (UZ 48). Planned pack → 0 (N/A).
func FactorySLADefaultHours() float64 {
	if strings.TrimSpace(os.Getenv("FACTORY_SLA_DEFAULT_HOURS")) != "" {
		return envFloat("FACTORY_SLA_DEFAULT_HOURS", 0)
	}
	hours, err := auth.FactorySLADefaultHoursFromContext(context.Background(), "")
	if err != nil {
		return 0
	}
	return hours
}

// FactorySLAAtRiskHours is how long before due the request becomes AT_RISK (default 6).
func FactorySLAAtRiskHours() float64 {
	return envFloat("FACTORY_SLA_AT_RISK_HOURS", 6)
}

func envFloat(key string, def float64) float64 {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	n, err := strconv.ParseFloat(v, 64)
	if err != nil || n < 0 {
		return def
	}
	return n
}

// EvaluateSLA computes handoff SLA status.
// dueFromRequest: RequestedDeliveryDate if set; else CreatedAt + default hours.
func EvaluateSLA(state string, createdAt, requestedDelivery time.Time, now time.Time) SLAEval {
	state = strings.ToUpper(strings.TrimSpace(state))
	now = now.UTC()
	createdAt = createdAt.UTC()

	var due time.Time
	if !requestedDelivery.IsZero() {
		due = requestedDelivery.UTC()
	} else if !createdAt.IsZero() {
		h := FactorySLADefaultHours()
		if h <= 0 {
			return SLAEval{Status: SLAStatusNA}
		}
		due = createdAt.Add(time.Duration(h * float64(time.Hour)))
	} else {
		return SLAEval{Status: SLAStatusNA}
	}

	_, open := slaOpenStates[state]
	_, met := slaMetStates[state]
	eval := SLAEval{DueAt: due, Open: open}

	hoursLeft := due.Sub(now).Hours()
	eval.HoursRemaining = &hoursLeft

	if met {
		// Completed: MET if finished before due (UpdatedAt unknown → treat as MET when terminal).
		eval.Status = SLAStatusMet
		return eval
	}
	if !open {
		// Cancelled / unknown terminal → not scored.
		eval.Status = SLAStatusNA
		eval.HoursRemaining = nil
		return eval
	}

	if now.After(due) {
		eval.Status = SLAStatusBreached
		return eval
	}
	atRiskWindow := time.Duration(FactorySLAAtRiskHours() * float64(time.Hour))
	if now.After(due.Add(-atRiskWindow)) {
		eval.Status = SLAStatusAtRisk
		return eval
	}
	eval.Status = SLAStatusOnTime
	return eval
}

// EnrichSupplyRequestSLA mutates a DTO map with sla_* fields.
func EnrichSupplyRequestSLA(m map[string]any, state, createdAtStr, deliveryDateStr string, now time.Time) {
	created, _ := time.Parse(time.RFC3339Nano, createdAtStr)
	if created.IsZero() {
		created, _ = time.Parse(time.RFC3339, createdAtStr)
	}
	var delivery time.Time
	if deliveryDateStr != "" {
		delivery, _ = time.Parse(time.RFC3339Nano, deliveryDateStr)
		if delivery.IsZero() {
			delivery, _ = time.Parse(time.RFC3339, deliveryDateStr)
		}
	}
	eval := EvaluateSLA(state, created, delivery, now)
	if !eval.DueAt.IsZero() {
		m["sla_due_at"] = eval.DueAt.UTC().Format(time.RFC3339)
	}
	m["sla_status"] = eval.Status
	if eval.HoursRemaining != nil {
		// Round to 1 decimal for wire.
		h := float64(int(*eval.HoursRemaining*10)) / 10
		m["sla_hours_remaining"] = h
	}
}

// SLABoardSummary aggregates open requests by status.
type SLABoardSummary struct {
	OnTime    int `json:"on_time"`
	AtRisk    int `json:"at_risk"`
	Breached  int `json:"breached"`
	TotalOpen int `json:"total_open"`
}

func slaStatusRank(status string) int {
	switch status {
	case SLAStatusBreached:
		return 0
	case SLAStatusAtRisk:
		return 1
	case SLAStatusOnTime:
		return 2
	case SLAStatusMet:
		return 3
	default:
		return 4
	}
}
