package factory

import (
	"testing"
	"time"
)

func TestEvaluateSLA_BreachedAtRiskOnTime(t *testing.T) {
	created := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	due := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)

	// On time: 24h before due with 6h at-risk window.
	now := due.Add(-24 * time.Hour)
	e := EvaluateSLA("IN_PRODUCTION", created, due, now)
	if e.Status != SLAStatusOnTime {
		t.Fatalf("want ON_TIME got %s", e.Status)
	}

	// At risk: 3h before due.
	now = due.Add(-3 * time.Hour)
	e = EvaluateSLA("READY", created, due, now)
	if e.Status != SLAStatusAtRisk {
		t.Fatalf("want AT_RISK got %s", e.Status)
	}

	// Breached: past due.
	now = due.Add(1 * time.Hour)
	e = EvaluateSLA("SUBMITTED", created, due, now)
	if e.Status != SLAStatusBreached {
		t.Fatalf("want BREACHED got %s", e.Status)
	}
}

func TestEvaluateSLA_DefaultHoursWhenNoDelivery(t *testing.T) {
	t.Setenv("FACTORY_SLA_DEFAULT_HOURS", "24")
	t.Setenv("FACTORY_SLA_AT_RISK_HOURS", "6")
	created := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	// 30h after create → breached under 24h default.
	now := created.Add(30 * time.Hour)
	e := EvaluateSLA("ACKNOWLEDGED", created, time.Time{}, now)
	if e.Status != SLAStatusBreached {
		t.Fatalf("default window: want BREACHED got %s due=%v", e.Status, e.DueAt)
	}
}

func TestEvaluateSLA_TerminalAndNA(t *testing.T) {
	created := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	due := created.Add(48 * time.Hour)
	e := EvaluateSLA("SHIPPED", created, due, created.Add(10*time.Hour))
	if e.Status != SLAStatusMet {
		t.Fatalf("want MET got %s", e.Status)
	}
	e = EvaluateSLA("CANCELLED", created, due, created.Add(10*time.Hour))
	if e.Status != SLAStatusNA {
		t.Fatalf("want N/A got %s", e.Status)
	}
}

func TestEnrichSupplyRequestSLA(t *testing.T) {
	m := map[string]any{}
	created := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	due := created.Add(48 * time.Hour)
	EnrichSupplyRequestSLA(m, "IN_PRODUCTION", created.Format(time.RFC3339), due.Format(time.RFC3339), due.Add(-2*time.Hour))
	if m["sla_status"] != SLAStatusAtRisk {
		t.Fatalf("got %v", m["sla_status"])
	}
	if m["sla_due_at"] == nil || m["sla_hours_remaining"] == nil {
		t.Fatalf("missing sla fields: %+v", m)
	}
}

func TestSlaStatusRank(t *testing.T) {
	if slaStatusRank(SLAStatusBreached) >= slaStatusRank(SLAStatusAtRisk) {
		t.Fatal("breach should rank before at-risk")
	}
}
