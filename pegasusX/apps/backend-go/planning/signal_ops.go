package planning

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"google.golang.org/api/iterator"
)

// SignalIngestStatus summarizes projection health for supplier ops surfaces.
type SignalIngestStatus struct {
	ProjectionCount        int64  `json:"projection_count"`
	LastIngestAt           string `json:"last_ingest_at,omitempty"`
	LagSeconds             int64  `json:"lag_seconds"`
	BaselineRowsFromSignal int64  `json:"baseline_rows_from_signals"`
	Topic                  string `json:"topic"`
	Healthy                bool   `json:"healthy"`
}

const signalIngestHealthyLag = 5 * time.Minute

// LoadSignalIngestStatus reads projection and baseline counters for a supplier.
func LoadSignalIngestStatus(ctx context.Context, client *spanner.Client, supplierID string, now time.Time) (SignalIngestStatus, error) {
	if client == nil {
		return SignalIngestStatus{}, fmt.Errorf("spanner_unavailable")
	}
	supplierID = strings.TrimSpace(supplierID)
	if supplierID == "" {
		return SignalIngestStatus{}, fmt.Errorf("supplier_id_required")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	out := SignalIngestStatus{
		Topic:   events.TopicPlanningSignalIngest,
		Healthy: true,
	}

	projStmt := spanner.Statement{
		SQL: `SELECT COUNT(*) AS cnt, MAX(IngestedAt) AS last_at
		      FROM PlanningSignalProjections
		      WHERE SupplierId = @sid`,
		Params: map[string]any{"sid": supplierID},
	}
	projIter := client.Single().Query(ctx, projStmt)
	defer projIter.Stop()
	row, err := projIter.Next()
	if err != nil && !errors.Is(err, iterator.Done) {
		return SignalIngestStatus{}, fmt.Errorf("query projections: %w", err)
	}
	if err == nil {
		var lastAt spanner.NullTime
		if scanErr := row.Columns(&out.ProjectionCount, &lastAt); scanErr != nil {
			return SignalIngestStatus{}, fmt.Errorf("scan projections: %w", scanErr)
		}
		if lastAt.Valid {
			last := lastAt.Time.UTC()
			out.LastIngestAt = last.Format(time.RFC3339Nano)
			lag := now.Sub(last)
			if lag < 0 {
				lag = 0
			}
			out.LagSeconds = int64(lag.Seconds())
			if out.ProjectionCount > 0 && lag > signalIngestHealthyLag {
				out.Healthy = false
			}
		}
	}

	baselineStmt := spanner.Statement{
		SQL: `SELECT COUNT(*) AS cnt
		      FROM DemandForecastBaseline
		      WHERE SupplierId = @sid
		        AND BaselineSource = @src`,
		Params: map[string]any{
			"sid": supplierID,
			"src": "signal_ingest",
		},
	}
	baseIter := client.Single().Query(ctx, baselineStmt)
	defer baseIter.Stop()
	baseRow, err := baseIter.Next()
	if err != nil && !errors.Is(err, iterator.Done) {
		return SignalIngestStatus{}, fmt.Errorf("query baselines: %w", err)
	}
	if err == nil {
		if scanErr := baseRow.Columns(&out.BaselineRowsFromSignal); scanErr != nil {
			return SignalIngestStatus{}, fmt.Errorf("scan baselines: %w", scanErr)
		}
	}

	return out, nil
}
