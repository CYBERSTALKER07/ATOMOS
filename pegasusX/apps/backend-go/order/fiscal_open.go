package order

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// OpenFiscalSnapshot is the soft-freeze signal for driver shift-end (Phase 6 / T10).
// Cash bag must not close while any assigned order is still fiscalizing or fiscal-failed.
type OpenFiscalSnapshot struct {
	Count    int64    `json:"open_fiscal_count"`
	OrderIDs []string `json:"order_ids,omitempty"`
	// Frozen is true when Count > 0 — blocks return-complete / cash-bag settle.
	Frozen bool `json:"cash_bag_frozen"`
}

// CountOpenFiscalForDriver counts orders assigned to the driver in FISCALIZING or FISCAL_FAILED.
func CountOpenFiscalForDriver(ctx context.Context, client *spanner.Client, driverID string) (OpenFiscalSnapshot, error) {
	var snap OpenFiscalSnapshot
	driverID = strings.TrimSpace(driverID)
	if client == nil || driverID == "" {
		return snap, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT OrderId FROM Orders
		      WHERE DriverId = @did
		        AND Status IN (@st1, @st2)
		      ORDER BY UpdatedAt DESC
		      LIMIT 50`,
		Params: map[string]any{
			"did": driverID,
			"st1": string(StatusFiscalizing),
			"st2": string(StatusFiscalFailed),
		},
	}
	// Strong enough for shift-end gate — avoid false-clear from 15s stale.
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return snap, fmt.Errorf("open fiscal query: %w", err)
		}
		var oid string
		if err := row.Columns(&oid); err != nil {
			return snap, err
		}
		oid = strings.TrimSpace(oid)
		if oid == "" {
			continue
		}
		snap.OrderIDs = append(snap.OrderIDs, oid)
		snap.Count++
	}
	snap.Frozen = snap.Count > 0
	return snap, nil
}

// CountOpenFiscalForDriverStale is a UI read with slight staleness (dashboard banner).
func CountOpenFiscalForDriverStale(ctx context.Context, client *spanner.Client, driverID string, stale time.Duration) (OpenFiscalSnapshot, error) {
	if stale <= 0 {
		return CountOpenFiscalForDriver(ctx, client, driverID)
	}
	// Reuse same SQL via strong path when staleness is tiny; for true stale use bound.
	// Spanner ExactStaleness is applied on the read below.
	var snap OpenFiscalSnapshot
	driverID = strings.TrimSpace(driverID)
	if client == nil || driverID == "" {
		return snap, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT OrderId FROM Orders
		      WHERE DriverId = @did
		        AND Status IN (@st1, @st2)
		      ORDER BY UpdatedAt DESC
		      LIMIT 50`,
		Params: map[string]any{
			"did": driverID,
			"st1": string(StatusFiscalizing),
			"st2": string(StatusFiscalFailed),
		},
	}
	iter := client.Single().WithTimestampBound(spanner.ExactStaleness(stale)).Query(ctx, stmt)
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return snap, fmt.Errorf("open fiscal stale query: %w", err)
		}
		var oid string
		if err := row.Columns(&oid); err != nil {
			return snap, err
		}
		oid = strings.TrimSpace(oid)
		if oid == "" {
			continue
		}
		snap.OrderIDs = append(snap.OrderIDs, oid)
		snap.Count++
	}
	snap.Frozen = snap.Count > 0
	return snap, nil
}
