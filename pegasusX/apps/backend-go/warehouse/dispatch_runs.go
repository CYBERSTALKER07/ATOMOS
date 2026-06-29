package warehouse

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/go-chi/chi/v5"
	"google.golang.org/api/iterator"
)

// DispatchRunRow is one persisted warehouse dispatch commit audit row.
type DispatchRunRow struct {
	RunID          string   `json:"run_id"`
	WarehouseID    string   `json:"warehouse_id"`
	SupplierID     string   `json:"supplier_id"`
	ActorID        string   `json:"actor_id,omitempty"`
	Mode           string   `json:"mode"`
	Status         string   `json:"status"`
	ManifestCount  int64    `json:"manifest_count"`
	OrdersAssigned int64    `json:"orders_assigned"`
	Warnings       []string `json:"warnings,omitempty"`
	ManifestIDs    []string `json:"manifest_ids,omitempty"`
	CreatedAt      string   `json:"created_at"`
}

func (s *Service) persistDispatchRun(ctx context.Context, result DispatchExecuteResult, mode, actorID string) {
	if s.spannerClient == nil {
		return
	}
	runID := uuid.NewString()
	manifestIDs := make([]string, 0, len(result.Manifests))
	for _, m := range result.Manifests {
		if id := strings.TrimSpace(m.ManifestID); id != "" {
			manifestIDs = append(manifestIDs, id)
		}
	}
	warningsRaw, _ := json.Marshal(result.Warnings)
	manifestsRaw, _ := json.Marshal(manifestIDs)
	_, _ = s.spannerClient.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("DispatchRuns", map[string]any{
			"RunId":          runID,
			"WarehouseId":    result.WarehouseID,
			"SupplierId":     result.SupplierID,
			"ActorId":        strings.TrimSpace(actorID),
			"Mode":           strings.TrimSpace(mode),
			"Status":         strings.TrimSpace(result.Status),
			"ManifestCount":  int64(result.ManifestsCreated),
			"OrdersAssigned": int64(result.OrdersAssigned),
			"WarningsJson":   warningsRaw,
			"ManifestsJson":  manifestsRaw,
			"CreatedAt":      spanner.CommitTimestamp,
		}),
	})
}

// HandleDispatchRuns serves GET /v1/warehouse/ops/dispatch/runs.
func (s *Service) HandleDispatchRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	whID := warehouseIDFromRequest(r)
	if s.spannerClient == nil {
		writeJSON(w, http.StatusOK, map[string]any{"runs": []DispatchRunRow{}})
		return
	}
	rows, err := s.listDispatchRuns(r.Context(), whID, 50)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "dispatch_runs_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"runs": rows})
}

// HandleDispatchRunDetail serves GET /v1/warehouse/ops/dispatch/runs/{runID}.
func (s *Service) HandleDispatchRunDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	runID := strings.TrimSpace(chi.URLParam(r, "runID"))
	if runID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "run_id_required"})
		return
	}
	if s.spannerClient == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "run_not_found"})
		return
	}
	row, ok, err := s.getDispatchRun(r.Context(), runID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "dispatch_run_failed"})
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "run_not_found"})
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func (s *Service) listDispatchRuns(ctx context.Context, warehouseID string, limit int) ([]DispatchRunRow, error) {
	stmt := spanner.Statement{
		SQL: `SELECT RunId, WarehouseId, SupplierId, ActorId, Mode, Status, ManifestCount, OrdersAssigned, WarningsJson, ManifestsJson, CreatedAt
		      FROM DispatchRuns
		      WHERE WarehouseId = @wh
		      ORDER BY CreatedAt DESC
		      LIMIT @lim`,
		Params: map[string]any{"wh": warehouseID, "lim": int64(limit)},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	out := make([]DispatchRunRow, 0, limit)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		parsed, err := scanDispatchRun(row)
		if err != nil {
			return nil, err
		}
		out = append(out, parsed)
	}
}

func (s *Service) getDispatchRun(ctx context.Context, runID string) (DispatchRunRow, bool, error) {
	stmt := spanner.Statement{
		SQL: `SELECT RunId, WarehouseId, SupplierId, ActorId, Mode, Status, ManifestCount, OrdersAssigned, WarningsJson, ManifestsJson, CreatedAt
		      FROM DispatchRuns
		      WHERE RunId = @id
		      LIMIT 1`,
		Params: map[string]any{"id": runID},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return DispatchRunRow{}, false, nil
	}
	if err != nil {
		return DispatchRunRow{}, false, err
	}
	parsed, err := scanDispatchRun(row)
	if err != nil {
		return DispatchRunRow{}, false, err
	}
	return parsed, true, nil
}

func scanDispatchRun(row *spanner.Row) (DispatchRunRow, error) {
	var out DispatchRunRow
	var actorID spanner.NullString
	var warningsRaw, manifestsRaw []byte
	var createdAt time.Time
	if err := row.Columns(&out.RunID, &out.WarehouseID, &out.SupplierID, &actorID, &out.Mode, &out.Status,
		&out.ManifestCount, &out.OrdersAssigned, &warningsRaw, &manifestsRaw, &createdAt); err != nil {
		return DispatchRunRow{}, fmt.Errorf("scan dispatch run: %w", err)
	}
	out.ActorID = actorID.StringVal
	out.CreatedAt = createdAt.UTC().Format(time.RFC3339Nano)
	_ = json.Unmarshal(warningsRaw, &out.Warnings)
	_ = json.Unmarshal(manifestsRaw, &out.ManifestIDs)
	return out, nil
}
