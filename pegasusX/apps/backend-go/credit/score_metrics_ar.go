package credit

import (
	"context"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ARScoreMetrics loads DPD + payment cadence from ARInvoices / credit reservations.
type ARScoreMetrics struct {
	Client *spanner.Client
}

// LoadScoreMetrics implements ScoreMetricsProvider.
func (a *ARScoreMetrics) LoadScoreMetrics(ctx context.Context, retailerID, supplierID string) (ScoreMetrics, error) {
	var m ScoreMetrics
	if a == nil || a.Client == nil {
		return m, nil
	}
	retailerID = strings.TrimSpace(retailerID)
	supplierID = strings.TrimSpace(supplierID)
	if retailerID == "" {
		return m, nil
	}
	now := time.Now().UTC()
	// Max DPD on OPEN invoices for this relationship.
	stmt := spanner.Statement{
		SQL: `SELECT DueAt FROM ArInvoices
		      WHERE RetailerId = @rid AND SupplierId = @sid AND Status IN ('OPEN','PARTIAL')
		      LIMIT 50`,
		Params: map[string]interface{}{"rid": retailerID, "sid": supplierID},
	}
	iter := a.Client.Single().Query(ctx, stmt)
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			// Table missing / schema drift — fail soft.
			return m, nil
		}
		var due time.Time
		if err := row.Columns(&due); err != nil {
			continue
		}
		if due.IsZero() || !now.After(due) {
			continue
		}
		dpd := int64(now.Sub(due).Hours() / 24)
		if dpd > m.MaxDaysPastDue {
			m.MaxDaysPastDue = dpd
		}
		m.ExpectedPayments90++
	}
	// Cleared credit reservations in last 90d as pay velocity proxy.
	since := now.Add(-90 * 24 * time.Hour)
	pstmt := spanner.Statement{
		SQL: `SELECT COUNT(1) FROM OrderCreditReservations
		      WHERE RetailerId = @rid AND SupplierId = @sid
		        AND Status = 'CLEARED' AND UpdatedAt >= @since`,
		Params: map[string]interface{}{"rid": retailerID, "sid": supplierID, "since": since},
	}
	piter := a.Client.Single().Query(ctx, pstmt)
	defer piter.Stop()
	if row, err := piter.Next(); err == nil {
		var n int64
		if err := row.Columns(&n); err == nil {
			m.PaymentsLast90d = n
		}
	}
	if m.ExpectedPayments90 == 0 && m.PaymentsLast90d > 0 {
		m.ExpectedPayments90 = m.PaymentsLast90d
	}
	return m, nil
}
