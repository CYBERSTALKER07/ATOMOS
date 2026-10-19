package order

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
)

func newConditionTestService(repo Repository, now time.Time) *Service {
	return NewService(ServiceConfig{
		Repo:       repo,
		SupplierID: "sup-1",
		Currency:   "UZS",
		Now: func() time.Time {
			return now
		},
		Log: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
}

func TestReportCondition_AuthorizedDriver(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	repo := &testRepo{
		found: true,
		order: Order{
			OrderID:    "ord-1",
			SupplierID: "sup-1",
			RetailerID: "ret-1",
			DriverID:   "drv-1",
			Status:     StatusInTransit,
		},
	}
	svc := newConditionTestService(repo, now)

	claims := auth.Claims{Role: auth.RoleDriver, Subject: "drv-1"}
	report, err := svc.ReportCondition(context.Background(), claims, ConditionReportRequest{
		OrderID:       "ord-1",
		ConditionType: ConditionTypeDamaged,
		Severity:      SeverityHigh,
		Description:   "crushed box",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.ConditionType != ConditionTypeDamaged {
		t.Fatalf("expected DAMAGED, got %s", report.ConditionType)
	}
	if report.Severity != SeverityHigh {
		t.Fatalf("expected HIGH, got %s", report.Severity)
	}
	if len(repo.conditionReports) != 1 {
		t.Fatalf("expected 1 stored report, got %d", len(repo.conditionReports))
	}
	if repo.bufferedEvents != 1 {
		t.Fatalf("expected 1 outbox event, got %d", repo.bufferedEvents)
	}
	if !bytes.Contains(repo.lastEvents[0].Payload, []byte(events.EventOrderConditionReported)) {
		t.Fatalf("expected ORDER_CONDITION_REPORTED payload, got %s", string(repo.lastEvents[0].Payload))
	}
}

func TestReportCondition_TemperatureBreachIncludesPartyIDs(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	repo := &testRepo{
		found: true,
		order: Order{
			OrderID:    "ord-1",
			SupplierID: "sup-1",
			RetailerID: "ret-1",
			DriverID:   "drv-1",
			Status:     StatusInTransit,
		},
	}
	svc := newConditionTestService(repo, now)

	claims := auth.Claims{Role: auth.RoleDriver, Subject: "drv-1"}
	_, err := svc.ReportCondition(context.Background(), claims, ConditionReportRequest{
		OrderID:       "ord-1",
		ConditionType: ConditionTypeTemperatureBreach,
		Severity:      SeverityHigh,
		Description:   "probe reading 22C",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	payload := string(repo.lastEvents[0].Payload)
	if !bytes.Contains(repo.lastEvents[0].Payload, []byte(`"supplier_id":"sup-1"`)) {
		t.Fatalf("expected supplier_id in payload, got %s", payload)
	}
	if !bytes.Contains(repo.lastEvents[0].Payload, []byte(`"retailer_id":"ret-1"`)) {
		t.Fatalf("expected retailer_id in payload, got %s", payload)
	}
	if !bytes.Contains(repo.lastEvents[0].Payload, []byte(`"condition_type":"TEMPERATURE_BREACH"`)) {
		t.Fatalf("expected TEMPERATURE_BREACH, got %s", payload)
	}
}

func TestReportCondition_RetailerReportsOwnOrder(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	repo := &testRepo{
		found: true,
		order: Order{
			OrderID:    "ord-1",
			SupplierID: "sup-1",
			RetailerID: "ret-1",
			Status:     StatusArrived,
		},
	}
	svc := newConditionTestService(repo, now)

	claims := auth.Claims{Role: auth.RoleRetailer, Subject: "ret-1"}
	_, err := svc.ReportCondition(context.Background(), claims, ConditionReportRequest{
		OrderID:       "ord-1",
		ConditionType: ConditionTypeQualityReject,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReportCondition_ForbiddenWrongDriver(t *testing.T) {
	repo := &testRepo{
		found: true,
		order: Order{
			OrderID:    "ord-1",
			SupplierID: "sup-1",
			RetailerID: "ret-1",
			DriverID:   "drv-1",
			Status:     StatusInTransit,
		},
	}
	svc := newConditionTestService(repo, time.Now())

	claims := auth.Claims{Role: auth.RoleDriver, Subject: "drv-2"}
	_, err := svc.ReportCondition(context.Background(), claims, ConditionReportRequest{
		OrderID:       "ord-1",
		ConditionType: ConditionTypeDamaged,
	})
	if !errors.Is(err, ErrOrderForbidden) {
		t.Fatalf("expected ErrOrderForbidden, got %v", err)
	}
}

func TestReportCondition_InvalidStatus(t *testing.T) {
	repo := &testRepo{
		found: true,
		order: Order{
			OrderID:    "ord-1",
			SupplierID: "sup-1",
			RetailerID: "ret-1",
			DriverID:   "drv-1",
			Status:     StatusPending,
		},
	}
	svc := newConditionTestService(repo, time.Now())

	claims := auth.Claims{Role: auth.RoleDriver, Subject: "drv-1"}
	_, err := svc.ReportCondition(context.Background(), claims, ConditionReportRequest{
		OrderID:       "ord-1",
		ConditionType: ConditionTypeDamaged,
	})
	if err == nil {
		t.Fatal("expected error for non-reportable status")
	}
}

func TestReportCondition_InvalidConditionType(t *testing.T) {
	repo := &testRepo{found: true, order: Order{OrderID: "ord-1", DriverID: "drv-1", Status: StatusInTransit}}
	svc := newConditionTestService(repo, time.Now())

	claims := auth.Claims{Role: auth.RoleDriver, Subject: "drv-1"}
	_, err := svc.ReportCondition(context.Background(), claims, ConditionReportRequest{
		OrderID:       "ord-1",
		ConditionType: ConditionType("UNKNOWN"),
	})
	if err == nil {
		t.Fatal("expected error for invalid condition type")
	}
}

func TestConditionReportable(t *testing.T) {
	reportable := []Status{StatusInTransit, StatusArrived, StatusAwaitingPayment, StatusPendingCashCollection, StatusDeliveredOnCredit, StatusCompleted}
	for _, s := range reportable {
		if !conditionReportable(s) {
			t.Errorf("expected status %s to be reportable", s)
		}
	}
	nonReportable := []Status{StatusPending, StatusLoaded, StatusCancelled}
	for _, s := range nonReportable {
		if conditionReportable(s) {
			t.Errorf("expected status %s to be non-reportable", s)
		}
	}
}

func TestNormalizeSeverity(t *testing.T) {
	if normalizeSeverity(SeverityLow) != SeverityLow {
		t.Error("expected LOW to remain LOW")
	}
	if normalizeSeverity(SeverityHigh) != SeverityHigh {
		t.Error("expected HIGH to remain HIGH")
	}
	if normalizeSeverity(Severity("")) != SeverityMedium {
		t.Error("expected empty severity to default to MEDIUM")
	}
}
