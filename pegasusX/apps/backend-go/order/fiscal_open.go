package order

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// OpenFiscalSnapshot is the policy-gated shift-end check for driver fiscal state (D-10).
// Blocking orders (unrecorded cash collection, missing QR/cryptographic PoD, or strict
// immediate fiscal mode) increment Count/OrderIDs and set Frozen = true.
// Retryable OFD-outage orders with captured payment and PoD increment RetryQueuedCount
// and FiscalIssueCount without freezing the driver's shift-end cash bag.
type OpenFiscalSnapshot struct {
	Count            int64    `json:"open_fiscal_count"`
	OrderIDs         []string `json:"order_ids,omitempty"`
	Frozen           bool     `json:"cash_bag_frozen"`
	RetryQueuedCount int64    `json:"retry_queued_count"`
	FiscalIssueCount int64    `json:"fiscal_issue_count"`
	RetryOrderIDs    []string `json:"retry_order_ids,omitempty"`
}

// OpenFiscalCandidate captures the minimal per-order state needed to evaluate whether
// an unfinished fiscal receipt blocks driver shift-end or routes cleanly to async retry.
type OpenFiscalCandidate struct {
	OrderID            string
	Status             string
	FiscalStatus       string
	HasPaymentRecorded bool
	HasCryptoOrQRPoD   bool
}

// IsStrictImmediateFiscalMode returns true when deployment policy requires synchronous
// fiscal completion before driver shift-end regardless of PoD/payment capture.
func IsStrictImmediateFiscalMode() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("FISCAL_STRICT_IMMEDIATE_BLOCK")))
	return v == "1" || v == "true" || v == "yes"
}

// EvaluateOpenFiscalOrders applies the D-10 non-blocking doorstep fiscalization policy.
// Retryable OFD outage orders with captured payment + QR/cryptographic PoD never freeze
// the cash bag unless strictImmediate is true.
func EvaluateOpenFiscalOrders(strictImmediate bool, candidates []OpenFiscalCandidate) OpenFiscalSnapshot {
	var snap OpenFiscalSnapshot
	seen := make(map[string]struct{}, len(candidates))
	for _, c := range candidates {
		oid := strings.TrimSpace(c.OrderID)
		if oid == "" {
			continue
		}
		if _, dup := seen[oid]; dup {
			continue
		}
		seen[oid] = struct{}{}

		st := strings.ToUpper(strings.TrimSpace(c.Status))
		fst := strings.ToUpper(strings.TrimSpace(c.FiscalStatus))
		if st == string(StatusMigrationOpen) || fst == FiscalStatusHistoricalSkipped {
			continue
		}
		isRetryableState := st == string(StatusFiscalizing) ||
			st == string(StatusFiscalFailed) ||
			fst == FiscalStatusPending ||
			fst == FiscalStatusFailed ||
			fst == "FISCAL_RETRY_QUEUED" ||
			fst == FiscalStatusDeadLetter
		if !isRetryableState {
			continue
		}

		snap.FiscalIssueCount++
		if !strictImmediate && c.HasPaymentRecorded && c.HasCryptoOrQRPoD {
			snap.RetryQueuedCount++
			snap.RetryOrderIDs = append(snap.RetryOrderIDs, oid)
			continue
		}

		snap.OrderIDs = append(snap.OrderIDs, oid)
		snap.Count++
	}
	snap.Frozen = snap.Count > 0
	return snap
}

// CountOpenFiscalForDriver evaluates orders assigned to the driver whose fiscalization
// is pending, failed, retry-queued, or dead-lettered against D-10 non-blocking policy.
func CountOpenFiscalForDriver(ctx context.Context, client *spanner.Client, driverID string) (OpenFiscalSnapshot, error) {
	return queryOpenFiscalForDriver(ctx, client, driverID, 0)
}

// CountOpenFiscalForDriverStale is a UI read with slight staleness (dashboard banner).
func CountOpenFiscalForDriverStale(ctx context.Context, client *spanner.Client, driverID string, stale time.Duration) (OpenFiscalSnapshot, error) {
	return queryOpenFiscalForDriver(ctx, client, driverID, stale)
}

func queryOpenFiscalForDriver(ctx context.Context, client *spanner.Client, driverID string, stale time.Duration) (OpenFiscalSnapshot, error) {
	driverID = strings.TrimSpace(driverID)
	if client == nil || driverID == "" {
		return OpenFiscalSnapshot{}, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT o.OrderId,
		             o.Status,
		             COALESCE(o.FiscalStatus, ''),
		             (EXISTS(SELECT 1 FROM OrderPaymentLegs l WHERE l.OrderId = o.OrderId AND l.Status = 'CAPTURED' AND l.Method != 'CREDIT' AND l.IdempotencyKey NOT LIKE 'credit-leave-%')
		              OR EXISTS(SELECT 1 FROM OrderDeliveryProofs p WHERE p.OrderId = o.OrderId AND p.ProofType = 'CASH_COLLECTION_GEOFENCE')
		              OR EXISTS(SELECT 1 FROM OrderFiscalReceipts r WHERE r.OrderId = o.OrderId AND r.Status = 'SUCCESS')),
		             EXISTS(SELECT 1 FROM OrderDeliveryProofs p
		                    WHERE p.OrderId = o.OrderId
		                      AND p.ProofType IN ('QR_HANDOFF', 'OFFLINE_CRYPTOGRAPHIC_POD')
		                      AND (COALESCE(p.QRTokenHash, '') != '' OR COALESCE(p.ScannedTokenHash, '') != ''))
		      FROM Orders o
		      WHERE o.DriverId = @did
		        AND o.Status != 'MIGRATION_OPEN'
		        AND COALESCE(o.OrderSource, '') != 'MIGRATION'
		        AND COALESCE(o.FiscalStatus, '') != 'HISTORICAL_SKIPPED'
		        AND (o.Status IN (@st1, @st2) OR o.FiscalStatus IN ('FAILED', 'FISCAL_RETRY_QUEUED', 'DEAD_LETTER'))
		      ORDER BY o.UpdatedAt DESC
		      LIMIT 50`,
		Params: map[string]any{
			"did": driverID,
			"st1": string(StatusFiscalizing),
			"st2": string(StatusFiscalFailed),
		},
	}
	ro := client.Single()
	if stale > 0 {
		ro = ro.WithTimestampBound(spanner.ExactStaleness(stale))
	}
	iter := ro.Query(ctx, stmt)
	defer iter.Stop()
	candidates := make([]OpenFiscalCandidate, 0, 8)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return OpenFiscalSnapshot{}, fmt.Errorf("open fiscal query: %w", err)
		}
		var cand OpenFiscalCandidate
		if err := row.Columns(&cand.OrderID, &cand.Status, &cand.FiscalStatus, &cand.HasPaymentRecorded, &cand.HasCryptoOrQRPoD); err != nil {
			return OpenFiscalSnapshot{}, err
		}
		candidates = append(candidates, cand)
	}
	return EvaluateOpenFiscalOrders(IsStrictImmediateFiscalMode(), candidates), nil
}
