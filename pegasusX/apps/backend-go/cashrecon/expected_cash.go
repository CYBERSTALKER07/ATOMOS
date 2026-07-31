package cashrecon

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

const (
	methodCash           = "CASH"
	paymentStatusCaptured = "CAPTURED"
)

// ShiftBounds returns UTC [start, end) for a calendar shift date.
func ShiftBounds(shiftDate time.Time) (time.Time, time.Time) {
	d := civil.DateOf(shiftDate.UTC())
	start := time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, time.UTC)
	return start, start.Add(24 * time.Hour)
}

// ComputeExpectedCashMinor sums captured CASH payment legs for a driver shift.
func (r *SpannerRepository) ComputeExpectedCashMinor(ctx context.Context, driverID string, shiftDate time.Time, routeID *string) (int64, error) {
	if r == nil || r.client == nil {
		return 0, fmt.Errorf("cashrecon repository unavailable")
	}
	driverID = strings.TrimSpace(driverID)
	if driverID == "" {
		return 0, fmt.Errorf("driver_id required")
	}
	start, end := ShiftBounds(shiftDate)
	params := map[string]interface{}{
		"driverId": driverID,
		"method":   methodCash,
		"status":   paymentStatusCaptured,
		"start":    start,
		"end":      end,
	}
	sql := `SELECT COALESCE(SUM(pl.AmountMinor), 0)
FROM OrderPaymentLegs pl
JOIN Orders o ON o.OrderId = pl.OrderId
WHERE o.DriverId = @driverId
  AND pl.Method = @method
  AND pl.Status = @status
  AND o.CreatedAt >= @start
  AND o.CreatedAt < @end`
	if routeID != nil && strings.TrimSpace(*routeID) != "" {
		sql += ` AND o.RouteId = @routeId`
		params["routeId"] = strings.TrimSpace(*routeID)
	}
	iter := r.client.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	var total int64
	if err := row.Column(0, &total); err != nil {
		return 0, err
	}
	return total, nil
}

// HasAcceptedReconciliation reports whether the driver has a clean reconciliation for the shift.
func (r *SpannerRepository) HasAcceptedReconciliation(ctx context.Context, driverID string, shiftDate time.Time) (bool, error) {
	if r == nil || r.client == nil {
		return false, fmt.Errorf("cashrecon repository unavailable")
	}
	driverID = strings.TrimSpace(driverID)
	if driverID == "" {
		return false, nil
	}
	d := civil.DateOf(shiftDate.UTC())
	iter := r.client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT COUNT(*) FROM CashReconciliations
		      WHERE DriverId = @driverId AND ShiftDate = @shiftDate AND Status = @status`,
		Params: map[string]interface{}{
			"driverId":   driverID,
			"shiftDate":  d,
			"status":     string(ReconciliationStatusAccepted),
		},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return false, err
	}
	var cnt int64
	if err := row.Column(0, &cnt); err != nil {
		return false, err
	}
	return cnt > 0, nil
}
