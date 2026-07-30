package compliance

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"log/slog"
	"time"
)

type Service struct {
	repo Repository
	log  *slog.Logger
}

func NewService(repo Repository, log *slog.Logger) *Service {
	return &Service{
		repo: repo,
		log:  log,
	}
}

func (s *Service) GetDashboard(ctx context.Context, f DashboardFilter) (DashboardStats, []ProblemOrder, error) {
	stats, err := s.repo.FetchDashboardStats(ctx, f)
	if err != nil {
		return DashboardStats{}, nil, fmt.Errorf("fetch dashboard stats: %w", err)
	}

	orders, err := s.repo.ListProblemOrders(ctx, f, 50)
	if err != nil {
		return DashboardStats{}, nil, fmt.Errorf("list problem orders: %w", err)
	}

	return stats, orders, nil
}

func (s *Service) ExportCSV(ctx context.Context, f DashboardFilter) ([]byte, error) {
	orders, err := s.repo.ExportProblemOrders(ctx, f)
	if err != nil {
		return nil, fmt.Errorf("export problem orders: %w", err)
	}

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Header: order_id,status,fiscal_status,ehf_id,buyer_acceptance_status,force_completed_at,force_reason,claim_id,claimed_amount_minor,created_at
	if err := writer.Write([]string{
		"order_id", "status", "fiscal_status", "ehf_id", "buyer_acceptance_status",
		"force_completed_at", "force_reason", "claim_id", "claimed_amount_minor", "created_at",
	}); err != nil {
		return nil, fmt.Errorf("write csv header: %w", err)
	}

	for _, o := range orders {
		var forceCompletedAt string
		if o.ForceCompletedAt != nil {
			forceCompletedAt = o.ForceCompletedAt.Format(time.RFC3339)
		}
		
		var claimedAmount string
		if o.ClaimedAmountMinor > 0 {
			claimedAmount = fmt.Sprintf("%d", o.ClaimedAmountMinor)
		}

		record := []string{
			o.OrderID,
			o.Status,
			o.FiscalStatus,
			o.EhfID,
			o.BuyerAcceptanceStatus,
			forceCompletedAt,
			o.ForceReason,
			o.ClaimID,
			claimedAmount,
			o.CreatedAt.Format(time.RFC3339),
		}
		if err := writer.Write(record); err != nil {
			return nil, fmt.Errorf("write csv record: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("csv flush: %w", err)
	}

	return buf.Bytes(), nil
}
