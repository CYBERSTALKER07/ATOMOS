package planning

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
)

// ErrScenarioNotFound is returned when a scenario id is unknown for the supplier.
var ErrScenarioNotFound = errors.New("scenario_not_found")

// ErrScenarioPublishConflict is returned on CAS failure (not DRAFT or raced).
var ErrScenarioPublishConflict = errors.New("scenario_publish_conflict")

// ScenarioCompareRequest selects two scenarios for side-by-side deltas.
type ScenarioCompareRequest struct {
	ScenarioIDs []string `json:"scenario_ids"`
}

// ScenarioCompareDeltas is right − left for numeric metrics.
type ScenarioCompareDeltas struct {
	SLARiskPctDelta         float64 `json:"sla_risk_pct_delta"`
	FleetVolumeDelta        int64   `json:"fleet_volume_orders_delta"`
	RevenueAtRiskMinorDelta int64   `json:"revenue_at_risk_minor_delta"`
	StockoutCountDelta      int     `json:"stockout_count_delta"`
	CapacityBreachChanged   bool    `json:"capacity_breach_changed"`
}

// ScenarioCompareResult is a side-by-side compare payload.
type ScenarioCompareResult struct {
	Left   ScenarioResult       `json:"left"`
	Right  ScenarioResult       `json:"right"`
	Deltas ScenarioCompareDeltas `json:"deltas"`
}

// ScenarioCloneInput optionally mutates shocks when cloning.
type ScenarioCloneInput struct {
	FactoryDowntimeHours *int     `json:"factory_downtime_hours,omitempty"`
	DemandDeltaPct       *float64 `json:"demand_delta_pct,omitempty"`
	HorizonDays          *int     `json:"horizon_days,omitempty"`
	Label                string   `json:"label,omitempty"`
}

type scenarioMetricsBlob struct {
	SLARiskPct         float64  `json:"sla_risk_pct"`
	BaselineSLARiskPct float64  `json:"baseline_sla_risk_pct,omitempty"`
	FleetVolume        int64    `json:"fleet_volume_orders"`
	StockoutSKUs       []string `json:"stockout_skus"`
	CapacityBreach     bool     `json:"capacity_breach"`
	RevenueAtRiskMinor int64    `json:"revenue_at_risk_minor,omitempty"`
	UnitValueSource    string   `json:"unit_value_source,omitempty"`
	Mode               string   `json:"mode,omitempty"`
}

func (s *Service) persistScenarioDraft(ctx context.Context, result ScenarioResult, snapshotAt *time.Time) error {
	if s == nil || s.Spanner == nil {
		return errors.New("planning unavailable")
	}
	blob, err := json.Marshal(scenarioMetricsBlob{
		SLARiskPct:         result.SLARiskPct,
		BaselineSLARiskPct: result.BaselineSLARiskPct,
		FleetVolume:        result.FleetVolume,
		StockoutSKUs:       result.StockoutSKUs,
		CapacityBreach:     result.CapacityBreach,
		RevenueAtRiskMinor: result.RevenueAtRiskMinor,
		UnitValueSource:    result.UnitValueSource,
		Mode:               result.Mode,
	})
	if err != nil {
		return err
	}
	row := map[string]any{
		"SupplierId":           result.SupplierID,
		"ScenarioId":           result.ScenarioID,
		"Version":              result.Version,
		"Status":               result.Status,
		"ParentScenarioId":     nilString(result.ParentScenarioID),
		"Label":                nilString(result.Label),
		"HorizonDays":          int64(result.HorizonDays),
		"FactoryDowntimeHours": int64(result.FactoryDowntimeHours),
		"DemandDeltaPct":       result.DemandDeltaPct,
		"ResultJSON":           string(blob),
		"Mode":                 nilString(result.Mode),
		"UnitValueSource":      nilString(result.UnitValueSource),
		"CreatedBy":            nilString(result.CreatedBy),
		"CreatedAt":            spanner.CommitTimestamp,
		"UpdatedAt":            spanner.CommitTimestamp,
	}
	if snapshotAt != nil && !snapshotAt.IsZero() {
		row["SnapshotCapturedAt"] = snapshotAt.UTC()
	}
	_, err = s.Spanner.Apply(ctx, []*spanner.Mutation{spanner.InsertOrUpdateMap("PlanningScenarios", row)})
	return err
}

func nilString(v string) any {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return v
}

func nullStr(v spanner.NullString) string {
	if v.Valid {
		return v.StringVal
	}
	return ""
}

// ListScenarios returns recent scenarios for a supplier (newest first).
func (s *Service) ListScenarios(ctx context.Context, supplierID string, limit int) ([]ScenarioResult, error) {
	if s == nil || s.Spanner == nil {
		return nil, errors.New("planning unavailable")
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	iter := s.Spanner.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT ScenarioId, Version, Status, COALESCE(ParentScenarioId,''), COALESCE(Label,''),
		             HorizonDays, FactoryDowntimeHours, DemandDeltaPct, ResultJSON,
		             COALESCE(Mode,''), COALESCE(UnitValueSource,''),
		             COALESCE(CreatedBy,''), COALESCE(PublishedBy,''), PublishedAt, UpdatedAt
		      FROM PlanningScenarios
		      WHERE SupplierId = @sid
		      ORDER BY UpdatedAt DESC
		      LIMIT @lim`,
		Params: map[string]any{"sid": supplierID, "lim": int64(limit)},
	})
	defer iter.Stop()
	var out []ScenarioResult
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		r, err := decodeScenarioRow(supplierID, row)
		if err != nil {
			continue
		}
		out = append(out, r)
	}
	return out, nil
}

// GetScenario loads one scenario by id.
func (s *Service) GetScenario(ctx context.Context, supplierID, scenarioID string) (ScenarioResult, error) {
	if s == nil || s.Spanner == nil {
		return ScenarioResult{}, errors.New("planning unavailable")
	}
	iter := s.Spanner.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT ScenarioId, Version, Status, COALESCE(ParentScenarioId,''), COALESCE(Label,''),
		             HorizonDays, FactoryDowntimeHours, DemandDeltaPct, ResultJSON,
		             COALESCE(Mode,''), COALESCE(UnitValueSource,''),
		             COALESCE(CreatedBy,''), COALESCE(PublishedBy,''), PublishedAt, UpdatedAt
		      FROM PlanningScenarios
		      WHERE SupplierId = @sid AND ScenarioId = @id
		      LIMIT 1`,
		Params: map[string]any{"sid": supplierID, "id": scenarioID},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if errors.Is(err, iterator.Done) {
		return ScenarioResult{}, ErrScenarioNotFound
	}
	if err != nil {
		return ScenarioResult{}, err
	}
	return decodeScenarioRow(supplierID, row)
}

func decodeScenarioRow(supplierID string, row *spanner.Row) (ScenarioResult, error) {
	var (
		id, status, parent, label, mode, unitSrc, createdBy, publishedBy string
		version, horizon, downtime                                       int64
		demand                                                           float64
		resultJSON                                                       string
		publishedAt                                                      spanner.NullTime
		updatedAt                                                        time.Time
	)
	if err := row.Columns(
		&id, &version, &status, &parent, &label,
		&horizon, &downtime, &demand, &resultJSON,
		&mode, &unitSrc, &createdBy, &publishedBy, &publishedAt, &updatedAt,
	); err != nil {
		return ScenarioResult{}, err
	}
	var blob scenarioMetricsBlob
	_ = json.Unmarshal([]byte(resultJSON), &blob)
	out := ScenarioResult{
		ScenarioID:           id,
		SupplierID:           supplierID,
		Version:              version,
		Status:               status,
		ParentScenarioID:     parent,
		Label:                label,
		HorizonDays:          int(horizon),
		FactoryDowntimeHours: int(downtime),
		DemandDeltaPct:       demand,
		SLARiskPct:           blob.SLARiskPct,
		BaselineSLARiskPct:   blob.BaselineSLARiskPct,
		FleetVolume:          blob.FleetVolume,
		StockoutSKUs:         blob.StockoutSKUs,
		CapacityBreach:       blob.CapacityBreach,
		RevenueAtRiskMinor:   blob.RevenueAtRiskMinor,
		UnitValueSource:      firstNonEmpty(blob.UnitValueSource, unitSrc),
		Mode:                 firstNonEmpty(blob.Mode, mode),
		CreatedBy:            createdBy,
		PublishedBy:          publishedBy,
		UpdatedAt:            updatedAt.UTC().Format(time.RFC3339Nano),
	}
	if out.StockoutSKUs == nil {
		out.StockoutSKUs = []string{}
	}
	if publishedAt.Valid {
		out.PublishedAt = publishedAt.Time.UTC().Format(time.RFC3339Nano)
	}
	return out, nil
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

// CloneScenario copies a parent scenario into a new DRAFT, optionally mutating shocks.
func (s *Service) CloneScenario(ctx context.Context, supplierID, parentID, createdBy string, in ScenarioCloneInput) (ScenarioResult, error) {
	parent, err := s.GetScenario(ctx, supplierID, parentID)
	if err != nil {
		return ScenarioResult{}, err
	}
	input := ScenarioInput{
		FactoryDowntimeHours: parent.FactoryDowntimeHours,
		DemandDeltaPct:       parent.DemandDeltaPct,
		HorizonDays:          parent.HorizonDays,
		Label:                strings.TrimSpace(in.Label),
	}
	if in.FactoryDowntimeHours != nil {
		input.FactoryDowntimeHours = *in.FactoryDowntimeHours
	}
	if in.DemandDeltaPct != nil {
		input.DemandDeltaPct = *in.DemandDeltaPct
	}
	if in.HorizonDays != nil && *in.HorizonDays > 0 {
		input.HorizonDays = *in.HorizonDays
	}
	if input.Label == "" {
		short := parent.ScenarioID
		if len(short) > 8 {
			short = short[:8]
		}
		input.Label = "Clone of " + short
	}

	computed, err := s.computeScenario(ctx, supplierID, input)
	if err != nil {
		return ScenarioResult{}, err
	}
	result := computed
	result.ScenarioID = uuid.NewString()
	result.SupplierID = supplierID
	result.Version = 1
	result.Status = ScenarioStatusDraft
	result.ParentScenarioID = parent.ScenarioID
	result.Label = input.Label
	result.HorizonDays = input.HorizonDays
	result.FactoryDowntimeHours = input.FactoryDowntimeHours
	result.DemandDeltaPct = input.DemandDeltaPct
	result.CreatedBy = strings.TrimSpace(createdBy)
	result.UpdatedAt = s.Now().Format(time.RFC3339Nano)

	if err := s.persistScenarioDraft(ctx, result, nil); err != nil {
		return ScenarioResult{}, err
	}
	return result, nil
}

// CompareScenarios returns side-by-side metrics for exactly two scenario ids.
func (s *Service) CompareScenarios(ctx context.Context, supplierID string, ids []string) (ScenarioCompareResult, error) {
	if len(ids) != 2 {
		return ScenarioCompareResult{}, errors.New("compare_requires_two_scenarios")
	}
	left, err := s.GetScenario(ctx, supplierID, strings.TrimSpace(ids[0]))
	if err != nil {
		return ScenarioCompareResult{}, err
	}
	right, err := s.GetScenario(ctx, supplierID, strings.TrimSpace(ids[1]))
	if err != nil {
		return ScenarioCompareResult{}, err
	}
	return ScenarioCompareResult{
		Left:  left,
		Right: right,
		Deltas: ScenarioCompareDeltas{
			SLARiskPctDelta:         right.SLARiskPct - left.SLARiskPct,
			FleetVolumeDelta:        right.FleetVolume - left.FleetVolume,
			RevenueAtRiskMinorDelta: right.RevenueAtRiskMinor - left.RevenueAtRiskMinor,
			StockoutCountDelta:      len(right.StockoutSKUs) - len(left.StockoutSKUs),
			CapacityBreachChanged:   right.CapacityBreach != left.CapacityBreach,
		},
	}, nil
}

// PublishScenario CAS-publishes a DRAFT as the supplier planning baseline.
// Prior PUBLISHED rows become SUPERSEDED. Does not mutate inventory or sealed manifests.
func (s *Service) PublishScenario(ctx context.Context, supplierID, scenarioID, actor string) (ScenarioResult, error) {
	if s == nil || s.Spanner == nil {
		return ScenarioResult{}, errors.New("planning unavailable")
	}
	scenarioID = strings.TrimSpace(scenarioID)
	actor = strings.TrimSpace(actor)
	if scenarioID == "" {
		return ScenarioResult{}, ErrScenarioNotFound
	}

	var published ScenarioResult
	_, err := s.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "PlanningScenarios", spanner.Key{supplierID, scenarioID}, []string{
			"ScenarioId", "Version", "Status", "ParentScenarioId", "Label",
			"HorizonDays", "FactoryDowntimeHours", "DemandDeltaPct", "ResultJSON",
			"Mode", "UnitValueSource", "CreatedBy",
		})
		if err != nil {
			if errors.Is(err, spanner.ErrRowNotFound) {
				return ErrScenarioNotFound
			}
			return err
		}
		var (
			id, status                                   string
			version, horizon, downtime                   int64
			demand                                       float64
			resultJSON                                   string
			parentNS, labelNS, modeNS, unitNS, createdNS spanner.NullString
		)
		if err := row.Columns(
			&id, &version, &status, &parentNS, &labelNS,
			&horizon, &downtime, &demand, &resultJSON,
			&modeNS, &unitNS, &createdNS,
		); err != nil {
			return err
		}
		parent := nullStr(parentNS)
		label := nullStr(labelNS)
		mode := nullStr(modeNS)
		unitSrc := nullStr(unitNS)
		createdBy := nullStr(createdNS)
		if status != ScenarioStatusDraft {
			return ErrScenarioPublishConflict
		}

		pubIter := txn.Query(ctx, spanner.Statement{
			SQL: `SELECT ScenarioId, Version FROM PlanningScenarios
			      WHERE SupplierId = @sid AND Status = @st`,
			Params: map[string]any{"sid": supplierID, "st": ScenarioStatusPublished},
		})
		defer pubIter.Stop()
		var mutations []*spanner.Mutation
		maxVersion := version
		for {
			prow, perr := pubIter.Next()
			if errors.Is(perr, iterator.Done) {
				break
			}
			if perr != nil {
				return perr
			}
			var pubID string
			var pubVer int64
			if err := prow.Columns(&pubID, &pubVer); err != nil {
				return err
			}
			if pubVer > maxVersion {
				maxVersion = pubVer
			}
			mutations = append(mutations, spanner.UpdateMap("PlanningScenarios", map[string]any{
				"SupplierId": supplierID,
				"ScenarioId": pubID,
				"Status":     ScenarioStatusSuperseded,
				"UpdatedAt":  spanner.CommitTimestamp,
			}))
		}

		newVersion := maxVersion + 1
		publishedAt := s.Now().UTC()
		mutations = append(mutations, spanner.UpdateMap("PlanningScenarios", map[string]any{
			"SupplierId":  supplierID,
			"ScenarioId":  scenarioID,
			"Status":      ScenarioStatusPublished,
			"Version":     newVersion,
			"PublishedBy": actor,
			"PublishedAt": publishedAt,
			"UpdatedAt":   spanner.CommitTimestamp,
		}))

		var blob scenarioMetricsBlob
		_ = json.Unmarshal([]byte(resultJSON), &blob)
		published = ScenarioResult{
			ScenarioID:           id,
			SupplierID:           supplierID,
			Version:              newVersion,
			Status:               ScenarioStatusPublished,
			ParentScenarioID:     parent,
			Label:                label,
			HorizonDays:          int(horizon),
			FactoryDowntimeHours: int(downtime),
			DemandDeltaPct:       demand,
			SLARiskPct:           blob.SLARiskPct,
			BaselineSLARiskPct:   blob.BaselineSLARiskPct,
			FleetVolume:          blob.FleetVolume,
			StockoutSKUs:         blob.StockoutSKUs,
			CapacityBreach:       blob.CapacityBreach,
			RevenueAtRiskMinor:   blob.RevenueAtRiskMinor,
			UnitValueSource:      firstNonEmpty(blob.UnitValueSource, unitSrc),
			Mode:                 firstNonEmpty(blob.Mode, mode),
			CreatedBy:            createdBy,
			PublishedBy:          actor,
			PublishedAt:          publishedAt.Format(time.RFC3339Nano),
			UpdatedAt:            publishedAt.Format(time.RFC3339Nano),
		}
		if published.StockoutSKUs == nil {
			published.StockoutSKUs = []string{}
		}

		payload := events.PlanningEvent{
			BaseEvent: events.BaseEvent{
				Type:      events.EventPlanningScenarioPublished,
				Timestamp: publishedAt.Format(time.RFC3339Nano),
			},
			SupplierID:  supplierID,
			ScenarioID:  scenarioID,
			Version:     newVersion,
			PublishedBy: actor,
			Action:      "PUBLISH",
		}
		buf := &planningTxnBuffer{}
		if emitErr := outbox.EmitJSON(ctx, buf, events.AggregatePlanning, supplierID, events.TopicMain, payload); emitErr != nil {
			return emitErr
		}
		mutations = append(mutations, planningOutboxMutations(buf.events)...)
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		return ScenarioResult{}, err
	}
	return published, nil
}
