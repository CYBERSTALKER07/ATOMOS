package compliance

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"
)

// mockRepo is a test double for the compliance Repository.
type mockRepo struct {
	stats       DashboardStats
	orders      []ProblemOrder
	statsErr    error
	ordersErr   error
	exportErr   error
}

func (m *mockRepo) FetchDashboardStats(ctx context.Context, f DashboardFilter) (DashboardStats, error) {
	return m.stats, m.statsErr
}

func (m *mockRepo) ListProblemOrders(ctx context.Context, f DashboardFilter, limit int) ([]ProblemOrder, error) {
	return m.orders, m.ordersErr
}

func (m *mockRepo) ExportProblemOrders(ctx context.Context, f DashboardFilter) ([]ProblemOrder, error) {
	return m.orders, m.exportErr
}

func TestService_ExportCSV(t *testing.T) {
	t.Run("success_with_records", func(t *testing.T) {
		t1 := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
		t2 := time.Date(2026, 1, 1, 11, 0, 0, 0, time.UTC)

		repo := &mockRepo{
			orders: []ProblemOrder{
				{
					OrderID:               "ORD-1",
					Status:                "COMPLETED",
					FiscalStatus:          "FAILED",
					EhfID:                 "EHF-123",
					BuyerAcceptanceStatus: "PENDING",
					ForceCompletedAt:      &t1,
					ForceReason:           "API timeout",
					ClaimID:               "CLM-1",
					ClaimedAmountMinor:    10050,
					CreatedAt:             t2,
				},
				{
					OrderID:               "ORD-2",
					Status:                "CANCELLED",
					FiscalStatus:          "NONE",
					BuyerAcceptanceStatus: "NONE",
					CreatedAt:             t2,
				},
			},
		}

		svc := NewService(repo, slog.Default())

		csvData, err := svc.ExportCSV(context.Background(), DashboardFilter{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		csvStr := string(csvData)

		expectedLines := []string{
			"order_id,status,fiscal_status,ehf_id,buyer_acceptance_status,force_completed_at,force_reason,claim_id,claimed_amount_minor,created_at",
			"ORD-1,COMPLETED,FAILED,EHF-123,PENDING,2026-01-01T10:00:00Z,API timeout,CLM-1,10050,2026-01-01T11:00:00Z",
			"ORD-2,CANCELLED,NONE,,NONE,,,,,2026-01-01T11:00:00Z",
		}

		lines := strings.Split(strings.TrimSpace(csvStr), "\n")
		if len(lines) != 3 {
			t.Fatalf("expected 3 lines, got %d:\n%s", len(lines), csvStr)
		}

		for i, expected := range expectedLines {
			if lines[i] != expected {
				t.Errorf("line %d mismatch:\nwant: %s\ngot:  %s", i, expected, lines[i])
			}
		}
	})

	t.Run("empty_records", func(t *testing.T) {
		repo := &mockRepo{
			orders: []ProblemOrder{},
		}
		svc := NewService(repo, slog.Default())

		csvData, err := svc.ExportCSV(context.Background(), DashboardFilter{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		csvStr := string(csvData)
		expected := "order_id,status,fiscal_status,ehf_id,buyer_acceptance_status,force_completed_at,force_reason,claim_id,claimed_amount_minor,created_at\n"

		if csvStr != expected {
			t.Errorf("expected only headers, got: %s", csvStr)
		}
	})
}
