package compliance

import (
	"context"
	"golang.org/x/sync/errgroup"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

type SpannerRepository struct {
	client *spanner.Client
}

func NewSpannerRepository(client *spanner.Client) *SpannerRepository {
	return &SpannerRepository{
		client: client,
	}
}

func (r *SpannerRepository) FetchDashboardStats(ctx context.Context, f DashboardFilter) (DashboardStats, error) {
	var stats DashboardStats
	g, gCtx := errgroup.WithContext(ctx)

	// 1. Fiscalizing
	g.Go(func() error {
		iter := r.client.Single().Query(gCtx, spanner.Statement{
			SQL: `SELECT COUNT(*) AS cnt
FROM Orders
WHERE SupplierId = @supplierId
  AND FiscalStatus = 'FISCALIZING'
  AND CreatedAt >= @from
  AND CreatedAt < @to;`,
			Params: map[string]interface{}{"supplierId": f.SupplierID, "from": f.From, "to": f.To},
		})
		defer iter.Stop()
		if row, err := iter.Next(); err == nil {
			_ = row.Columns(&stats.Fiscalizing)
		}
		return nil
	})

	// 2. FiscalFailed
	g.Go(func() error {
		iter := r.client.Single().Query(gCtx, spanner.Statement{
			SQL: `SELECT COUNT(*) AS cnt
FROM Orders
WHERE SupplierId = @supplierId
  AND FiscalStatus = 'FISCAL_FAILED'
  AND CreatedAt >= @from
  AND CreatedAt < @to;`,
			Params: map[string]interface{}{"supplierId": f.SupplierID, "from": f.From, "to": f.To},
		})
		defer iter.Stop()
		if row, err := iter.Next(); err == nil {
			_ = row.Columns(&stats.FiscalFailed)
		}
		return nil
	})

	// 3. ForceCompleted
	g.Go(func() error {
		iter := r.client.Single().Query(gCtx, spanner.Statement{
			SQL: `SELECT COUNT(*) AS cnt
FROM Orders
WHERE SupplierId = @supplierId
  AND FiscalStatus = 'FORCE_COMPLETED'
  AND ForceCompletedAt >= @from
  AND ForceCompletedAt < @to;`,
			Params: map[string]interface{}{"supplierId": f.SupplierID, "from": f.From, "to": f.To},
		})
		defer iter.Stop()
		if row, err := iter.Next(); err == nil {
			_ = row.Columns(&stats.ForceCompleted)
		}
		return nil
	})

	// 4. BuyerAcceptancePending
	g.Go(func() error {
		iter := r.client.Single().Query(gCtx, spanner.Statement{
			SQL: `SELECT COUNT(*) AS cnt
FROM Orders
WHERE SupplierId = @supplierId
  AND BuyerAcceptanceStatus = 'PENDING'
  AND EhfSubmittedAt IS NOT NULL
  AND CreatedAt >= @from
  AND CreatedAt < @to;`,
			Params: map[string]interface{}{"supplierId": f.SupplierID, "from": f.From, "to": f.To},
		})
		defer iter.Stop()
		if row, err := iter.Next(); err == nil {
			_ = row.Columns(&stats.BuyerAcceptancePending)
		}
		return nil
	})

	// 5. BuyerAcceptanceRejected
	g.Go(func() error {
		iter := r.client.Single().Query(gCtx, spanner.Statement{
			SQL: `SELECT COUNT(*) AS cnt
FROM Orders
WHERE SupplierId = @supplierId
  AND BuyerAcceptanceStatus = 'REJECTED'
  AND CreatedAt >= @from
  AND CreatedAt < @to;`,
			Params: map[string]interface{}{"supplierId": f.SupplierID, "from": f.From, "to": f.To},
		})
		defer iter.Stop()
		if row, err := iter.Next(); err == nil {
			_ = row.Columns(&stats.BuyerAcceptanceRejected)
		}
		return nil
	})

	// 6. Claim mismatches
	g.Go(func() error {
		iter := r.client.Single().Query(gCtx, spanner.Statement{
			SQL: `SELECT COUNT(*) AS cnt
FROM Claims c
JOIN Orders o ON c.OrderId = o.OrderId
WHERE o.SupplierId = @supplierId
  AND c.Status IN ('OPEN', 'APPROVED')
  AND (
       o.FiscalStatus = 'FORCE_COMPLETED'
    OR o.BuyerAcceptanceStatus = 'REJECTED'
    OR c.ClaimedAmountMinor > (
         SELECT IFNULL(SUM(ol.GrossMinor), 0) - IFNULL((
           SELECT SUM(c2.ClaimedAmountMinor)
           FROM Claims c2
           WHERE c2.OrderId = o.OrderId
             AND c2.Status IN ('APPROVED', 'SETTLED')
             AND c2.ClaimId != c.ClaimId
         ), 0)
         FROM OrderLineFiscalSnapshots ol
         WHERE ol.OrderId = o.OrderId
       )
  )
  AND o.CreatedAt >= @from
  AND o.CreatedAt < @to;`,
			Params: map[string]interface{}{"supplierId": f.SupplierID, "from": f.From, "to": f.To},
		})
		defer iter.Stop()
		if row, err := iter.Next(); err == nil {
			_ = row.Columns(&stats.ClaimMismatches)
		}
		return nil
	})

	// 7. Credit frozen
	g.Go(func() error {
		iter := r.client.Single().Query(gCtx, spanner.Statement{
			SQL: `SELECT COUNT(*) AS cnt
FROM CreditProfiles
WHERE SupplierId = @supplierId
  AND Status = 'FROZEN';`,
			Params: map[string]interface{}{"supplierId": f.SupplierID},
		})
		defer iter.Stop()
		if row, err := iter.Next(); err == nil {
			_ = row.Columns(&stats.CreditFrozen)
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return stats, err
	}
	return stats, nil
}

func (r *SpannerRepository) ListProblemOrders(ctx context.Context, f DashboardFilter, limit int) ([]ProblemOrder, error) {
	stmt := spanner.Statement{
		SQL: `SELECT
  o.OrderId,
  o.Status,
  o.FiscalStatus,
  o.EhfId,
  o.BuyerAcceptanceStatus,
  o.ForceCompletedAt,
  o.ForceCompleteReason,
  c.ClaimId,
  c.ClaimedAmountMinor,
  o.CreatedAt
FROM Orders o
LEFT JOIN Claims c
  ON c.OrderId = o.OrderId
 AND c.Status IN ('OPEN', 'APPROVED')
WHERE o.SupplierId = @supplierId
  AND o.CreatedAt >= @from
  AND o.CreatedAt < @to
  AND (
       o.FiscalStatus IN ('FISCALIZING', 'FISCAL_FAILED', 'FORCE_COMPLETED')
    OR o.BuyerAcceptanceStatus IN ('PENDING', 'REJECTED')
    OR c.ClaimId IS NOT NULL
  )
ORDER BY o.CreatedAt DESC
LIMIT @limit;`,
		Params: map[string]interface{}{
			"supplierId": f.SupplierID,
			"from":       f.From,
			"to":         f.To,
			"limit":      limit,
		},
	}
	return runProblemOrdersQuery(ctx, r.client, stmt)
}

func (r *SpannerRepository) ExportProblemOrders(ctx context.Context, f DashboardFilter) ([]ProblemOrder, error) {
	stmt := spanner.Statement{
		SQL: `SELECT
  o.OrderId,
  o.Status,
  o.FiscalStatus,
  o.EhfId,
  o.BuyerAcceptanceStatus,
  o.ForceCompletedAt,
  o.ForceCompleteReason,
  c.ClaimId,
  c.ClaimedAmountMinor,
  o.CreatedAt
FROM Orders o
LEFT JOIN Claims c
  ON c.OrderId = o.OrderId
 AND c.Status IN ('OPEN', 'APPROVED')
WHERE o.SupplierId = @supplierId
  AND o.CreatedAt >= @from
  AND o.CreatedAt < @to
  AND (
       o.FiscalStatus IN ('FISCALIZING', 'FISCAL_FAILED', 'FORCE_COMPLETED')
    OR o.BuyerAcceptanceStatus IN ('PENDING', 'REJECTED')
    OR c.ClaimId IS NOT NULL
  )
ORDER BY o.CreatedAt DESC;`,
		Params: map[string]interface{}{
			"supplierId": f.SupplierID,
			"from":       f.From,
			"to":         f.To,
		},
	}
	return runProblemOrdersQuery(ctx, r.client, stmt)
}

func runProblemOrdersQuery(ctx context.Context, client *spanner.Client, stmt spanner.Statement) ([]ProblemOrder, error) {
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var rows []ProblemOrder
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var r ProblemOrder
		var status spanner.NullString
		var fiscalStatus spanner.NullString
		var ehfID spanner.NullString
		var buyerAcceptanceStatus spanner.NullString
		var forceCompletedAt spanner.NullTime
		var forceReason spanner.NullString
		var claimID spanner.NullString
		var claimedAmountMinor spanner.NullInt64

		if err := row.Columns(
			&r.OrderID, &status, &fiscalStatus, &ehfID, &buyerAcceptanceStatus, 
			&forceCompletedAt, &forceReason, &claimID, &claimedAmountMinor, &r.CreatedAt,
		); err != nil {
			return nil, err
		}

		if status.Valid {
			r.Status = status.StringVal
		}
		if fiscalStatus.Valid {
			r.FiscalStatus = fiscalStatus.StringVal
		}
		if ehfID.Valid {
			r.EhfID = ehfID.StringVal
		}
		if buyerAcceptanceStatus.Valid {
			r.BuyerAcceptanceStatus = buyerAcceptanceStatus.StringVal
		}
		if forceCompletedAt.Valid {
			r.ForceCompletedAt = &forceCompletedAt.Time
		}
		if forceReason.Valid {
			r.ForceReason = forceReason.StringVal
		}
		if claimID.Valid {
			r.ClaimID = claimID.StringVal
		}
		if claimedAmountMinor.Valid {
			r.ClaimedAmountMinor = claimedAmountMinor.Int64
		}

		rows = append(rows, r)
	}
	if rows == nil {
		rows = []ProblemOrder{}
	}
	return rows, nil
}
