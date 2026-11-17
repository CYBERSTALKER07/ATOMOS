package order

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestBuildScheduledPreordersPageStatement(t *testing.T) {
	first := buildScheduledPreordersPageStatement(0, false, nil, "")
	if !strings.Contains(first.SQL, "ORDER BY RequestedDeliveryDate ASC, OrderId ASC") {
		t.Fatalf("unexpected ORDER BY in first page SQL: %s", first.SQL)
	}
	if strings.Contains(first.SQL, "@afterOrderId") {
		t.Fatalf("first page SQL should not have cursor predicate: %s", first.SQL)
	}
	if first.Params["lim"] != 200 {
		t.Fatalf("default lim = %v, want 200", first.Params["lim"])
	}

	nullCursor := buildScheduledPreordersPageStatement(900, true, nil, "ord-001")
	if !strings.Contains(nullCursor.SQL, "(RequestedDeliveryDate IS NULL AND OrderId > @afterOrderId) OR RequestedDeliveryDate IS NOT NULL") {
		t.Fatalf("missing null-date cursor predicate in SQL: %s", nullCursor.SQL)
	}
	if nullCursor.Params["afterOrderId"] != "ord-001" {
		t.Fatalf("afterOrderId = %v, want ord-001", nullCursor.Params["afterOrderId"])
	}
	if nullCursor.Params["lim"] != 500 {
		t.Fatalf("clamped lim = %v, want 500", nullCursor.Params["lim"])
	}

	afterDate := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	dateCursor := buildScheduledPreordersPageStatement(50, true, &afterDate, "ord-002")
	if !strings.Contains(dateCursor.SQL, "(RequestedDeliveryDate = @afterDate AND OrderId > @afterOrderId) OR RequestedDeliveryDate > @afterDate") {
		t.Fatalf("missing dated cursor predicate in SQL: %s", dateCursor.SQL)
	}
	if dateCursor.Params["afterDate"] != afterDate {
		t.Fatalf("afterDate = %v, want %v", dateCursor.Params["afterDate"], afterDate)
	}
	if dateCursor.Params["lim"] != 50 {
		t.Fatalf("lim = %v, want 50", dateCursor.Params["lim"])
	}
}

func TestBuildPreorderPromotePageStatement(t *testing.T) {
	cutoff := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	first := buildPreorderPromotePageStatement(cutoff, 0, nil, "")
	if !strings.Contains(first.SQL, "ORDER BY RequestedDeliveryDate ASC, OrderId ASC") {
		t.Fatalf("unexpected ORDER BY in promote SQL: %s", first.SQL)
	}
	if strings.Contains(first.SQL, "@afterOrderId") {
		t.Fatalf("first page promote SQL should not have cursor predicate: %s", first.SQL)
	}
	if first.Params["cutoff"] != cutoff || first.Params["lim"] != 100 {
		t.Fatalf("unexpected params: %+v", first.Params)
	}

	afterDate := time.Date(2026, 6, 18, 8, 0, 0, 0, time.UTC)
	next := buildPreorderPromotePageStatement(cutoff, 1000, &afterDate, "ord-010")
	if !strings.Contains(next.SQL, "(RequestedDeliveryDate > @afterDate OR (RequestedDeliveryDate = @afterDate AND OrderId > @afterOrderId))") {
		t.Fatalf("missing cursor predicate in promote SQL: %s", next.SQL)
	}
	if next.Params["afterDate"] != afterDate || next.Params["afterOrderId"] != "ord-010" || next.Params["lim"] != 500 {
		t.Fatalf("unexpected cursor params: %+v", next.Params)
	}
}

func TestBuildExpiredNegotiationsPageStatement(t *testing.T) {
	now := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	first := buildExpiredNegotiationsPageStatement(now, 0, nil, "")
	if !strings.Contains(first.SQL, "ORDER BY ExpiresAt ASC, ProposalId ASC") {
		t.Fatalf("unexpected ORDER BY in negotiation sweep SQL: %s", first.SQL)
	}
	if strings.Contains(first.SQL, "@afterProposalId") {
		t.Fatalf("first page negotiation sweep SQL should not have cursor predicate: %s", first.SQL)
	}
	if first.Params["lim"] != 100 {
		t.Fatalf("default lim = %v, want 100", first.Params["lim"])
	}

	afterExp := now.Add(-10 * time.Minute)
	next := buildExpiredNegotiationsPageStatement(now, 800, &afterExp, "prop-009")
	if !strings.Contains(next.SQL, "(ExpiresAt > @afterExpiresAt OR (ExpiresAt = @afterExpiresAt AND ProposalId > @afterProposalId))") {
		t.Fatalf("missing cursor predicate in negotiation sweep SQL: %s", next.SQL)
	}
	if next.Params["afterExpiresAt"] != afterExp || next.Params["afterProposalId"] != "prop-009" || next.Params["lim"] != 500 {
		t.Fatalf("unexpected cursor params: %+v", next.Params)
	}
}

func TestIsOptionalCreditSkipReason(t *testing.T) {
	for _, allow := range []string{
		"no_credit_profile",
		"credit_program_not_enabled",
		"credit_relationship_not_enabled",
	} {
		if !isOptionalCreditSkipReason(allow) {
			t.Fatalf("expected %q to be treated as optional credit skip", allow)
		}
	}
	for _, deny := range []string{
		"",
		"credit_limit_breached",
		"profile_frozen",
		"overdue_invoices_exist",
	} {
		if isOptionalCreditSkipReason(deny) {
			t.Fatalf("expected %q to fail closed instead of skipping", deny)
		}
	}
}

func TestServiceCreate_AtomicFailureReturnsCleanError(t *testing.T) {
	now := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	wantErr := errors.New("atomic txn aborted")
	repo := &testRepo{createErr: wantErr}
	resolver := &testWarehouseResolver{warehouseID: "wh-1"}
	svc := newTestServiceWithResolver(repo, resolver, now)

	resp, err := svc.Create(context.Background(), "ret-1", CreateRequest{
		LineItems: []LineItem{{SKU: "sku-1", Quantity: 2, UnitPrice: 500}},
		H3Cell:    "872830828ffffff",
		Lat:       41.3,
		Lng:       69.2,
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
	if resp.OrderID != "" {
		t.Fatalf("expected empty response OrderID on rollback, got %q", resp.OrderID)
	}
	if repo.bufferedEvents != 0 {
		t.Fatalf("expected 0 buffered outbox events on failure, got %d", repo.bufferedEvents)
	}
}
