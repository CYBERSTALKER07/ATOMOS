package proximity

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"backend-go/auth"
	"backend-go/spannerx"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

const defaultDispatchAuditLimit = 50

type dispatchAuditFilter struct {
	auditType  string
	unresolved bool
	limit      int64
}

// DispatchAuditEntry is a single supplier dispatch-coverage audit row.
type DispatchAuditEntry struct {
	AuditID      string     `json:"audit_id"`
	RetailerID   string     `json:"retailer_id"`
	RetailerCell string     `json:"retailer_cell"`
	AuditType    string     `json:"audit_type"`
	WarehouseID  string     `json:"warehouse_id,omitempty"`
	DistanceKm   *float64   `json:"distance_km,omitempty"`
	ResolvedAt   *time.Time `json:"resolved_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// DispatchAuditListResponse is the supplier-facing dispatch audit feed.
type DispatchAuditListResponse struct {
	Audits []DispatchAuditEntry `json:"audits"`
	Count  int                  `json:"count"`
}

// HandleDispatchAudits serves the supplier dispatch-audit feed used by the
// supplier dashboard coverage-alerts cell.
func HandleDispatchAudits(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims.UserID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		filter, err := parseDispatchAuditFilter(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		supplierID := claims.ResolveSupplierID()
		traceID := requestTraceID(r)
		audits, err := listDispatchAudits(r.Context(), spannerClient, supplierID, filter)
		if err != nil {
			slog.ErrorContext(r.Context(), "dispatch audits list failed",
				"trace_id", traceID,
				"supplier_id", supplierID,
				"audit_type", filter.auditType,
				"unresolved", filter.unresolved,
				"limit", filter.limit,
				"error", err.Error(),
			)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(DispatchAuditListResponse{
			Audits: audits,
			Count:  len(audits),
		})
	}
}

func requestTraceID(r *http.Request) string {
	if traceID := r.Header.Get("X-Trace-Id"); traceID != "" {
		return traceID
	}
	return r.Header.Get("X-Request-Id")
}

func parseDispatchAuditFilter(r *http.Request) (dispatchAuditFilter, error) {
	filter := dispatchAuditFilter{limit: defaultDispatchAuditLimit}

	if rawType := strings.TrimSpace(r.URL.Query().Get("type")); rawType != "" {
		switch rawType {
		case "ORPHAN_DETECTED", "COVERAGE_RESTORED", "COVERAGE_GAP":
			filter.auditType = rawType
		default:
			return filter, fmt.Errorf("invalid type")
		}
	}

	if rawUnresolved := strings.TrimSpace(r.URL.Query().Get("unresolved")); rawUnresolved != "" {
		value, err := strconv.ParseBool(rawUnresolved)
		if err != nil {
			return filter, fmt.Errorf("invalid unresolved")
		}
		filter.unresolved = value
	}

	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		value, err := strconv.Atoi(rawLimit)
		if err != nil || value <= 0 || value > 100 {
			return filter, fmt.Errorf("invalid limit")
		}
		filter.limit = int64(value)
	}

	return filter, nil
}

func listDispatchAudits(ctx context.Context, client *spanner.Client, supplierID string, filter dispatchAuditFilter) ([]DispatchAuditEntry, error) {
	var query strings.Builder
	query.WriteString(`SELECT AuditId, RetailerId, RetailerCell, AuditType, WarehouseId, DistanceKm, ResolvedAt, CreatedAt
		FROM DispatchAudit@{FORCE_INDEX=Idx_DispatchAudit_BySupplierId}
		WHERE SupplierId = @supplier_id`)
	params := map[string]interface{}{
		"supplier_id": supplierID,
		"limit":       filter.limit,
	}

	if filter.auditType != "" {
		query.WriteString(" AND AuditType = @audit_type")
		params["audit_type"] = filter.auditType
	}
	if filter.unresolved {
		query.WriteString(" AND ResolvedAt IS NULL")
	}
	query.WriteString(" ORDER BY CreatedAt DESC LIMIT @limit")

	stmt := spanner.Statement{
		SQL:    query.String(),
		Params: params,
	}

	iter := spannerx.StaleQuery(ctx, client, stmt)
	defer iter.Stop()

	var audits []DispatchAuditEntry
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return audits, nil
		}
		if err != nil {
			return nil, fmt.Errorf("query dispatch audits: %w", err)
		}

		var audit DispatchAuditEntry
		var warehouseID spanner.NullString
		var distanceKm spanner.NullFloat64
		var resolvedAt spanner.NullTime
		if err := row.Columns(
			&audit.AuditID,
			&audit.RetailerID,
			&audit.RetailerCell,
			&audit.AuditType,
			&warehouseID,
			&distanceKm,
			&resolvedAt,
			&audit.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan dispatch audit row: %w", err)
		}

		if warehouseID.Valid {
			audit.WarehouseID = warehouseID.StringVal
		}
		if distanceKm.Valid {
			value := distanceKm.Float64
			audit.DistanceKm = &value
		}
		if resolvedAt.Valid {
			value := resolvedAt.Time
			audit.ResolvedAt = &value
		}

		audits = append(audits, audit)
	}
}
