package analytics

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"google.golang.org/api/iterator"
)

type Handlers struct {
	Client *spanner.Client
	// SupplierID returns request tenant scope (PreferTenantSupplierID). Empty → fail-closed empty list.
	SupplierID func(ctx context.Context) string
}

type routePerfWire struct {
	RouteID            string `json:"route_id"`
	SupplierID         string `json:"supplier_id,omitempty"`
	DriverID           string `json:"driver_id,omitempty"`
	PlannedStops       int64  `json:"planned_stops,omitempty"`
	ActualStops        int64  `json:"actual_stops,omitempty"`
	PlannedDurationSec int64  `json:"planned_duration_sec,omitempty"`
	ActualDurationSec  int64  `json:"actual_duration_sec,omitempty"`
	ReplanCount        int64  `json:"replan_count,omitempty"`
	ComputedAt         string `json:"computed_at"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// routePerfListStmt builds a tenant-scoped list query. Empty supplierId → WHERE FALSE (fail-closed).
func routePerfListStmt(supplierID string, limit int) spanner.Statement {
	if limit <= 0 {
		limit = 50
	}
	sid := strings.TrimSpace(supplierID)
	if sid == "" {
		return spanner.Statement{
			SQL: `SELECT RouteId, SupplierId, DriverId, PlannedStops, ActualStops, PlannedDurationSec, ActualDurationSec, ReplanCount, ComputedAt
			      FROM RoutePerformanceAnalytics WHERE FALSE LIMIT @limit`,
			Params: map[string]any{"limit": limit},
		}
	}
	return spanner.Statement{
		SQL: `SELECT RouteId, SupplierId, DriverId, PlannedStops, ActualStops, PlannedDurationSec, ActualDurationSec, ReplanCount, ComputedAt
		      FROM RoutePerformanceAnalytics
		      WHERE SupplierId = @supplierId
		      ORDER BY ComputedAt DESC LIMIT @limit`,
		Params: map[string]any{"supplierId": sid, "limit": limit},
	}
}

func (h *Handlers) HandleListRoutePerformance(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok || (claims.Role != auth.RoleAdmin && claims.Role != auth.RolePlatformAdmin) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	if h == nil || h.Client == nil {
		writeJSON(w, http.StatusOK, map[string]any{"routes": []routePerfWire{}})
		return
	}
	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	supplierID := ""
	if h.SupplierID != nil {
		supplierID = h.SupplierID(r.Context())
	}
	iter := h.Client.Single().Query(r.Context(), routePerfListStmt(supplierID, limit))
	defer iter.Stop()

	rows := make([]routePerfWire, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list_failed"})
			return
		}
		var routeID string
		var supplier, driver spanner.NullString
		var plannedStops, actualStops, plannedDur, actualDur, replan spanner.NullInt64
		var computedAt time.Time
		if err := row.Columns(&routeID, &supplier, &driver, &plannedStops, &actualStops, &plannedDur, &actualDur, &replan, &computedAt); err != nil {
			continue
		}
		wire := routePerfWire{
			RouteID:    routeID,
			ComputedAt: computedAt.UTC().Format(time.RFC3339Nano),
		}
		if supplier.Valid {
			wire.SupplierID = supplier.StringVal
		}
		if driver.Valid {
			wire.DriverID = driver.StringVal
		}
		if plannedStops.Valid {
			wire.PlannedStops = plannedStops.Int64
		}
		if actualStops.Valid {
			wire.ActualStops = actualStops.Int64
		}
		if plannedDur.Valid {
			wire.PlannedDurationSec = plannedDur.Int64
		}
		if actualDur.Valid {
			wire.ActualDurationSec = actualDur.Int64
		}
		if replan.Valid {
			wire.ReplanCount = replan.Int64
		}
		rows = append(rows, wire)
	}
	writeJSON(w, http.StatusOK, map[string]any{"routes": rows})
}
